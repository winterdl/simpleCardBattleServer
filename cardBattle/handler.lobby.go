package cardBattle

import (
	"io"
	"log"
	"time"
)

func (c *CardBattleServer) CardBattleLobbyStream(stream CardBattleService_CardBattleLobbyStreamServer) error {

	for {

		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		switch evt := msg.Event.(type) {

		case *LobbyStream_PlayerJoin:

			if player, isExist := c.Players[evt.PlayerJoin.Id]; isExist {

				// receive broadcast
				// for any event happend in lobby
				go c.Lobby.receiveBroadcasts(stream, player.Owner.Id)

				// broadcast to lobby
				// new player hass join
				c.Lobby.Broadcast <- LobbyStream{
					Event: &LobbyStream_PlayerJoin{
						PlayerJoin: player.Owner,
					},
				}

				time.Sleep(1 * time.Second)

				err := stream.Send(&LobbyStream{
					Event: &LobbyStream_PlayerSuccessJoin{
						PlayerSuccessJoin: player.Owner,
					},
				})

				if err != nil {
					return err
				}
			}

		case *LobbyStream_PlayerLeft:

			// broadcast to lobby
			// player hass left
			c.Lobby.Broadcast <- LobbyStream{
				Event: evt,
			}

			c.streamsMtx.RLock()

			// delete player
			delete(c.Players, evt.PlayerLeft.Id)
			// close again his connection
			c.Lobby.closeStream(evt.PlayerLeft.Id)

			c.streamsMtx.RUnlock()

			time.Sleep(1 * time.Second)

			// send back to client
			err := stream.Send(&LobbyStream{
				Event: &LobbyStream_PlayerSuccessLeft{
					PlayerSuccessLeft: evt.PlayerLeft,
				},
			})

			if err != nil {
				return err
			}

		case *LobbyStream_CreateRoom:

			// create game match
			// this is for custome match
			// but currently not dev yet

			// broadcast to lobby
			// player hass create room
			c.Lobby.Broadcast <- LobbyStream{
				Event: evt,
			}

		case *LobbyStream_ShopRefreshTime:

			// just let this empty

		case *LobbyStream_ShopRefresh:

			// just let this empty

		case *LobbyStream_GetOneRoom:

			// if room exist
			if room, isExist := c.Room[evt.GetOneRoom.Id]; isExist {

				// send back to client
				err := stream.Send(&LobbyStream{
					Event: &LobbyStream_GetOneRoom{
						GetOneRoom: room.Data,
					},
				})

				if err != nil {
					return err
				}
			}

		case *LobbyStream_GetAllRooms:

			// get all room
			rooms := []*RoomData{}
			for _, room := range c.Room {
				rooms = append(rooms, room.Data)
			}

			// send back to client
			err := stream.Send(&LobbyStream{
				Event: &LobbyStream_GetAllRooms{
					GetAllRooms: &AllRoom{
						Rooms: rooms,
					},
				},
			})

			if err != nil {
				return err
			}

		case *LobbyStream_GetAllPlayers:

			// get all player
			players := []*Player{}
			for _, player := range c.Players {
				players = append(players, player.Owner)
			}

			// send back to client
			err := stream.Send(&LobbyStream{
				Event: &LobbyStream_GetAllPlayers{
					GetAllPlayers: &AllPlayer{
						Players: players,
					},
				},
			})

			if err != nil {
				return err
			}

		case *LobbyStream_GetOneplayer:

			// check player if exist
			if player, isExist := c.Players[evt.GetOneplayer.Id]; isExist {

				// send back to client
				err := stream.Send(&LobbyStream{
					Event: &LobbyStream_GetOneplayer{
						GetOneplayer: player.Owner,
					},
				})

				if err != nil {
					return err
				}
			}

		case *LobbyStream_PlayerSuccessJoin:

			// just left this empty

		case *LobbyStream_PlayerSuccessLeft:

			// just left this empty

		case *LobbyStream_OnePlayerWithCards:

			// check player if exist
			if player, isExist := c.Players[evt.OnePlayerWithCards.Owner.Id]; isExist {

				// send back to client
				err := stream.Send(&LobbyStream{
					Event: &LobbyStream_OnePlayerWithCards{
						OnePlayerWithCards: player,
					},
				})

				if err != nil {
					return err
				}
			}

		case *LobbyStream_AllCardInShopping:

			// query all card in shop
			cards := []*Card{}
			for _, card := range c.Shop.Cards {
				cards = append(cards, card)
			}

			// send back to client
			err := stream.Send(&LobbyStream{
				Event: &LobbyStream_AllCardInShopping{
					AllCardInShopping: &AllCard{
						Cards: cards,
					},
				},
			})

			if err != nil {
				return err
			}

		case *LobbyStream_OnBuyCard:

			successBuy := false

			// check is card exist in shop item
			if card, isExist := c.Shop.Cards[evt.OnBuyCard.CardData.Id]; isExist {

				// check if player hass enought cash
				if c.Players[evt.OnBuyCard.Client.Id].Owner.Cash >= card.Price && c.Shop.Cards[evt.OnBuyCard.CardData.Id].isObtainable(c.Players[evt.OnBuyCard.Client.Id].Owner) {

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
					c.Lobby.Broadcast <- LobbyStream{
						Event: &LobbyStream_ShopRefresh{
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
			err := stream.Send(&LobbyStream{
				Event: &LobbyStream_OnCardBought{
					OnCardBought: successBuy,
				},
			})

			if err != nil {
				return err
			}

		case *LobbyStream_OnCardBought:

			// just let this empty

		case *LobbyStream_OnSellCard:

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
			err := stream.Send(&LobbyStream{
				Event: &LobbyStream_OnCardSold{
					OnCardSold: successSell,
				},
			})

			if err != nil {
				return err
			}

		case *LobbyStream_OnCardSold:

			// just let this empty

		case *LobbyStream_AddCardToDeck:

			if player, isExist := c.Players[evt.AddCardToDeck.Client.Id]; isExist {
				newDeck := []*Card{}
				cardTarget := &Card{}
				for _, card := range player.Reserve {
					if card.Id != evt.AddCardToDeck.CardData.Id {
						newDeck = append(newDeck, card)
					} else {
						cardTarget = card
					}
				}

				player.Reserve = newDeck
				player.Deck = append(player.Deck, cardTarget)

				// send back to client
				err := stream.Send(&LobbyStream{
					Event: evt,
				})

				if err != nil {
					return err
				}
			}

		case *LobbyStream_RemoveCardFromDeck:

			if player, isExist := c.Players[evt.RemoveCardFromDeck.Client.Id]; isExist {
				newDeck := []*Card{}
				cardTarget := &Card{}
				for _, card := range player.Deck {
					if card.Id != evt.RemoveCardFromDeck.CardData.Id {
						newDeck = append(newDeck, card)
					} else {
						cardTarget = card
					}
				}

				player.Deck = newDeck
				player.Reserve = append(player.Reserve, cardTarget)

				// send back to client
				err := stream.Send(&LobbyStream{
					Event: evt,
				})

				if err != nil {
					return err
				}
			}

		case *LobbyStream_OnjoinWaitingRoom:

			player, errCopy := c.Players[evt.OnjoinWaitingRoom.Id].makeCopy()
			if errCopy != nil {
				log.Println(errCopy)
			}

			c.streamsMtx.RLock()
			c.PlayersInWaitingRoom[evt.OnjoinWaitingRoom.Id] = player
			c.streamsMtx.RUnlock()

			err := stream.Send(&LobbyStream{
				Event: evt,
			})

			if err != nil {
				return err
			}

		case *LobbyStream_OnLeftWaitingRoom:

			c.streamsMtx.RLock()
			delete(c.PlayersInWaitingRoom, evt.OnLeftWaitingRoom.Id)
			c.streamsMtx.RUnlock()

			err := stream.Send(&LobbyStream{
				Event: evt,
			})

			if err != nil {
				return err
			}

		case *LobbyStream_OnBattleFound:

			// left this empty

		default:
		}
	}
}
