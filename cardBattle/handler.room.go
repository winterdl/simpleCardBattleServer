package cardBattle

import (
	"errors"
	"io"
)

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
				return errors.New("room is not exist")
			}

			// player is join
			c.Room[msg.IdRoom].Data.Players = append(c.Room[msg.IdRoom].Data.Players, evt.PlayerJoin)

			// player receive broadcast
			go c.Room[msg.IdRoom].receiveRoomBroadcasts(stream, evt.PlayerJoin.Owner)

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
				go c.startRoomBattleCountdown(msg.IdRoom)

			}

		case *RoomStream_PlayerLeft:

			// check if room is exist
			_, isExist := c.Room[msg.IdRoom]
			if !isExist {
				return errors.New("room is not exist")
			}

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

			// check player length in room
			if len(c.Room[msg.IdRoom].Data.Players) < int(c.Room[msg.IdRoom].Data.MaxPlayer) || len(c.Room[msg.IdRoom].ClientStreams) < int(c.Room[msg.IdRoom].Data.MaxPlayer) {

				// stop countdown game
				// if player is not present
				// by sending battle flag signal to 1
				c.Room[msg.IdRoom].LocalBroadcast <- RoomStream{
					IdRoom:      msg.IdRoom,
					PlayersFlag: 1,
				}

			}

		case *RoomStream_OnRoomUpdate:

			// left this empty

		case *RoomStream_CountDown:

			// left this empty

		case *RoomStream_Result:

			// left this empty

		case *RoomStream_OnWinner:

			// left this empty

		case *RoomStream_DeployCard:

			// add card from player deck
			// to deployed deck
			// broadcast to all player
			// in room, room data is updated

		case *RoomStream_PickupCard:

			// remove card from deployed deck
			// and put back to player deck
			// broadcast to all player
			// in room, room data is updated

		case *RoomStream_GetOneRoom:

			// check if room is exist
			// and return as response
			if room, isExist := c.Room[msg.IdRoom]; isExist {
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
