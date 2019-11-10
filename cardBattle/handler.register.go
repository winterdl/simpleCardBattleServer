package cardBattle

import (
	context "context"
	fmt "fmt"

	uuid "github.com/satori/go.uuid"
)

func (c *CardBattleServer) CardBattleRegister(ctx context.Context, in *Player) (*Player, error) {

	response := &Player{}

	_, isExist := c.Players[in.Id]

	if isExist {
		return c.Players[in.Id].Owner, nil
	}

	if !isExist {

		id := fmt.Sprint("", uuid.Must(uuid.NewV4()))

		// init data player
		// by add card and cash
		player := &PlayerWithCards{
			Owner: &Player{
				Id:             id,
				Name:           in.Name,
				Avatar:         in.Avatar,
				Level:          c.Config.Level,
				Exp:            0,
				MaxExp:         c.Config.AmountDefaultExp,
				Cash:           c.Config.AmountDefaultCash,
				MaxDeckSlot:    c.Config.MaxDeckSlot,
				MaxReserveSlot: c.Config.MaxReserveSlot,
			},
			Deck:     []*Card{},
			Deployed: []*Card{},
			Reserve:  []*Card{},
			Hp:       100,
		}

		lvlCard := 2
		if c.Config.Level > 1 {
			lvlCard = int(c.Config.Level)
		}

		player.initPlayerCard(c.Config.AmountDefaultCard, lvlCard, c.Shop.URLFile)

		c.streamsMtx.RLock()
		// add to lobby
		c.Players[id] = player

		c.streamsMtx.RUnlock()

		response = player.Owner

	}

	return response, nil
}
