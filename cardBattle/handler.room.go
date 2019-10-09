package cardBattle

import "io"

func (c *CardBattleServer) CardBattleRoomStream(stream CardBattleService_CardBattleRoomStreamServer) error {

	for {

		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		switch msg.Event.(type) {

		case *RoomStream_PlayerJoin:

			// broadcast to all player
			// in room, player in room
			// is joining

		case *RoomStream_PlayerLeft:

			// broadcast to all player
			// in room, player in room
			// is leaving

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

			// return to player room data

		default:
		}
	}

}
