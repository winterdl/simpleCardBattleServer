package cardBattle

import (
	"io"
	"sync"
	"time"
)

type Lobby struct {
	Broadcast     chan LobbyStream
	ClientStreams map[string]chan LobbyStream
	streamsMtx    sync.RWMutex
}

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

			} else {

				err := stream.Send(PLAYER_NOT_FOUND)
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

			player, isExist := c.Players[evt.GetOneplayer.Id]
			if !isExist {
				err = stream.Send(PLAYER_NOT_FOUND)
				if err != nil {
					return err
				}

			} else {

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

		case *LobbyStream_AddCardToDeck:

			if player, isExist := c.Players[evt.AddCardToDeck.Client.Id]; isExist {

				if player.Owner.MaxDeckSlot > int32(len(player.Deck)) {

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

				}

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

				if player.Owner.MaxReserveSlot > int32(len(player.Reserve)) {

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

				}

				// send back to client
				err := stream.Send(&LobbyStream{
					Event: evt,
				})

				if err != nil {
					return err
				}
			}

		default:
		}
	}
}
