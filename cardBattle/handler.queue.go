package cardBattle

import (
	"io"
	"log"
	"sync"
)

type QueueRoom struct {
	PlayersInWaitingRoom map[string]*PlayerWithCards
	Broadcast            chan QueueStream
	ClientStreams        map[string]chan QueueStream
	streamsMtx           sync.RWMutex
}

func (c *CardBattleServer) CardBattleQueueStream(stream CardBattleService_CardBattleQueueStreamServer) error {

	for {

		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		switch evt := msg.Event.(type) {

		case *QueueStream_OnjoinWaitingRoom:

			// make copy of player
			player, errCopy := c.Players[evt.OnjoinWaitingRoom.Id].makeCopy()
			if errCopy != nil {
				log.Println(errCopy)
			}

			c.Queue.streamsMtx.RLock()
			c.Queue.PlayersInWaitingRoom[evt.OnjoinWaitingRoom.Id] = player
			c.Queue.streamsMtx.RUnlock()

			// receive broadcast
			go c.Queue.receiveBroadcasts(stream, evt.OnjoinWaitingRoom.Id)

			err := stream.Send(&QueueStream{
				Event: evt,
			})

			if err != nil {
				return err
			}

		case *QueueStream_OnLeftWaitingRoom:

			c.Queue.streamsMtx.Lock()
			if p, exists := c.Queue.PlayersInWaitingRoom[evt.OnLeftWaitingRoom.Id]; exists {
				delete(c.Queue.PlayersInWaitingRoom, p.Owner.Id)
			}
			c.Queue.streamsMtx.Unlock()

			err := stream.Send(&QueueStream{
				Event: evt,
			})

			if err != nil {
				return err
			}

		case *QueueStream_OnBattleFound:

			// left this empty

		case *QueueStream_OnBattleNotFound:

			// left this empty

		default:
		}
	}
}
