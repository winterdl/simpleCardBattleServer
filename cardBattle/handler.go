package cardBattle

import (
	context "context"
	"sync"
)

type CardBattleServer struct {
	Ctx        context.Context
	Config     *ServerConfig
	Lobby      *Lobby
	Queue      *QueueRoom
	RoomConfig *RoomManagementConfig
	Room       map[string]*Room
	Players    map[string]*PlayerWithCards
	Shop       *CardShop
	streamsMtx sync.RWMutex
}
