package ws

import (
	"ElMakina/backend/engine"
	"ElMakina/backend/models"
)

type counterResultPayload struct {
	Action    ActionPayload           `json:"action"`
	Applied   bool                    `json:"applied"`
	Canceled  bool                    `json:"canceled"`
	Challenge *engine.ChallengeResult `json:"challenge,omitempty"`
}

func buildActionPayload(cmd *models.PlayerAction) ActionPayload {
	if cmd == nil {
		return ActionPayload{}
	}
	payload := ActionPayload{
		ID:          cmd.ID,
		SourceIndex: &cmd.SourceIndex,
		Pass:        cmd.ID == models.Pass,
	}
	if cmd.MainAction != "" {
		main := cmd.MainAction
		payload.MainAction = &main
	}
	switch p := cmd.Payload.(type) {
	case models.TargetPayload:
		target := p.TargetIndex
		payload.TargetIndex = &target
	case *models.TargetPayload:
		if p != nil {
			target := p.TargetIndex
			payload.TargetIndex = &target
		}
	case models.AccusePayload:
		target := p.TargetIndex
		guess := string(p.Guess)
		payload.TargetIndex = &target
		payload.Guess = &guess
	case *models.AccusePayload:
		if p != nil {
			target := p.TargetIndex
			guess := string(p.Guess)
			payload.TargetIndex = &target
			payload.Guess = &guess
		}
	}
	return payload
}

func buildTurnResolvedPayload(turnNumber int, res *engine.TurnResult) map[string]any {
	if res == nil {
		return map[string]any{"turn_number": turnNumber}
	}
	counters := make([]counterResultPayload, 0, len(res.CounterResults))
	for _, counter := range res.CounterResults {
		counters = append(counters, counterResultPayload{
			Action:    buildActionPayload(counter.Action),
			Applied:   counter.Applied,
			Canceled:  counter.Canceled,
			Challenge: counter.Challenge,
		})
	}
	return map[string]any{
		"turn_number":       turnNumber,
		"main_action":       buildActionPayload(res.Main),
		"main_applied":      res.MainApplied,
		"main_canceled":     res.MainCanceled,
		"counter_results":   counters,
		"challenge_results": res.ChallengeResults,
		"logs":              res.Logs,
	}
}
