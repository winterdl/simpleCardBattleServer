package cardBattle

import (
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func (b *CardShop) NewHub() *CardShop {
	go func(h *CardShop) {
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

func (c *CardShop) openStream(tkn string) (stream chan ShopStream) {

	stream = make(chan ShopStream, 100)
	c.streamsMtx.Lock()
	c.ClientStreams[tkn] = stream
	c.streamsMtx.Unlock()

	return
}

func (c *CardShop) closeStream(tkn string) {

	c.streamsMtx.Lock()
	if stream, ok := c.ClientStreams[tkn]; ok {
		delete(c.ClientStreams, tkn)
		close(stream)
	}

	c.streamsMtx.Unlock()
}

func (c *CardShop) receiveBroadcasts(stream CardBattleService_CardBattleShopStreamServer, idPlayer string) {

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
