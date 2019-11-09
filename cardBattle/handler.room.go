package cardBattle

import (
	"io"
	"sync"
	"time"
)

type Room struct {
	ID             string
	RoomExpired    time.Time
	Data           *RoomData
	LocalBroadcast chan RoomStream
	Broadcast      chan RoomStream
	ClientStreams  map[string]chan RoomStream
	streamsMtx     sync.RWMutex
}

func (c *CardBattleServer) CardBattleRoomStream(stream CardBattleService_CardBattleRoomStreamServer) error {

	for {

		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		switch evt := msg.Event.(type) {

		case *RoomStream_PlayerJoin:

			// check if room is exist
			_, isExist := c.Room[msg.IdRoom]
			if !isExist {
				err := stream.Send(ROOM_NOT_FOUND)
				if err != nil {
					return err
				}

			} else {

				// check is player is already join
				isJoin := false
				for _, v := range c.Room[msg.IdRoom].Data.Players {
					if v.Owner.Id == evt.PlayerJoin.Owner.Id {
						isJoin = true
						break
					}
				}

				if !isJoin {

					// player is join
					p, err := c.Players[evt.PlayerJoin.Owner.Id].makeCopy()
					if err != nil {
						return err
					}

					p.Hp = c.Room[msg.IdRoom].Data.EachPlayerHealth

					c.Room[msg.IdRoom].Data.Players = append(c.Room[msg.IdRoom].Data.Players, p)

					// player receive broadcast
					go c.Room[msg.IdRoom].receiveRoomBroadcasts(stream, p.Owner)

					// broadcast to all player
					// in room, player in room
					// is joining
					c.Room[msg.IdRoom].Broadcast <- RoomStream{
						IdRoom: msg.IdRoom,
						Event:  evt,
					}

					// check player length in room
					if len(c.Room[msg.IdRoom].Data.Players) == int(c.Room[msg.IdRoom].Data.MaxPlayer) || len(c.Room[msg.IdRoom].ClientStreams) == int(c.Room[msg.IdRoom].Data.MaxPlayer) {

						// start the game
						c.startRoomBattleCountdown(msg.IdRoom)

					}

					time.Sleep(5 * time.Second)

					_, isExist = c.Room[msg.IdRoom]
					if isExist {
						// broadcast room is update
						c.Room[msg.IdRoom].Broadcast <- RoomStream{
							IdRoom: msg.IdRoom,
							Event: &RoomStream_OnRoomUpdate{
								OnRoomUpdate: c.Room[msg.IdRoom].Data,
							},
						}
					}
				}
			}

		case *RoomStream_PlayerLeft:

			// check if room is exist
			_, isExist := c.Room[msg.IdRoom]
			if !isExist {
				err := stream.Send(ROOM_NOT_FOUND)
				if err != nil {
					return err
				}

			} else {

				// remove from player list
				// by simply create new player list
				// without player who leave
				players := []*PlayerWithCards{}
				for _, p := range c.Room[msg.IdRoom].Data.Players {
					if p.Owner.Id != evt.PlayerLeft.Owner.Id {
						players = append(players, p)
					}
				}
				c.Room[msg.IdRoom].Data.Players = players

				// broadcast to all player
				// in room, player in room
				// is leaving
				c.Room[msg.IdRoom].Broadcast <- RoomStream{
					IdRoom: msg.IdRoom,
					Event:  evt,
				}
			}

		case *RoomStream_OnRoomUpdate:

			// left this empty

		case *RoomStream_CountDown:

			// left this empty

		case *RoomStream_BattleResult:

			// left this empty

		case *RoomStream_Result:

			// left this empty

		case *RoomStream_OnDraw:

			// left this empty

		case *RoomStream_DeployCard:

			// check if room is exist
			_, isExist := c.Room[msg.IdRoom]
			if !isExist {
				err := stream.Send(ROOM_NOT_FOUND)
				if err != nil {
					return err
				}

			} else {

				room := c.Room[msg.IdRoom]
				posPlayer, playerExist := findPlayer(evt.DeployCard.Client.Id, room.Data.Players)
				cardPos, cardExist := findCard(evt.DeployCard.CardData.Id, room.Data.Players[posPlayer].Deck)

				// add card from player deck
				// to deployed deck
				if playerExist && cardExist && int32(len(room.Data.Players[posPlayer].Deployed)) < room.Data.MaxCurrentDeployment {
					card, _ := c.Room[msg.IdRoom].Data.Players[posPlayer].Deck[cardPos].makeCopy()
					c.Room[msg.IdRoom].Data.Players[posPlayer].Deployed = append(c.Room[msg.IdRoom].Data.Players[posPlayer].Deployed, card)
					c.Room[msg.IdRoom].Data.Players[posPlayer].Deck = removeOneCard(evt.DeployCard.CardData.Id, c.Room[msg.IdRoom].Data.Players[posPlayer].Deck)
				}

				// broadcast to all player
				// in room, room data is updated
				c.Room[msg.IdRoom].Broadcast <- RoomStream{
					IdRoom: msg.IdRoom,
					Event: &RoomStream_OnRoomUpdate{
						OnRoomUpdate: c.Room[msg.IdRoom].Data,
					},
				}
			}

		case *RoomStream_PickupCard:

			// check if room is exist
			_, isExist := c.Room[msg.IdRoom]
			if !isExist {
				err := stream.Send(ROOM_NOT_FOUND)
				if err != nil {
					return err
				}

			} else {

				posPlayer, playerExist := findPlayer(evt.PickupCard.Client.Id, c.Room[msg.IdRoom].Data.Players)
				cardPos, cardExist := findCard(evt.PickupCard.CardData.Id, c.Room[msg.IdRoom].Data.Players[posPlayer].Deployed)

				// add card from player deck
				// to deployed deck
				if playerExist && cardExist {

					card, _ := c.Room[msg.IdRoom].Data.Players[posPlayer].Deployed[cardPos].makeCopy()

					// deployable
					if int32(len(c.Room[msg.IdRoom].Data.Players[posPlayer].Deployed)) < c.Room[msg.IdRoom].Data.MaxCurrentDeployment {
						c.Room[msg.IdRoom].Data.Players[posPlayer].Deployed = removeOneCard(evt.PickupCard.CardData.Id, c.Room[msg.IdRoom].Data.Players[posPlayer].Deployed)
						c.Room[msg.IdRoom].Data.Players[posPlayer].Deck = append(c.Room[msg.IdRoom].Data.Players[posPlayer].Deck, card)
					}
				}

				// broadcast to all player
				// in room, room data is updated
				c.Room[msg.IdRoom].Broadcast <- RoomStream{
					IdRoom: msg.IdRoom,
					Event: &RoomStream_OnRoomUpdate{
						OnRoomUpdate: c.Room[msg.IdRoom].Data,
					},
				}
			}

		case *RoomStream_GetOneRoom:

			// check if room is exist
			// and return as response
			room, isExist := c.Room[msg.IdRoom]
			if !isExist {

				err := stream.Send(ROOM_NOT_FOUND)
				if err != nil {
					return err
				}

			} else {

				err := stream.Send(&RoomStream{
					IdRoom: msg.IdRoom,
					Event: &RoomStream_GetOneRoom{
						GetOneRoom: room.Data,
					},
				})
				if err != nil {
					return err
				}

			}

		default:
		}
	}

}
