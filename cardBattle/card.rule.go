package cardBattle

import (
	"encoding/json"
	"math/rand"
)

type ServerConfig struct {
	AmountDefaultCash int64
	AmountDefaultCard int32
	AmountDefaultExp  int64
	Level             int32
	MaxReserveSlot    int32
	MaxDeckSlot       int32
}

type NameData struct {
	Names []string `json:"names"`
}

func (c *Card) isObtainable(player *Player) bool {
	return player.Level+3 >= c.Level
}

func random(min int, max int) int {
	return rand.Intn(max-min) + min
}

func randomInt64(min int64, max int64) int64 {
	return rand.Int63n(max-min) + min
}

func (p *PlayerWithCards) initPlayerCard(MaxCardOnDeck int32, maxCardLevel int) error {

	// set card for player card deck
	for i := 0; i < int(MaxCardOnDeck); i++ {

		cards, err := (&CardShop{TotalCard: int(MaxCardOnDeck)}).RandomCard(maxCardLevel)
		if err != nil {
			return err
		}

		p.Deck = cards

	}
	// set card for player card reserve
	for i := 0; i < int(MaxCardOnDeck); i++ {

		cards, err := (&CardShop{TotalCard: int(MaxCardOnDeck)}).RandomCard(maxCardLevel)
		if err != nil {
			return err
		}

		p.Reserve = cards

	}

	return nil

}

func checkIfAllPlayerHpIsSame(p []*PlayerWithCards) bool {
	same := true
	current := int64(0)

	if len(p) == 1 {
		return false
	}

	for i, v := range p {

		if i == 0 {
			current = v.Hp
		}

		if v.Hp != current {
			return false
		}

		current = v.Hp
	}

	return same
}

func (p *PlayerWithCards) makeCopy() (*PlayerWithCards, error) {
	player := &PlayerWithCards{}

	j, err := json.Marshal(p)
	if err != nil {
		return player, err
	}
	err = json.Unmarshal(j, player)
	if err != nil {
		return player, err
	}

	return player, nil
}

func (p *Card) makeCopy() (*Card, error) {
	card := &Card{}

	j, err := json.Marshal(p)
	if err != nil {
		return card, err
	}
	err = json.Unmarshal(j, card)
	if err != nil {
		return card, err
	}

	return card, nil
}

func (r *RoomData) makeCopy() (*RoomData, error) {
	room := &RoomData{}

	j, err := json.Marshal(r)
	if err != nil {
		return room, err
	}
	err = json.Unmarshal(j, room)
	if err != nil {
		return room, err
	}

	return room, nil
}

func (p *Player) getTotalPlayerAtkCards(cards []*Card) int64 {
	var total int64 = 0
	for _, c := range cards {
		total += c.Atk
	}

	return total
}

func (p *Player) getTotalPlayerDefCards(cards []*Card) int64 {
	var total int64 = 0
	for _, c := range cards {
		total += c.Def
	}

	return total
}
func getPlayerWithMaxHp(players []*PlayerWithCards) (*PlayerWithCards, int64) {
	var current = int64(0)
	pos := 0
	for i, c := range players {
		if i == 0 || c.Hp > current {
			current = c.Hp
			pos = i
		}
	}

	return players[pos], current
}
func findPlayer(id string, p []*PlayerWithCards) (int, bool) {
	for i, v := range p {
		if v.Owner.Id == id {
			return i, true
		}
	}
	return 0, false
}

func findCard(id string, c []*Card) (int, bool) {
	for i, v := range c {
		if v.Id == id {
			return i, true
		}
	}
	return 0, false
}

func removeOneCard(id string, c []*Card) []*Card {
	cards := []*Card{}
	for _, v := range c {
		if v.Id != id {
			c, _ := v.makeCopy()
			cards = append(cards, c)
		}
	}
	return cards
}
