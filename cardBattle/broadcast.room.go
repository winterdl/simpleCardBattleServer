package cardBattle

import (
	"time"

	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func (rm *Room) newRoomHub(c *CardBattleServer) {
	go func(s *CardBattleServer, r *Room) {

		for {

			select {
			case res := <-r.Broadcast:
				switch res.RoomFlag {
				case 0:

					r.streamsMtx.RLock()
					for _, stream := range r.ClientStreams {
						select {
						case stream <- res:
						default:
						}
					}
					r.streamsMtx.RUnlock()

				case 1:

					close(r.Broadcast)
					close(r.LocalBroadcast)

					s.streamsMtx.Lock()
					delete(s.Room, r.ID)
					s.streamsMtx.Unlock()

					return

				default:
				}

			default:
			}
		}

	}(c, rm)

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

func (c *CardBattleServer) startRoomBattleCountdown(id string) {
	go func(s *CardBattleServer, idRoom string) {

		var value int32 = s.Room[idRoom].Data.CoolDownTime

		for {

			// check if room exist
			if _, isExist := s.Room[idRoom]; !isExist {
				// stoop loop cause
				// room not exist
				return
			}

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

					results := []*PlayerBattleResult{}

					s.Room[idRoom].streamsMtx.Lock()
					// to start compare result

					for _, p1 := range s.Room[idRoom].Data.Players {
						pResult := &PlayerBattleResult{
							Owner:         p1.Owner,
							DamageReceive: 0,
							EnemyAtk:      0,
							OwnerDef:      0,
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
								pResult.EnemyAtk = atkP2
								pResult.OwnerDef = defP1
							}

						}

						results = append(results, pResult)
					}

					// apply player damage
					for _, d := range results {
						for i, v := range s.Room[idRoom].Data.Players {
							if v.Owner.Id == d.Owner.Id {
								s.Room[idRoom].Data.Players[i].Hp -= d.DamageReceive

								// if player hp is negative
								// force set to 0
								if v.Hp < 0 {
									s.Room[idRoom].Data.Players[i].Hp = 0
								}
							}
						}
					}

					winers := []*Player{}
					totalHp := int64(0)
					totalCard := 0
					totalCardDeck := 0
					totalCardDeploy := 0
					players := s.Room[idRoom].Data.Players
					flagWinner := 0

					for _, v := range players {
						totalCardDeck += len(v.Deck)
						totalCardDeploy += len(v.Deployed)
						totalHp += v.Hp
					}

					if !checkIfAllPlayerHpIsIntack(c.Room[idRoom].Data.EachPlayerHealth, c.Room[idRoom].Data.Players) {

						// check the winner
						// from player with hp not 0
						for _, v := range players {
							if v.Hp > 0 && !checkIfAllPlayerHpIsSame(c.Room[idRoom].Data.Players) {
								winers = append(winers, v.Owner)
							}

							flagWinner = 0
						}

						totalCard = totalCardDeck + totalCardDeploy

						// check the winner
						// from player with more hp
						if totalCard == 0 && !checkIfAllPlayerHpIsSame(c.Room[idRoom].Data.Players) {
							p, _ := getPlayerWithMaxHp(players)
							winers = []*Player{}
							winers = append(winers, p.Owner)
							flagWinner = 1
						}

						// check the winner
						// from player with more hp
						// and no card deploy
						if totalCardDeploy == 0 && totalCardDeck > 0 && !checkIfAllPlayerHpIsSame(c.Room[idRoom].Data.Players) {
							p, _ := getPlayerWithMaxHp(players)
							winers = []*Player{}
							winers = append(winers, p.Owner)
							flagWinner = 2
						}
					}

					// remove all player deployed deck
					for i, _ := range s.Room[idRoom].Data.Players {
						s.Room[idRoom].Data.Players[i].Deployed = []*Card{}
					}

					// add more deployable card
					if s.Room[idRoom].Data.MaxCurrentDeployment < s.Room[idRoom].Data.MaxDeployment {
						s.Room[idRoom].Data.MaxCurrentDeployment++
					}

					// from player deck and
					// update data in room
					s.Room[idRoom].streamsMtx.Unlock()

					// only need one winer
					// send who is the winer is
					// and stop the looping
					if len(winers) == 1 {

						// set winner prize
						c.Players[winers[0].Id].Owner.Cash += c.Room[idRoom].Data.Reward.CashReward

						// all player in room receive xp
						for _, v := range c.Room[idRoom].Data.Players {
							c.Players[v.Owner.Id].Owner.Exp += c.Room[idRoom].Data.Reward.ExpReward
							if c.Players[v.Owner.Id].Owner.Exp >= c.Players[v.Owner.Id].Owner.MaxExp {
								c.Players[v.Owner.Id].Owner.Exp -= c.Players[v.Owner.Id].Owner.MaxExp
								c.Players[v.Owner.Id].Owner.Level++
								c.Players[v.Owner.Id].Owner.MaxExp = c.Config.AmountDefaultExp * int64(c.Players[v.Owner.Id].Owner.Level)
							}
						}

						cardsReceive := []*Card{}
						for _, card := range c.Room[idRoom].Data.Reward.CardReward {

							// reserve slot still have space
							if c.Players[winers[0].Id].Owner.MaxReserveSlot > int32(len(c.Players[winers[0].Id].Reserve)) {
								c.Players[winers[0].Id].Reserve = append(c.Players[winers[0].Id].Reserve, card)
								cardsReceive = append(cardsReceive, card)
							}
						}

						c.Room[idRoom].Data.Reward.CardReward = cardsReceive

						// broadcash who is winner is
						//  with winning status
						// 0 = is enemy player hp is 0
						// 1 = is enemy hp is lower and all deck card is 0
						// 2 = is enemy hp is lower and all deploy card is 0
						c.Room[idRoom].Broadcast <- RoomStream{
							Event: &RoomStream_Result{
								Result: &EndResult{
									Winner: c.Players[winers[0].Id].Owner,

									// not yet code
									AllBattleResult: []*PlayerBattleResult{},
									FlagResult:      int32(flagWinner),
									Reward:          c.Room[idRoom].Data.Reward,
								},
							},
						}

						// stop the loop
						return
					}

					// draw
					// 0 = is all player hp is 0
					// 1 = is all player card is 0 && hp is 0
					// 2 is all player card is 0 and all player hp is same
					// 3 no one fight on first round
					if totalHp == 0 && totalCardDeck > 0 {

						// broadcash draw
						c.Room[idRoom].Broadcast <- RoomStream{
							Event: &RoomStream_OnDraw{
								OnDraw: int32(0),
							},
						}

						// set player hp back to normal
						for i, _ := range c.Room[idRoom].Data.Players {
							c.Room[idRoom].Data.Players[i].Hp += c.Room[idRoom].Data.EachPlayerHealth
						}

					} else if totalHp == 0 && totalCardDeck == 0 {

						// broadcash draw
						c.Room[idRoom].Broadcast <- RoomStream{
							Event: &RoomStream_OnDraw{
								OnDraw: int32(1),
							},
						}

						// stop the loop
						return

					} else if checkIfAllPlayerHpIsSame(c.Room[idRoom].Data.Players) && totalCardDeck == 0 {

						// broadcash draw
						c.Room[idRoom].Broadcast <- RoomStream{
							Event: &RoomStream_OnDraw{
								OnDraw: int32(2),
							},
						}

						// stop the loop
						return

					} else if totalCardDeck > 0 && totalCardDeploy == 0 && checkIfAllPlayerHpIsIntack(c.Room[idRoom].Data.EachPlayerHealth, c.Room[idRoom].Data.Players) {

						// broadcash draw
						c.Room[idRoom].Broadcast <- RoomStream{
							Event: &RoomStream_OnDraw{
								OnDraw: int32(3),
							},
						}

						// stop the loop
						return

					} else {

						// send signal
						// announce the battle result
						c.Room[idRoom].Broadcast <- RoomStream{
							Event: &RoomStream_BattleResult{
								BattleResult: &AllPlayerBattleResult{
									Results: results,
								},
							},
						}
					}

					// reset countdown
					value = s.Room[idRoom].Data.CoolDownTime
					if totalCardDeck == 0 {
						value = 5
					}

				default:

					// broadcast to player in room
					// count down for battle value
					c.Room[idRoom].Broadcast <- RoomStream{
						Event: &RoomStream_CountDown{
							CountDown: &CountDownRoomUpdate{
								UpdatedRoom: s.Room[idRoom].Data,
								BattleTime:  value,
							},
						},
					}

					value--
				}

			}

			time.Sleep(1 * time.Second)
		}

	}(c, id)
}
