package cardBattle

import (
	"encoding/json"
	fmt "fmt"
	"log"
	"math/rand"
	"time"

	util "github.com/renosyah/simpleCardBattleServer/util"
	uuid "github.com/satori/go.uuid"
)

func (s *CardBattleServer) RunCardShop() {

	go func(serv *CardBattleServer) {

		rand.Seed(time.Now().UnixNano())

		var s = serv.Shop
		var timeLeft int32 = 0

		for {

			select {
			case <-serv.Ctx.Done():

				log.Println("shop is stoped")
				return

			default:
				switch timeLeft {
				case 0:

					if len(s.Cards) == s.MaxItem {
						idEndCard := ""
						for _, card := range s.Cards {
							idEndCard = card.Id
						}

						delete(s.Cards, idEndCard)
					}

					cards, err := s.RandomCard(random(s.MinLevel, s.MaxLevel))
					if err != nil {
						break
					}

					for _, card := range cards {
						s.Cards[card.Id] = card
					}

					// broadcast to all player
					// shop hass been refresh
					serv.Shop.Broadcast <- ShopStream{
						Event: &ShopStream_ShopRefresh{
							ShopRefresh: true,
						},
					}

					timeLeft = int32(s.Time)

				default:

					time.Sleep(1 * time.Second)
					timeLeft--

					// broadcast to all player
					// countdown for next shop item
					serv.Shop.Broadcast <- ShopStream{
						Event: &ShopStream_ShopRefreshTime{
							ShopRefreshTime: timeLeft,
						},
					}
				}
			}
		}

	}(s)

}

func (c *CardShop) RandomCard(Level int) ([]*Card, error) {

	cards := []*Card{}

	minLevel := 1
	maxLevel := Level

	minAtkDef := 10
	maxAtkDef := 150

	nameData := &NameData{}
	file, err := util.ReadFile("json/card_name.json")
	if err != nil {
		return cards, err
	}
	err = json.Unmarshal(file, nameData)
	if err != nil {
		return cards, err
	}

	images := []string{}
	for _, n := range nameData.Names {
		images = append(images, n)
	}

	names := nameData.Names

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < 15; i++ {
		names = append(names, util.GenerateRandomName())
	}

	for i := 0; i < int(c.TotalCard); i++ {

		firstName := names[random(0, len(names))]
		lastName := names[random(0, len(names))]

		if len(firstName) > 8 {
			lastName = ""
		}
		if len(lastName) > 8 {
			firstName = ""
		}

		card := &Card{
			Id:    fmt.Sprint("", uuid.Must(uuid.NewV4())),
			Image: fmt.Sprintf("%s%s.jpg", c.URLFile, images[random(0, len(images))]),
			Name:  fmt.Sprintf(`%s %s`, firstName, lastName),
			Color: 0,
			Price: int64(random(minAtkDef, maxAtkDef)),
			Level: int32(random(minLevel, maxLevel)),
			Atk:   int64(random(minAtkDef, maxAtkDef)),
			Def:   int64(random(minAtkDef, maxAtkDef)),
		}

		card.Color = checkCardColor(card, card.Level)
		card.Price = card.Price * int64(card.Level)
		card.Atk = card.Atk * int64(card.Level)
		card.Def = card.Def * int64(card.Level)

		cards = append(cards, card)

	}

	return cards, nil
}

func checkCardColor(card *Card, lvl int32) int32 {
	if card.Atk >= int64(140*lvl) || card.Def >= int64(140*lvl) {
		return 4
	} else if card.Atk >= int64(120*lvl) || card.Def >= int64(120*lvl) {
		return 3
	} else if card.Atk >= int64(90*lvl) || card.Def >= int64(90*lvl) {
		return 2
	} else if card.Atk >= int64(80*lvl) || card.Def >= int64(80*lvl) {
		return 1
	} else if card.Atk >= int64(60*lvl) || card.Def >= int64(60*lvl) {
		return 1
	} else if card.Atk >= int64(30*lvl) || card.Def >= int64(30*lvl) {
		return 0
	}
	return 0
}
