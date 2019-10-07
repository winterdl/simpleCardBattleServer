package cardBattle

import (
	"sync"

	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type Lobby struct {
	Broadcast     chan LobbyStream
	ClientStreams map[string]chan LobbyStream
	streamsMtx    sync.RWMutex
}

func (b *Lobby) NewHub() *Lobby {
	go func(h *Lobby) {
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
func (c *Lobby) openStream(tkn string) (stream chan LobbyStream) {

	stream = make(chan LobbyStream, 100)
	c.streamsMtx.Lock()
	c.ClientStreams[tkn] = stream
	c.streamsMtx.Unlock()

	return
}

func (c *Lobby) closeStream(tkn string) {

	c.streamsMtx.Lock()
	if stream, ok := c.ClientStreams[tkn]; ok {
		delete(c.ClientStreams, tkn)
		close(stream)
	}

	c.streamsMtx.Unlock()
}

func (c *Lobby) receiveBroadcasts(stream CardBattleService_CardBattleLobbyStreamServer, idPlayer string) {

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
