package cardBattle

import (
	"sync"
	"time"

	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

func (c *CardBattleServer) newRoomHub(id string) {
	go func(s *CardBattleServer, idRoom string) {

		for {
			select {
			case res := <-s.Room[idRoom].Broadcast:
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
					close(s.Room[idRoom].LocalBroadcast)

					s.streamsMtx.RLock()
					delete(s.Room, idRoom)
					s.streamsMtx.RUnlock()

					return

				default:
				}

			default:
			}

		}

	}(c, id)

}

func (c *Room) openStream(p *Player) (stream chan RoomStream) {

	stream = make(chan RoomStream, 100)
	c.streamsMtx.Lock()
	c.ClientStreams[p.Id] = stream
	c.streamsMtx.Unlock()

	return
}

func (c *Room) closeStream(p *Player) {

	c.streamsMtx.Lock()
	if stream, ok := c.ClientStreams[p.Id]; ok {
		delete(c.ClientStreams, p.Id)
		close(stream)
	}

	c.streamsMtx.Unlock()
}

func (c *Room) receiveRoomBroadcasts(stream CardBattleService_CardBattleRoomStreamServer, p *Player) {

	streamClient := c.openStream(p)
	defer c.closeStream(p)

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

// this function only run
// if player flag in room status
// is 1
func (c *CardBattleServer) startRoomBattleCountdown(id string) {
	go func(s *CardBattleServer, idRoom string) {

		var value int32 = s.Room[idRoom].Data.CoolDownTime

		for {
			select {

			// this for stoping the loop manual way
			// if a signal comming from local broadcast
			case res := <-s.Room[idRoom].LocalBroadcast:
				switch res.BattleFlag {
				case 0:
				case 1:

					// stop this loop
					// for countdown battle
					return
				default:
				}

			default:

				// if countdown is 0
				switch value {
				case 0:

					winer := &Player{}
					withFullHp := []*Player{}
					withZeroHp := []*Player{}
					results := []*PlayerBattleResult{}

					s.Room[idRoom].streamsMtx.RLock()
					// to start compare result

					for _, p1 := range s.Room[idRoom].Data.Players {
						pResult := &PlayerBattleResult{
							Owner:         p1.Owner,
							DamageReceive: 0,
						}

						for _, p2 := range s.Room[idRoom].Data.Players {
							if p2.Owner.Id != p1.Owner.Id {
								atkP2 := p2.Owner.getTotalPlayerAtkCards(p2.Deployed)
								defP1 := p1.Owner.getTotalPlayerDefCards(p1.Deployed)
								dam := atkP2 - defP1
								if dam < 0 {
									dam = 0
								}
								pResult.DamageReceive += dam
							}

						}

						results = append(results, pResult)
					}

					// apply player damage
					for _, d := range results {
						for i, _ := range s.Room[idRoom].Data.Players {
							if s.Room[idRoom].Data.Players[i].Owner.Id == d.Owner.Id {
								s.Room[idRoom].Data.Players[i].Hp -= d.DamageReceive
							}

							// if player hp is negative
							// force set to 0
							if s.Room[idRoom].Data.Players[i].Hp < 0 {
								s.Room[idRoom].Data.Players[i].Hp = 0
								withZeroHp = append(withZeroHp, s.Room[idRoom].Data.Players[i].Owner)
							}

							if s.Room[idRoom].Data.Players[i].Hp > 0 {
								withFullHp = append(withFullHp, s.Room[idRoom].Data.Players[i].Owner)
							}
						}
					}

					// remove all player deployed deck
					for i, _ := range s.Room[idRoom].Data.Players {
						s.Room[idRoom].Data.Players[i].Deployed = []*Card{}
					}

					// only need one winer
					if len(withFullHp) == 1 {
						for _, p := range withFullHp {
							winer = p
						}
					}

					// from player deck and
					// update data in room
					s.Room[idRoom].streamsMtx.RUnlock()

					//send who is the winer is
					// and stop the looping
					if winer.Id != "" {

						// set winner prize
						c.Players[winer.Id].Owner.Cash += c.Room[idRoom].Data.CashReward

						for _, card := range c.Room[idRoom].Data.CardReward {
							c.Players[winer.Id].Reserve = append(c.Players[winer.Id].Reserve, card)
						}

						// broadcash who is winner is
						c.Room[idRoom].Broadcast <- RoomStream{
							Event: &RoomStream_OnWinner{
								OnWinner: winer,
							},
						}

						// stop the loop
						return
					}

					// send signal
					// announce the battle result
					c.Room[idRoom].Broadcast <- RoomStream{
						Event: &RoomStream_Result{
							Result: &AllPlayerBattleResult{
								Results: results,
							},
						},
					}

					// reset countdown
					value = s.Room[idRoom].Data.CoolDownTime

				default:

					// broadcast to player in room
					// count down for battle value
					c.Room[idRoom].Broadcast <- RoomStream{
						Event: &RoomStream_CountDown{
							CountDown: value,
						},
					}

					value--
				}

			}

			time.Sleep(1 * time.Second)
		}

	}(c, id)
}
