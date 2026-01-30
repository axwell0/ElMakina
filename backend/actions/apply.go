package actions

import (
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
	"fmt"
	"math/rand"
)

// This file is the "Rulebook" translated into Go.
//
// While Engine (in `backend/engine`) handles the FLOW of the game (turns,
// challenges, counts), this file handles the PAYLOAD of those actions.
//
// Every function here (`applyBusiness`, `applySteal`, etc.) follows a strict pattern:
// it takes the current GameState and a Command, and it performs the atomic
// mutation required by that rule.
//
// Separating this from the Engine is crucial because it keeps the "Mechanics"
// (how a Coup works) away from the "Orchestration" (how a Turn works).
// If we wanted to change the cost of a Coup, or add a new role, we mostly
// just touch things in the `/actions` or `/models` folders, leaving the
// high-level turn logic completely untouched.
//
// ACTIONFLOW INTEGRATION:
// Most actions are simple and execute atomically (e.g., Income, ForeignAid).
// However, some actions require mid-action player input:
//   - Exchange: Draw cards → [WAIT] Player selects cards to return → Return cards
//   - Coup/Assassinate: Pay cost → [WAIT] Target selects card to discard → Discard
//   - Investigate: [WAIT] Player selects card to reveal → Reveal card
//
// These actions use the ActionFlow mechanism (see flow.go) to pause execution
// and request input. The action calls flow.Yield() or flow.RequestCardSelection()
// which blocks until the engine provides the player's response via FlowRunner.Next().
// This allows multi-step actions to be written as linear, readable code instead
// of splitting them into multiple state machines.
var randIntn = rand.Intn

func applyBusiness(_ *ActionFlow, s *state.GameState, _ *state.TurnState, log *state.TurnLog, playerAction *models.PlayerAction) error {
	source := &s.Players[playerAction.SourceIndex]
	if _, err := s.AddCoins(playerAction.SourceIndex, 4); err != nil {
		return err
	}
	log.Logf("%s takes 4 coins", source.Name)
	logCoins(log, source.Name, 4)
	return nil
}

func applyTax(_ *ActionFlow, s *state.GameState, _ *state.TurnState, log *state.TurnLog, playerAction *models.PlayerAction) error {
	source := &s.Players[playerAction.SourceIndex]

	// Normal Tax action: apply global 7+ coin tax (no +3 gain)
	taxedCount := 0
	for i := range s.Players {
		if s.Players[i].Coins >= 7 && len(s.Players[i].Hand) > 0 {
			if _, err := s.AddCoins(i, -1); err != nil {
				return err
			}
			taxedCount++
			logCoins(log, s.Players[i].Name, -1)
		}
	}
	if taxedCount > 0 {
		log.Logf("%s triggers global tax: %d player(s) with 7+ coins pay 1 coin", source.Name, taxedCount)
	} else {
		log.Logf("%s triggers global tax: no players with 7+ coins", source.Name)
	}
	return nil
}

// applyTaxBusinessWoman is a counter-action that taxes the Businesswoman
// 1 coin when she takes her 4-coin action. Each TaxCollector can do this once
// per turn, with a maximum of 4 total TaxBusinessWoman actions per turn.
func applyTaxBusinessWoman(_ *ActionFlow, s *state.GameState, turn *state.TurnState, log *state.TurnLog, playerAction *models.PlayerAction) error {
	source := &s.Players[playerAction.SourceIndex]

	if turn.TaxBusinessWomanCount >= 4 {
		log.Logf("%s attempts to tax but the limit is reached", source.Name)
		return nil
	}
	turn.TaxBusinessWomanCount++
	if _, err := s.AddCoins(s.CurrentPlayerIndex, -1); err != nil {
		return err
	}
	if _, err := s.AddCoins(playerAction.SourceIndex, 1); err != nil {
		return err
	}
	log.Logf("%s taxes %s for 1 coin", source.Name, s.Players[s.CurrentPlayerIndex].Name)
	logCoins(log, s.Players[s.CurrentPlayerIndex].Name, -1)
	logCoins(log, source.Name, 1)
	return nil
}

