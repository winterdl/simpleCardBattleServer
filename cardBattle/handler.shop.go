package cardBattle

import (
	"io"
	"sync"
	"time"
)

type CardShop struct {
	Cards         map[string]*Card
	TotalCard     int
	MinLevel      int
	MaxLevel      int
	Time          int
	MaxItem       int
	Broadcast     chan ShopStream
	ClientStreams map[string]chan ShopStream
	streamsMtx    sync.RWMutex
}

func (c *CardBattleServer) CardBattleShopStream(stream CardBattleService_CardBattleShopStreamServer) error {


	for {

		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		switch evt := msg.Event.(type) {

		case *ShopStream_PlayerJoin:

			go c.Shop.receiveBroadcasts(stream,evt.PlayerJoin.Id)
			
			err := stream.Send(&ShopStream{
				Event: evt,
			})

			if err != nil {
				return err
			}


		case *ShopStream_ShopRefreshTime:

			// just let this empty

		case *ShopStream_ShopRefresh:

			// just let this empty

		case *ShopStream_OnePlayerWithCards:

			// check player if exist
			if player, isExist := c.Players[evt.OnePlayerWithCards.Owner.Id]; isExist {

				// send back to client
				err := stream.Send(&ShopStream{
					Event: &ShopStream_OnePlayerWithCards{
						OnePlayerWithCards: player,
					},
				})

				if err != nil {
					return err
				}
			}

		case *ShopStream_AllCardInShopping:

			// query all card in shop
			cards := []*Card{}
			for _, card := range c.Shop.Cards {
				cards = append(cards, card)
			}

			// send back to client
			err := stream.Send(&ShopStream{
				Event: &ShopStream_AllCardInShopping{
					AllCardInShopping: &AllCard{
						Cards: cards,
					},
				},
			})

			if err != nil {
				return err
			}

		case *ShopStream_OnBuyCard:

			successBuy := false

			// check is card exist in shop item
			if card, isExist := c.Shop.Cards[evt.OnBuyCard.CardData.Id]; isExist {

				// check if player hass enought cash
				// and slot
				p := c.Players[evt.OnBuyCard.Client.Id]
				if p.Owner.Cash >= card.Price && card.isObtainable(p.Owner) && p.Owner.MaxReserveSlot > int32(len(p.Reserve)) {

					c.Players[evt.OnBuyCard.Client.Id].Reserve = append(c.Players[evt.OnBuyCard.Client.Id].Reserve, &Card{
						Id:    card.Id,
						Image: card.Image,
						Price: card.Price,
						Level: card.Level,
						Atk:   card.Atk,
						Def:   card.Def,
						Color: card.Color,
						Name:  card.Name,
					})

					delete(c.Shop.Cards, card.Id)

					c.Players[evt.OnBuyCard.Client.Id].Owner.Cash -= card.Price

					successBuy = true

					// broadcast to all player
					// to update shop item
					c.Shop.Broadcast <- ShopStream{
						Event: &ShopStream_ShopRefresh{
							ShopRefresh: true,
						},
					}

					time.Sleep(1 * time.Second)
				}
			}

			// send result to client
			// card hass been bought
			// and hass been added to player
			// reserve deck
			err := stream.Send(&ShopStream{
				Event: &ShopStream_OnCardBought{
					OnCardBought: successBuy,
				},
			})

			if err != nil {
				return err
			}

		case *ShopStream_OnCardBought:

			// just let this empty

		case *ShopStream_OnSellCard:

			successSell := false
			_, isExist := c.Players[evt.OnSellCard.Client.Id]

			// check is card exist in player reserve
			if isExist {

				card := &Card{}
				for _, c := range c.Players[evt.OnSellCard.Client.Id].Reserve {
					if c.Id == evt.OnSellCard.CardData.Id {
						card = c
						break
					}
				}

				c.Shop.Cards[evt.OnSellCard.CardData.Id] = &Card{
					Id:    card.Id,
					Image: card.Image,
					Price: card.Price,
					Level: card.Level,
					Atk:   card.Atk,
					Def:   card.Def,
					Color: card.Color,
					Name:  card.Name,
				}

				newDeck := []*Card{}
				for _, cardReserve := range c.Players[evt.OnSellCard.Client.Id].Reserve {
					if cardReserve.Id != card.Id {
						newDeck = append(newDeck, cardReserve)
					}
				}

				c.Players[evt.OnSellCard.Client.Id].Reserve = newDeck

				c.Players[evt.OnSellCard.Client.Id].Owner.Cash += card.Price

				successSell = true

			}

			// send result to client
			// card hass been sold
			// and hass been removefrom player
			// reserve deck
			err := stream.Send(&ShopStream{
				Event: &ShopStream_OnCardSold{
					OnCardSold: successSell,
				},
			})

			if err != nil {
				return err
			}

		case *ShopStream_OnCardSold:

			// just let this empty

		case *ShopStream_OnCardDeckSlot:

			success := false
			if player, isExist := c.Players[evt.OnCardDeckSlot.Owner.Id]; isExist {

				switch evt.OnCardDeckSlot.SlotType {
				case 0:

					deckPrice := int64(int32(120) * player.Owner.MaxDeckSlot)
					if player.Owner.Cash >= deckPrice {
						c.Players[evt.OnCardDeckSlot.Owner.Id].Owner.MaxDeckSlot++
						c.Players[evt.OnCardDeckSlot.Owner.Id].Owner.Cash -= deckPrice
						success = true
					}

				case 1:

					deckPrice := int64(int32(100) * player.Owner.MaxReserveSlot)
					if player.Owner.Cash >= deckPrice {
						c.Players[evt.OnCardDeckSlot.Owner.Id].Owner.MaxReserveSlot++
						c.Players[evt.OnCardDeckSlot.Owner.Id].Owner.Cash -= deckPrice
						success = true
					}

				default:
				}

			}

			// send back to client
			err := stream.Send(&ShopStream{
				Event: &ShopStream_OnSuccessAddSlot{
					OnSuccessAddSlot: success,
				},
			})

			if err != nil {
				return err
			}

		case *ShopStream_OnSuccessAddSlot:

			// just let this empty

		case *ShopStream_ExcMessage:

			// just let this empty

		default:

		}
	}
}
