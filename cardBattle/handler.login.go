package cardBattle

import (
	context "context"
	fmt "fmt"

	uuid "github.com/satori/go.uuid"
)

func (c *CardBattleServer) CardBattleLogin(ctx context.Context, in *Player) (*Player, error) {

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
				Id:     id,
				Name:   in.Name,
				Avatar: in.Avatar,
				Level:  1,
				Cash:   c.Config.AmountDefaultCash,
			},
			Deck:     []*Card{},
			Deployed: []*Card{},
			Reserve:  []*Card{},
			Hp:       100,
		}

		player.initPlayerCard(c.Config.AmountDefaultCard)

		c.streamsMtx.RLock()
		// add to lobby
		c.Players[id] = player

		c.streamsMtx.RUnlock()

		response = player.Owner

	}

	return response, nil
}