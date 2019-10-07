package cardBattle

import (
	"sync"
	"time"
)

type Room struct {
	ID            string
	RoomExpired   time.Time
	Data          *RoomData
	Broadcast     chan RoomStream
	ClientStreams map[string]chan RoomStream
	streamsMtx    sync.RWMutex
}

func (c *CardBattleServer) newRoomHub(id string) {
	go func(s *CardBattleServer, idRoom string) {

		for {
			res := <-s.Room[idRoom].Broadcast
			switch res.RoomFlag {
			case 0:

				s.Room[idRoom].streamsMtx.RLock()
				for _, stream := range s.Room[idRoom].ClientStreams {
					select {
					case stream <- res:
					default:
					}
				}
				s.Room[idRoom].streamsMtx.RUnlock()

			case 1:

				close(s.Room[idRoom].Broadcast)
				delete(s.Room, idRoom)

				return

			default:
			}

		}

	}(c, id)

}
