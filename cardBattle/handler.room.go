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

		case *RoomStream_PlayerLeft:

		case *RoomStream_GetOneplayer:

		case *RoomStream_GetOneRoom:

		default:
		}
	}
}
