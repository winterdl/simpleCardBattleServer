package cardBattle

import (
	context "context"
	"sync"
)

type CardBattleServer struct {
	Ctx                  context.Context
	Config               *ServerConfig
	Lobby                *Lobby
	RoomConfig           *RoomManagementConfig
	Room                 map[string]*Room
	Players              map[string]*PlayerWithCards
	PlayersInWaitingRoom map[string]*PlayerWithCards
	Shop                 *CardShop
	streamsMtx           sync.RWMutex
}
