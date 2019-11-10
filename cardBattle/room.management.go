package cardBattle

import (
	fmt "fmt"
	"log"
	"time"

	util "github.com/renosyah/simpleCardBattleServer/util"
	uuid "github.com/satori/go.uuid"
)

type RoomManagementConfig struct {
	ExpiredTime              int32
	DefaultMaxPlayer         int32
	DefaultMaxPlayerDeck     int32
	DefaultCurrentDeployment int32
	DefaultMaxDeployment     int32
	DefaultEachPlayerHealth  int64
	DefaultCoolDownTime      int32
	DefaultMaxCardReward     int32
	DefaultCashReward        int64
	DefaultExpReward         int64
}

func checkIfAlreadyInQueue(p *PlayerWithCards, ps []*PlayerWithCards) bool {
	for _, player := range ps {
		if player.Owner.Id == p.Owner.Id {
			return true
		}
	}

	return false
}

func checkIfLevelSame(p *PlayerWithCards, ps []*PlayerWithCards) bool {
	// for _, player := range ps {
	// 	if player.Owner.Level != p.Owner.Level {
	// 		return false
	// 	}
	// }

	return true
}

// make room every one second
func (r *CardBattleServer) RunRoomMaker() {

	go func(s *CardBattleServer) {
		for {
			select {
			case <-s.Ctx.Done():

				log.Println("room maker is stoped")
				return

			default:

				// choose player from waiting room
				// choose until player needed reach max
				players := []*PlayerWithCards{}

				s.Queue.streamsMtx.RLock()
				for _, p := range s.Queue.PlayersInWaitingRoom {

					player, errCopy := p.makeCopy()
					if errCopy != nil {
						log.Println(errCopy)
					}

					if !checkIfAlreadyInQueue(player, players) && checkIfLevelSame(player, players) {
						players = append(players, player)

					}

					if len(players) == int(s.RoomConfig.DefaultMaxPlayer) {
						break
					}
				}
				s.Queue.streamsMtx.RUnlock()

				if len(players) == int(s.RoomConfig.DefaultMaxPlayer) {

					s.Queue.streamsMtx.Lock()
					for _, p := range players {
						if _, isExist := s.Queue.PlayersInWaitingRoom[p.Owner.Id]; isExist {
							delete(s.Queue.PlayersInWaitingRoom, p.Owner.Id)
						}
					}
					s.Queue.streamsMtx.Unlock()

					// make random room
					room, err := s.RoomConfig.makeRandomRoom(players, s.Shop.URLFile)
					if err != nil {
						log.Println(err)
					}

					timeSet := time.Now().Local()
					timeExp := timeSet.Add(time.Hour*time.Duration(0) +
						time.Minute*time.Duration(0) +
						time.Second*time.Duration(int(s.RoomConfig.ExpiredTime)))

					// add room to room list
					s.streamsMtx.Lock()
					s.Room[room.Id] = &Room{
						ID:             room.Id,
						Data:           room,
						RoomExpired:    timeExp,
						Broadcast:      make(chan RoomStream),
						ClientStreams:  make(map[string]chan RoomStream),
						LocalBroadcast: make(chan RoomStream),
					}
					s.streamsMtx.Unlock()

					// run room hub
					s.Room[room.Id].newRoomHub(s)

					// broadcast to player
					// room hass been create
					// for player to battle
					rcopy, _ := room.makeCopy()
					rcopy.Players = players
					for _, player := range players {
						s.Queue.ClientStreams[player.Owner.Id] <- QueueStream{
							Event: &QueueStream_OnBattleFound{
								OnBattleFound: rcopy,
							},
						}
					}

				} else {

					// for _, p := range players {
					// 	if _, isExist := s.PlayersInWaitingRoom[p.Owner.Id]; isExist {
					// 		s.streamsMtx.RLock()
					// 		delete(s.PlayersInWaitingRoom, p.Owner.Id)
					// 		s.streamsMtx.RUnlock()
					// 	}
					// }

					// // broadcast to player
					// // battle is not found
					// // because room not have enought player
					// for _, player := range players {
					// 	s.Lobby.ClientStreams[player.Owner.Id] <- LobbyStream{
					// 		Event: &LobbyStream_OnBattleNotFound{
					// 			OnBattleNotFound: true,
					// 		},
					// 	}
					// }

				}

				time.Sleep(5 * time.Second)

			}
		}
	}(r)

}
func (r *CardBattleServer) RemoveEmptyRoom() {
	go func(s *CardBattleServer) {
		for {
			select {
			case <-s.Ctx.Done():

				// just close all room
				s.streamsMtx.Lock()
				for _, room := range s.Room {
					s.Room[room.ID].Broadcast <- RoomStream{
						RoomFlag: 1,
					}
				}

				s.streamsMtx.Unlock()
				log.Println("room remover is stoped")
				return

			default:

				s.streamsMtx.Lock()
				for _, room := range s.Room {

					// if room player is none
					// and
					// room is expired
					if (len(room.Data.Players) == 0 || len(room.ClientStreams) == 0) && time.Now().Local().After(room.RoomExpired) {

						// send room flag set into 1
						// this will triger looping in hub
						// to stop broadcast in room
						// and shut down the room it self
						s.Room[room.ID].Broadcast <- RoomStream{
							RoomFlag: 1,
						}

					}
				}
				s.streamsMtx.Unlock()

				time.Sleep(1 * time.Second)
			}
		}
	}(r)
}
func (r *RoomManagementConfig) makeRandomRoom(players []*PlayerWithCards, URLFile string) (*RoomData, error) {

	var avgLevel int32 = 2
	for _, p := range players {
		if p.Owner.Level > avgLevel {
			avgLevel = p.Owner.Level
		}
	}

	cards, err := (&CardShop{TotalCard: random(2, int(r.DefaultMaxCardReward)), URLFile: URLFile}).RandomCard(int(avgLevel))
	if err != nil {
		return &RoomData{}, err
	}

	reward := &RoomReward{
		CardReward: cards,
		CashReward: randomInt64(r.DefaultCashReward, r.DefaultCashReward*int64(avgLevel)),
		ExpReward:  randomInt64(r.DefaultExpReward, r.DefaultExpReward*int64(avgLevel)),
	}

	room := &RoomData{
		Id:                   fmt.Sprint("", uuid.Must(uuid.NewV4())),
		RoomName:             fmt.Sprintf("%s", util.GenerateRandomName()),
		Players:              []*PlayerWithCards{},
		MaxPlayer:            r.DefaultMaxPlayer,
		MaxPlayerDeck:        r.DefaultMaxPlayerDeck,
		MaxCurrentDeployment: r.DefaultCurrentDeployment,
		MaxDeployment:        r.DefaultMaxDeployment,
		EachPlayerHealth:     SetHp(r.DefaultEachPlayerHealth, avgLevel),
		CoolDownTime:         r.DefaultCoolDownTime,
		Reward:               reward,
	}

	return room, nil
}

func SetHp(min int64, level int32) int64 {
	hp := min
	max := min * int64(level)
	for i := min; i < max; i = i * 10 {
		hp = i
	}
	return hp
}