func applyBlock(_ *ActionFlow, s *state.GameState, _ *state.TurnState, log *state.TurnLog, playerAction *models.PlayerAction) error {
	source := &s.Players[playerAction.SourceIndex]
	if playerAction.MainAction != "" {
		log.Logf("%s blocks %s", source.Name, playerAction.MainAction)
	} else {
		log.Logf("%s blocks the action", source.Name)
	}
	return nil
}

func applyIncome(_ *ActionFlow, s *state.GameState, _ *state.TurnState, log *state.TurnLog, playerAction *models.PlayerAction) error {
	_, err := s.AddCoins(playerAction.SourceIndex, 1)
	if err == nil {
		logCoins(log, s.Players[playerAction.SourceIndex].Name, 1)
	}
	return err
}

func applyForeignAid(_ *ActionFlow, s *state.GameState, _ *state.TurnState, log *state.TurnLog, playerAction *models.PlayerAction) error {
	_, err := s.AddCoins(playerAction.SourceIndex, 2)
	if err == nil {
		logCoins(log, s.Players[playerAction.SourceIndex].Name, 2)
	}
	return err
}

func applyCoup(flow *ActionFlow, s *state.GameState, _ *state.TurnState, log *state.TurnLog, playerAction *models.PlayerAction) error {
	source := &s.Players[playerAction.SourceIndex]
	payload, err := targetPayload(playerAction)
	if err != nil {
		return err
	}
	log.Logf("%s coups %s", source.Name, s.Players[payload.TargetIndex].Name)
	return requestAndDiscardOne(flow, s, log, playerAction, payload.TargetIndex, "coup")
}

func applyInvestigate(_ *ActionFlow, s *state.GameState, _ *state.TurnState, log *state.TurnLog, playerAction *models.PlayerAction) error {
	source := &s.Players[playerAction.SourceIndex]
	payload, err := targetPayload(playerAction)
	if err != nil {
		return err
	}
	target := &s.Players[payload.TargetIndex]

	if len(target.Hand) == 0 {
		return fmt.Errorf("target has no cards")
	}

	cardIndex := randIntn(len(target.Hand))
	card := target.Hand[cardIndex]
	log.PrivateLogf(source.ID, "You see %s has %s", target.Name, card.Role)
	log.Logf("%s investigates %s", source.Name, target.Name)
	return nil
}

func applyAccuse(_ *ActionFlow, s *state.GameState, _ *state.TurnState, log *state.TurnLog, playerAction *models.PlayerAction) error {
	source := &s.Players[playerAction.SourceIndex]
	payload, err := accusePayload(playerAction)
	if err != nil {
		return err
	}
	target := &s.Players[payload.TargetIndex]
	found := false
	for i, c := range target.Hand {
		if c.Role == payload.Guess {
			if _, err := s.DiscardFromHand(payload.TargetIndex, i); err != nil {
				return err
			}
			log.Logf("%s discards %s", target.Name, payload.Guess)
			if len(s.Players[payload.TargetIndex].Hand) == 0 {
				log.Logf("%s is eliminated", s.Players[payload.TargetIndex].Name)
			}
			found = true
			log.Logf("%s is correct! %s had %s", source.Name, target.Name, payload.Guess)
			break
		}
	}

	if !found {
		if _, err := s.AddCoins(payload.TargetIndex, 4); err != nil {
			return err
		}
		log.Logf("%s is wrong! %s gets 4 coins", source.Name, target.Name)
		logCoins(log, target.Name, 4)
	}
	return nil
}

func applyAssassinate(flow *ActionFlow, s *state.GameState, _ *state.TurnState, log *state.TurnLog, playerAction *models.PlayerAction) error {
	source := &s.Players[playerAction.SourceIndex]
	payload, err := targetPayload(playerAction)
	if err != nil {
		return err
	}
	log.Logf("%s assassinates %s", source.Name, s.Players[payload.TargetIndex].Name)
	return requestAndDiscardOne(flow, s, log, playerAction, payload.TargetIndex, "assassinate")
}

