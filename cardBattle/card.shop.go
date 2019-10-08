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

type CardShop struct {
	Cards     map[string]*Card
	TotalCard int
	MinLevel  int
	MaxLevel  int
	Time      int
	MaxItem   int
}

func (s *CardBattleServer) RunCardShop() {

	go func(serv *CardBattleServer) {

		rand.Seed(time.Now().UnixNano())

		var s = serv.Shop
		var timeLeft int = 0

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
					serv.Lobby.Broadcast <- LobbyStream{
						Event: &LobbyStream_ShopRefresh{
							ShopRefresh: true,
						},
					}

					timeLeft = s.Time

				default:

					time.Sleep(1 * time.Second)
					timeLeft--

					// broadcast to all player
					// countdown for next shop item
					serv.Lobby.Broadcast <- LobbyStream{
						Event: &LobbyStream_ShopRefreshTime{
							ShopRefreshTime: int32(timeLeft),
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
			Image: fmt.Sprintf("%s.jpg", images[random(0, len(images))]),
			Name:  fmt.Sprintf(`%s %s`, firstName, lastName),
			Color: 0,
			Price: int32(random(minAtkDef, maxAtkDef)),
			Level: int32(random(minLevel, maxLevel)),
			Atk:   int32(random(minAtkDef, maxAtkDef)),
			Def:   int32(random(minAtkDef, maxAtkDef)),
		}

		if card.Atk >= 140 || card.Def >= 140 {
			card.Color = 4
		} else if card.Atk >= 120 || card.Def >= 120 {
			card.Color = 3
		} else if card.Atk >= 90 || card.Def >= 90 {
			card.Color = 2
		} else if card.Atk >= 80 || card.Def >= 80 {
			card.Color = 1
		} else if card.Atk >= 60 || card.Def >= 60 {
			card.Color = 1
		} else if card.Atk >= 30 || card.Def >= 30 {
			card.Color = 0
		}

		card.Price = card.Price * card.Level
		card.Atk = card.Atk * card.Level
		card.Def = card.Def * card.Level

		cards = append(cards, card)

	}

	return cards, nil
}
