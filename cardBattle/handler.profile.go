package cardBattle

import "io"

func (c *CardBattleServer) CardBattleProfileStream(stream CardBattleService_CardBattleProfileStreamServer) error {

	for {

		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		switch evt := msg.Event.(type) {

		case *ProfileStream_OnePlayerWithCards:

			// check player if exist
			if player, isExist := c.Players[evt.OnePlayerWithCards.Owner.Id]; isExist {

				// send back to client
				err := stream.Send(&ProfileStream{
					Event: &ProfileStream_OnePlayerWithCards{
						OnePlayerWithCards: player,
					},
				})

				if err != nil {
					return err
				}
			}

		case *ProfileStream_AddCardToDeck:

			if player, isExist := c.Players[evt.AddCardToDeck.Client.Id]; isExist {

				if player.Owner.MaxDeckSlot > int32(len(player.Deck)) {

					newDeck := []*Card{}
					cardTarget := &Card{}
					for _, card := range player.Reserve {
						if card.Id != evt.AddCardToDeck.CardData.Id {
							newDeck = append(newDeck, card)
						} else {
							cardTarget = card
						}
					}

					player.Reserve = newDeck
					player.Deck = append(player.Deck, cardTarget)

				}

				// send back to client
				err := stream.Send(&ProfileStream{
					Event: evt,
				})

				if err != nil {
					return err
				}
			}

		case *ProfileStream_RemoveCardFromDeck:

			if player, isExist := c.Players[evt.RemoveCardFromDeck.Client.Id]; isExist {

				if player.Owner.MaxReserveSlot > int32(len(player.Reserve)) {

					newDeck := []*Card{}
					cardTarget := &Card{}
					for _, card := range player.Deck {
						if card.Id != evt.RemoveCardFromDeck.CardData.Id {
							newDeck = append(newDeck, card)
						} else {
							cardTarget = card
						}
					}

					player.Deck = newDeck
					player.Reserve = append(player.Reserve, cardTarget)

				}

				// send back to client
				err := stream.Send(&ProfileStream{
					Event: evt,
				})

				if err != nil {
					return err
				}
			}

		case *ProfileStream_ExcMessage:

			// just left this empty

		default:
		}
	}

}
