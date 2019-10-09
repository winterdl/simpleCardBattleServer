package cardBattle

import (
	"encoding/json"
	"math/rand"
)

type ServerConfig struct {
	AmountDefaultCash int32
	AmountDefaultCard int32
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

func (p *PlayerWithCards) initPlayerCard(MaxCardOnDeck int32) error {

	// set card for player card deck
	for i := 0; i < int(MaxCardOnDeck); i++ {

		cards, err := (&CardShop{TotalCard: int(MaxCardOnDeck)}).RandomCard(4)
		if err != nil {
			return err
		}

		p.Deck = cards

	}
	// set card for player card reserve
	for i := 0; i < int(MaxCardOnDeck); i++ {

		cards, err := (&CardShop{TotalCard: int(MaxCardOnDeck)}).RandomCard(4)
		if err != nil {
			return err
		}

		p.Reserve = cards

	}

	return nil

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

func (p *Player) getTotalPlayerAtkCards(cards []*Card) int32 {
	var total int32 = 0
	for _, c := range cards {
		total += c.Atk
	}

	return total
}

func (p *Player) getTotalPlayerDefCards(cards []*Card) int32 {
	var total int32 = 0
	for _, c := range cards {
		total += c.Def
	}

	return total
}