func applySteal(_ *ActionFlow, s *state.GameState, _ *state.TurnState, log *state.TurnLog, playerAction *models.PlayerAction) error {
	source := &s.Players[playerAction.SourceIndex]
	payload, err := targetPayload(playerAction)
	if err != nil {
		return err
	}
	target := &s.Players[payload.TargetIndex]

	if target.Coins < 2 {
		return fmt.Errorf("target does not have 2 coins")
	}

	if _, err := s.AddCoins(payload.TargetIndex, -2); err != nil {
		return err
	}
	if _, err := s.AddCoins(playerAction.SourceIndex, 2); err != nil {
		return err
	}
	log.Logf("%s steals 2 from %s", source.Name, target.Name)
	logCoins(log, target.Name, -2)
	logCoins(log, source.Name, 2)
	return nil
}

func applyExchange(flow *ActionFlow, s *state.GameState, _ *state.TurnState, log *state.TurnLog, playerAction *models.PlayerAction) error {
	source := &s.Players[playerAction.SourceIndex]
	handCount := len(source.Hand)
	if handCount == 0 {
		return fmt.Errorf("no cards to exchange")
	}

	if err := s.DrawCards(playerAction.SourceIndex, handCount); err != nil {
		return err
	}
	log.Logf("%s draws %d cards to exchange", source.Name, handCount)

	indices, err := flow.RequestCardSelection(playerAction.ID, playerAction.SourceIndex, handCount, state.ContextExchange, cardOptions(source.Hand))
	if err != nil {
		return err
	}

	if err := s.ReturnCardsToDeck(playerAction.SourceIndex, indices); err != nil {
		return err
	}

	log.Logf("%s finished exchange", source.Name)
	return nil
}

func applyEscape(_ *ActionFlow, s *state.GameState, _ *state.TurnState, log *state.TurnLog, playerAction *models.PlayerAction) error {
	log.Logf("%s pays 9 to escape", s.Players[playerAction.SourceIndex].Name)
	logCoins(log, s.Players[playerAction.SourceIndex].Name, -9)
	return nil
}

func applyPass(_ *ActionFlow, s *state.GameState, _ *state.TurnState, log *state.TurnLog, playerAction *models.PlayerAction) error {
	log.Logf("%s passes", s.Players[playerAction.SourceIndex].Name)
	return nil
}

func requestAndDiscardOne(flow *ActionFlow, s *state.GameState, log *state.TurnLog, playerAction *models.PlayerAction, targetIndex int, actionLabel string) error {
	indices, err := flow.RequestCardSelection(playerAction.ID, targetIndex, 1, state.ContextDiscard, cardOptions(s.Players[targetIndex].Hand))
	if err != nil {
		return err
	}
	if len(indices) != 1 {
		return fmt.Errorf("invalid response to %s: expected 1 card index", actionLabel)
	}

	discarded, err := s.DiscardFromHand(targetIndex, indices[0])
	if err != nil {
		return err
	}
	log.Logf("%s discards %s", s.Players[targetIndex].Name, discarded.Role)
	if len(s.Players[targetIndex].Hand) == 0 {
		log.Logf("%s is eliminated", s.Players[targetIndex].Name)
		if playerAction.ID == models.Coup || playerAction.ID == models.Assassinate {
			coins := s.Players[targetIndex].Coins
			if coins > 0 {
				sourceCoins := s.Players[playerAction.SourceIndex].Coins
				if _, err := s.AddCoins(targetIndex, -coins); err != nil {
					return err
				}
				newSourceCoins, err := s.AddCoins(playerAction.SourceIndex, coins)
				if err != nil {
					return err
				}
				gained := newSourceCoins - sourceCoins
				log.Logf("%s takes %d coins from %s", s.Players[playerAction.SourceIndex].Name, coins, s.Players[targetIndex].Name)
				if gained > 0 {
					logCoins(log, s.Players[playerAction.SourceIndex].Name, gained)
				}
				logCoins(log, s.Players[targetIndex].Name, -coins)
			}
		}
	}
	return nil
}

func cardOptions(hand []state.Card) []string {
	options := make([]string, 0, len(hand))
	for i, card := range hand {
		label := card.Role
		if label == "" {
			label = models.Role(fmt.Sprintf("Card %d", i+1))
		}
		options = append(options, string(label))
	}
	return options
}

func logCoins(log *state.TurnLog, name string, delta int) {
	if delta == 0 {
		return
	}
	if delta > 0 {
		log.Logf("%s gains %d coins", name, delta)
		return
	}
	log.Logf("%s loses %d coins", name, -delta)
}
