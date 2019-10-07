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
				break
			default:

				// choose player from waiting room
				// choose until player needed reach max
				players := []*PlayerWithCards{}

				for len(players) < int(s.RoomConfig.DefaultMaxPlayer) {

					s.streamsMtx.RLock()
					for _, p := range s.PlayersInWaitingRoom {

						player, errCopy := p.makeCopy()
						if errCopy != nil {
							log.Println(errCopy)
						}

						if !checkIfAlreadyInQueue(player, players) && checkIfLevelSame(player, players) {
							players = append(players, player)

						}
					}
					s.streamsMtx.RUnlock()

				}

				if len(players) == int(r.RoomConfig.DefaultMaxPlayer) {

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
					s.Room[room.Id] = &Room{
						ID:            room.Id,
						Data:          room,
						RoomExpired:   timeExp,
						Broadcast:     make(chan RoomStream),
						ClientStreams: make(map[string]chan RoomStream),
					}

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
				}

				time.Sleep(15 * time.Second)

			}
		}
	}(r)

}
func (r *CardBattleServer) RemoveEmptyRoom() {
	go func(s *CardBattleServer) {
		for {
			select {
			case <-s.Ctx.Done():
				break
			default:

				for _, room := range s.Room {
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
