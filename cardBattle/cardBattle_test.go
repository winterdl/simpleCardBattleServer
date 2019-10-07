package cardBattle

import (
	"encoding/json"
	fmt "fmt"
	"testing"

	uuid "github.com/satori/go.uuid"
)

func TestPlayer(t *testing.T) {
	id := fmt.Sprint("", uuid.Must(uuid.NewV4()))

	// init data player
	// by add card and cash
	player := &PlayerWithCards{
		Owner: &Player{
			Id:     id,
			Name:   "test",
			Avatar: "test",
			Level:  1,
			Cash:   120,
		},
		Deck:             []*Card{},
		Deployed:         []*Card{},
		Reserve:          []*Card{},
		TurnId:           0,
		PassTurn:         false,
		Hp:               100,
		MaxCardOnDeck:    5,
		MaxCardOnReserve: 5,
	}

	err := player.initPlayerCard()
	if err != nil {
		t.Log(err)
	}

	res, _ := json.Marshal(player)
	t.Logf(string(res))
}
