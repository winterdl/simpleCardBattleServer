package cardBattle

var (
	PLAYER_NOT_FOUND = &LobbyStream{
		Event: &LobbyStream_ExcMessage{
			ExcMessage: &ExceptionErrorMessage{
				ExceptionMessage: "player is not found",
				ExceptionFlag:    1,
			},
		},
	}
	ROOM_NOT_FOUND = &RoomStream{
		Event: &RoomStream_ExcMessage{
			ExcMessage: &ExceptionErrorMessage{
				ExceptionMessage: "room is not found",
				ExceptionFlag:    2,
			},
		},
	}
)
