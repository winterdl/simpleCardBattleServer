package cardBattle

import (
	"sync"

	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type QueueRoom struct {
	PlayersInWaitingRoom map[string]*PlayerWithCards
	Broadcast            chan QueueStream
	ClientStreams        map[string]chan QueueStream
	streamsMtx           sync.RWMutex
}

func (b *QueueRoom) NewHub() *QueueRoom {
	go func(h *QueueRoom) {
		for {
			for res := range h.Broadcast {
				h.streamsMtx.RLock()
				for _, stream := range h.ClientStreams {
					select {
					case stream <- res:
					default:
					}
				}
				h.streamsMtx.RUnlock()
			}
		}
	}(b)

	return b
}

func (c *QueueRoom) openStream(tkn string) (stream chan QueueStream) {

	stream = make(chan QueueStream, 100)
	c.streamsMtx.Lock()
	c.ClientStreams[tkn] = stream
	c.streamsMtx.Unlock()

	return
}

func (c *QueueRoom) closeStream(tkn string) {

	c.streamsMtx.Lock()
	if stream, ok := c.ClientStreams[tkn]; ok {
		delete(c.ClientStreams, tkn)
		close(stream)
	}

	c.streamsMtx.Unlock()
}

func (c *QueueRoom) receiveBroadcasts(stream CardBattleService_CardBattleQueueStreamServer, idPlayer string) {

	streamClient := c.openStream(idPlayer)
	defer c.closeStream(idPlayer)

	for {
		select {
		case <-stream.Context().Done():
			return
		case res := <-streamClient:
			if s, ok := status.FromError(stream.Send(&res)); ok {
				switch s.Code() {
				case codes.OK:
				case codes.Unavailable, codes.Canceled, codes.DeadlineExceeded:
					return
				default:
					return
				}
			}
		}
	}
}
