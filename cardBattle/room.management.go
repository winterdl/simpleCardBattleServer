package cardBattle

import (
	fmt "fmt"
	"log"
	"time"

	util "github.com/renosyah/simpleCardBattleServer/util"
	uuid "github.com/satori/go.uuid"
)

type RoomManagementConfig struct {
	ExpiredTime             int32
	DefaultMaxPlayer        int32
	DefaultMaxPlayerDeck    int32
	DefaultMaxDeploment     int32
	DefaultEachPlayerHealth int32
	DefaultCoolDownTime     int32
	DefaultMaxCardReward    int32
	DefaultCashReward       int32
	DefaultLevelReward      int32
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
	for _, player := range ps {
		if player.Owner.Level != p.Owner.Level {
			return false
		}
	}

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

				s.streamsMtx.RLock()
				for _, p := range s.PlayersInWaitingRoom {

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
				s.streamsMtx.RUnlock()

				if len(players) == int(s.RoomConfig.DefaultMaxPlayer) {

					for _, p := range players {
						if _, isExist := s.PlayersInWaitingRoom[p.Owner.Id]; isExist {
							s.streamsMtx.RLock()
							delete(s.PlayersInWaitingRoom, p.Owner.Id)
							s.streamsMtx.RUnlock()
						}
					}

					// make random room
					room, err := s.RoomConfig.makeRandomRoom(players)
					if err != nil {
						log.Println(err)
					}

					timeSet := time.Now().Local()
					timeExp := timeSet.Add(time.Hour*time.Duration(0) +
						time.Minute*time.Duration(0) +
						time.Second*time.Duration(int(s.RoomConfig.ExpiredTime)))

					// add room to room list
					s.streamsMtx.RLock()
					s.Room[room.Id] = &Room{
						ID:            room.Id,
						Data:          room,
						RoomExpired:   timeExp,
						Broadcast:     make(chan RoomStream),
						ClientStreams: make(map[string]chan RoomStream),
					}
					s.streamsMtx.RUnlock()

					// run room hub
					s.newRoomHub(room.Id)

					// broadcast to player
					// room hass been create
					// for player to battle
					for _, player := range players {
						s.Lobby.ClientStreams[player.Owner.Id] <- LobbyStream{
							Event: &LobbyStream_OnBattleFound{
								OnBattleFound: room,
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
				s.streamsMtx.RLock()
				for _, room := range s.Room {
					s.Room[room.ID].Broadcast <- RoomStream{
						RoomFlag: 1,
					}
				}

				s.streamsMtx.RUnlock()
				log.Println("room remover is stoped")
				return

			default:

				s.streamsMtx.RLock()
				for _, room := range s.Room {

					// if room player is none
					// and
					// room is expired
					if len(room.Data.Players) == 0 && time.Now().Local().After(room.RoomExpired) {

						// send room flag set into 1
						// this will triger looping in hub
						// to stop broadcast in room
						// and shut down the room it self
						s.Room[room.ID].Broadcast <- RoomStream{
							RoomFlag: 1,
						}

					}
				}
				s.streamsMtx.RUnlock()

				time.Sleep(1 * time.Second)
			}
		}
	}(r)
}
func (r *RoomManagementConfig) makeRandomRoom(players []*PlayerWithCards) (*RoomData, error) {

	var avgLevel int32 = 2
	for _, p := range players {
		if p.Owner.Level > avgLevel {
			avgLevel = p.Owner.Level
		}
	}

	cards, err := (&CardShop{TotalCard: int(r.DefaultMaxCardReward)}).RandomCard(int(avgLevel))
	if err != nil {
		return &RoomData{}, err
	}

	room := &RoomData{
		Id:               fmt.Sprint("", uuid.Must(uuid.NewV4())),
		RoomName:         fmt.Sprintf("Room - %s", util.GenerateRandomName()),
		Players:          []*PlayerWithCards{},
		MaxPlayer:        r.DefaultMaxPlayer,
		MaxPlayerDeck:    r.DefaultMaxPlayerDeck,
		MaxDeploment:     r.DefaultMaxDeploment,
		EachPlayerHealth: r.DefaultEachPlayerHealth,
		CoolDownTime:     r.DefaultCoolDownTime,
		CardReward:       cards,
		CashReward:       r.DefaultCashReward,
		LevelReward:      r.DefaultLevelReward,
	}

	return room, nil
}
