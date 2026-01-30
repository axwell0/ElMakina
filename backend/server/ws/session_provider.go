package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"ElMakina/backend/actions"
	"ElMakina/backend/engine"
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
	"ElMakina/backend/server"
	"ElMakina/backend/server/replay"
)

type responseEnvelope struct {
	sender int
	env    Envelope
}

type actionWindow struct {
	requestID string
	actor     int
	allowed   []models.ActionID
}

type challengeWindowState struct {
	requestID          string
	actorIndex         int
	actionID           models.ActionID
	claimedRole        models.Role
	kind               engine.ChallengeKind
	targetIndex        *int
	allowedChallengers []int
	timeoutMs          int64
}

type counterWindowState struct {
	requestID      string
	actorIndex     int
	actionID       models.ActionID
	allowedActions []models.ActionID
	allowedPlayers []int
	targetIndex    *int
	timeoutMs      int64
}

type stepWindowState struct {
	requestID string
	req       state.StepRequest
}

type sessionProvider struct {
	mu       sync.RWMutex
	pending  map[string]chan responseEnvelope
	players  map[int]*clientConn
	nextID   uint64
	errCh    chan error
	game     *engine.Game
	cfg      engine.TurnConfig
	clock    engine.Clock
	pauseMu  sync.Mutex
	paused   bool
	pauseCh  chan struct{}
	resumeCh chan struct{}
	events   *eventRecorder
	promptMu sync.Mutex
	action   *actionWindow
	chal     *challengeWindowState
	counter  *counterWindowState
	step     *stepWindowState
}

func newSessionProvider(session *server.GameSession, cfg engine.TurnConfig, clock engine.Clock, events *eventRecorder) *sessionProvider {
	players := make(map[int]*clientConn, len(session.IndexByPlayer))
	return &sessionProvider{
		pending: make(map[string]chan responseEnvelope),
		players: players,
		errCh:   make(chan error, 8),
		game:    session.Game,
		cfg:     cfg,
		clock:   clock,
		pauseCh: make(chan struct{}),
		events:  events,
	}
}

func (p *sessionProvider) Errors() <-chan error { return p.errCh }

func (p *sessionProvider) setActionWindow(reqID string, actor int, allowed []models.ActionID) {
	p.promptMu.Lock()
	p.action = &actionWindow{requestID: reqID, actor: actor, allowed: allowed}
	p.promptMu.Unlock()
}

func (p *sessionProvider) clearActionWindow(reqID string) {
	p.promptMu.Lock()
	if p.action != nil && p.action.requestID == reqID {
		p.action = nil
	}
	p.promptMu.Unlock()
}

func (p *sessionProvider) setChallengeWindow(reqID string, window engine.ChallengeWindow, timeout time.Duration) {
	p.promptMu.Lock()
	p.chal = &challengeWindowState{
		requestID:          reqID,
		actorIndex:         window.ActorIndex,
		actionID:           window.Action.ID,
		claimedRole:        window.ClaimedRole,
		kind:               window.Kind,
		targetIndex:        targetIndexFromAction(window.Action),
		allowedChallengers: window.AllowedChallengers,
		timeoutMs:          timeout.Milliseconds(),
	}
	p.promptMu.Unlock()
}

func (p *sessionProvider) clearChallengeWindow(reqID string) {
	p.promptMu.Lock()
	if p.chal != nil && p.chal.requestID == reqID {
		p.chal = nil
	}
	p.promptMu.Unlock()
}

func (p *sessionProvider) setCounterWindow(reqID string, window engine.CounterWindow, timeout time.Duration) {
	p.promptMu.Lock()
	p.counter = &counterWindowState{
		requestID:      reqID,
		actorIndex:     window.MainAction.SourceIndex,
		actionID:       window.MainAction.ID,
		allowedActions: window.AllowedActions,
		allowedPlayers: window.AllowedPlayers,
		targetIndex:    targetIndexFromAction(window.MainAction),
		timeoutMs:      timeout.Milliseconds(),
	}
	p.promptMu.Unlock()
}

func (p *sessionProvider) clearCounterWindow(reqID string) {
	p.promptMu.Lock()
	if p.counter != nil && p.counter.requestID == reqID {
		p.counter = nil
	}
	p.promptMu.Unlock()
}

func (p *sessionProvider) setStepWindow(reqID string, req state.StepRequest) {
	p.promptMu.Lock()
	p.step = &stepWindowState{requestID: reqID, req: req}
	p.promptMu.Unlock()
}

func (p *sessionProvider) clearStepWindow(reqID string) {
	p.promptMu.Lock()
	if p.step != nil && p.step.requestID == reqID {
		p.step = nil
	}
	p.promptMu.Unlock()
}

func (p *sessionProvider) resendPrompts(player int) {
	if p == nil {
		return
	}
	p.promptMu.Lock()
	action := p.action
	chal := p.chal
	counter := p.counter
	step := p.step
	p.promptMu.Unlock()

	if action != nil && action.actor == player {
		_ = p.send(player, Envelope{
			Type:      MsgRequestAction,
			RequestID: action.requestID,
			Payload: mustJSON(RequestActionPayload{
				ActorIndex:    action.actor,
				AllowedAction: action.allowed,
				Prompt:        "choose your action",
			}),
		})
	}

	if chal != nil {
		eligible := isAllowedPlayer(chal.allowedChallengers, player)
		_ = p.send(player, Envelope{
			Type:      MsgChallengeWindow,
			RequestID: chal.requestID,
			Payload: mustJSON(ChallengeWindowPayload{
				ActorIndex:  chal.actorIndex,
				ActionID:    string(chal.actionID),
				ClaimedRole: string(chal.claimedRole),
				Kind:        string(chal.kind),
				TargetIndex: chal.targetIndex,
				Eligible:    eligible,
				TimeoutMs:   chal.timeoutMs,
				Prompt:      "challenge or pass",
			}),
		})
	}

	if counter != nil {
		eligible := isAllowedPlayer(counter.allowedPlayers, player)
		allowed := []models.ActionID{}
		if eligible {
			if p.game != nil {
				allowed = affordableActions(&p.game.CurrentState, player, counter.allowedActions)
			} else {
				allowed = append([]models.ActionID{}, counter.allowedActions...)
			}
			eligible = len(allowed) > 0
		}
		_ = p.send(player, Envelope{
			Type:      MsgCounterWindow,
			RequestID: counter.requestID,
			Payload: mustJSON(CounterWindowPayload{
				ActorIndex:    counter.actorIndex,
				ActionID:      string(counter.actionID),
				AllowedAction: allowed,
				TargetIndex:   counter.targetIndex,
				Eligible:      eligible,
				TimeoutMs:     counter.timeoutMs,
				Prompt:        "play a counter or pass",
			}),
		})
	}

	if step != nil && step.req.Actor == player {
		_ = p.send(player, Envelope{
			Type:      MsgRequestStep,
			RequestID: step.requestID,
			Payload: mustJSON(RequestStepPayload{
				Step:   step.req,
				Prompt: "select cards",
			}),
		})
	}
}

func (p *sessionProvider) recordEvent(ctx context.Context, eventType string, visibility string, playerIndex *int, payload any) {
	if p.events == nil {
		return
	}
	var playerID *string
	if playerIndex != nil {
		playerID = p.events.playerIDForIndex(*playerIndex)
	}
	p.events.record(ctx, eventType, visibility, playerID, payload)
}

func (p *sessionProvider) RequestAction(ctx context.Context, actor int, gs *state.GameState) (*models.PlayerAction, error) {
	if err := p.waitIfPaused(ctx); err != nil {
		return nil, err
	}
	reqID, ch := p.register()
	allowed := allowedMainActions(gs, actor)
	p.setActionWindow(reqID, actor, allowed)
	defer p.clearActionWindow(reqID)
	p.recordEvent(ctx, "request_action", replay.VisibilityPublic, nil, map[string]any{
		"request_id":      reqID,
		"actor_index":     actor,
		"allowed_actions": allowed,
	})
	if err := p.send(actor, Envelope{
		Type:      MsgRequestAction,
		RequestID: reqID,
		Payload: mustJSON(RequestActionPayload{
			ActorIndex:    actor,
			AllowedAction: allowed,
			Prompt:        "choose your action",
		}),
	}); err != nil {
		p.unregister(reqID)
		return nil, err
	}

	timerActive := p.cfg.TurnTimeout > 0
	var timeout <-chan time.Time
	if timerActive && p.clock != nil {
		timeout = p.clock.After(p.cfg.TurnTimeout)
		p.broadcastTurnTimer("start", actor, gs.TurnNumber, p.cfg.TurnTimeout)
	}
	stopped := false
	stopTimer := func() {
		if !timerActive || stopped {
			return
		}
		stopped = true
		p.broadcastTurnTimer("stop", actor, gs.TurnNumber, p.cfg.TurnTimeout)
	}

	for {
		select {
		case <-ctx.Done():
			p.unregister(reqID)
			stopTimer()
			return nil, ctx.Err()
		case <-timeout:
			p.unregister(reqID)
			stopTimer()
			_ = p.send(actor, Envelope{
				Type:      MsgPromptClosed,
				RequestID: reqID,
				Payload:   mustJSON(PromptClosedPayload{Reason: "turn_timeout"}),
			})
			return &models.PlayerAction{ID: models.Pass, SourceIndex: actor}, nil
		case err := <-p.errCh:
			if err != nil {
				p.unregister(reqID)
				stopTimer()
				return nil, err
			}
		case resp := <-ch:
			p.unregister(reqID)
			if resp.sender != actor {
				stopTimer()
				return nil, fmt.Errorf("unexpected action sender %d", resp.sender)
			}
			if resp.env.Type != MsgAction {
				stopTimer()
				return nil, fmt.Errorf("unexpected action response %q", resp.env.Type)
			}
			cmd, err := parseActionPayload(resp.env.Payload)
			if err == nil {
				p.recordEvent(ctx, "action_submitted", replay.VisibilityPublic, nil, map[string]any{
					"request_id": reqID,
					"action":     buildActionPayload(cmd),
				})
			}
			_ = p.send(actor, Envelope{
				Type:      MsgPromptClosed,
				RequestID: reqID,
				Payload:   mustJSON(PromptClosedPayload{Reason: "action_submitted"}),
			})
			stopTimer()
			return cmd, err
		}
	}
}

func (p *sessionProvider) ChallengeResponses(ctx context.Context, window engine.ChallengeWindow) (<-chan engine.ChallengeDecision, error) {
	if err := p.waitIfPaused(ctx); err != nil {
		return nil, err
	}
	turnNumber := 0
	if p.game != nil {
		turnNumber = p.game.CurrentState.TurnNumber
	}
	p.broadcastTurnTimer("pause", window.ActorIndex, turnNumber, p.cfg.TurnTimeout)
	reqID, ch := p.register()
	p.setChallengeWindow(reqID, window, p.cfg.ChallengeTimeout)
	playerIndices := p.playerIndices()
	p.recordEvent(ctx, "challenge_window", replay.VisibilityPublic, nil, map[string]any{
		"request_id":   reqID,
		"actor_index":  window.ActorIndex,
		"action_id":    string(window.Action.ID),
		"claimed_role": string(window.ClaimedRole),
		"kind":         string(window.Kind),
		"target_index": targetIndexFromAction(window.Action),
		"eligible":     window.AllowedChallengers,
		"timeout_ms":   p.cfg.ChallengeTimeout.Milliseconds(),
	})
	for _, player := range playerIndices {
		isEligible := isAllowedPlayer(window.AllowedChallengers, player)
		if err := p.send(player, Envelope{
			Type:      MsgChallengeWindow,
			RequestID: reqID,
			Payload: mustJSON(ChallengeWindowPayload{
				ActorIndex:  window.ActorIndex,
				ActionID:    string(window.Action.ID),
				ClaimedRole: string(window.ClaimedRole),
				Kind:        string(window.Kind),
				TargetIndex: targetIndexFromAction(window.Action),
				Eligible:    isEligible,
				TimeoutMs:   p.cfg.ChallengeTimeout.Milliseconds(),
				Prompt:      "challenge or pass",
			}),
		}); err != nil {
			p.unregister(reqID)
			return nil, err
		}
	}

	out := make(chan engine.ChallengeDecision)
	go func() {
		defer close(out)
		defer p.unregister(reqID)
		defer p.clearChallengeWindow(reqID)
		defer p.broadcastTurnTimer("resume", window.ActorIndex, turnNumber, p.cfg.TurnTimeout)
		defer func() {
			for _, player := range playerIndices {
				_ = p.send(player, Envelope{
					Type:      MsgPromptClosed,
					RequestID: reqID,
					Payload:   mustJSON(PromptClosedPayload{Reason: "window_closed"}),
				})
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-p.errCh:
				if err != nil {
					return
				}
			case resp := <-ch:
				if resp.env.Type != MsgChallenge {
					p.notifyError(fmt.Errorf("unexpected challenge response %q", resp.env.Type))
					return
				}
				if !isAllowedPlayer(window.AllowedChallengers, resp.sender) {
					continue
				}
				decision, err := parseChallengePayload(resp.env.Payload)
				if err != nil {
					p.notifyError(err)
					return
				}
				p.recordEvent(ctx, "challenge_response", replay.VisibilityPublic, nil, map[string]any{
					"request_id":               reqID,
					"challenger_index":         decision.ChallengerIndex,
					"pass":                     decision.Pass,
					"challenger_discard_index": decision.ChallengerDiscardIndex,
					"actor_discard_index":      decision.ActorDiscardIndex,
					"actor_proving_card_index": decision.ActorProvingCardIndex,
				})
				out <- decision
			}
		}
	}()

	return out, nil
}

func (p *sessionProvider) CounterResponses(ctx context.Context, window engine.CounterWindow) (<-chan engine.CounterDecision, error) {
	if err := p.waitIfPaused(ctx); err != nil {
		return nil, err
	}
	turnNumber := 0
	if p.game != nil {
		turnNumber = p.game.CurrentState.TurnNumber
	}
	p.broadcastTurnTimer("pause", window.MainAction.SourceIndex, turnNumber, p.cfg.TurnTimeout)
	perPlayer := make(map[int][]models.ActionID, len(window.AllowedPlayers))
	var gs *state.GameState
	if p.game != nil {
		gs = &p.game.CurrentState
	}
	for _, player := range window.AllowedPlayers {
		allowed := affordableActions(gs, player, window.AllowedActions)
		if len(allowed) == 0 {
			continue
		}
		perPlayer[player] = allowed
	}
	if len(perPlayer) == 0 {
		out := make(chan engine.CounterDecision)
		close(out)
		return out, nil
	}
	reqID, ch := p.register()
	p.setCounterWindow(reqID, window, p.cfg.CounterTimeout)
	p.recordEvent(ctx, "counter_window", replay.VisibilityPublic, nil, map[string]any{
		"request_id":      reqID,
		"actor_index":     window.MainAction.SourceIndex,
		"action_id":       string(window.MainAction.ID),
		"allowed_actions": window.AllowedActions,
		"eligible":        window.AllowedPlayers,
		"timeout_ms":      p.cfg.CounterTimeout.Milliseconds(),
	})
	playerIndices := p.playerIndices()
	for _, player := range playerIndices {
		allowed := perPlayer[player]
		isEligible := len(allowed) > 0
		if err := p.send(player, Envelope{
			Type:      MsgCounterWindow,
			RequestID: reqID,
			Payload: mustJSON(CounterWindowPayload{
				ActorIndex:    window.MainAction.SourceIndex,
				ActionID:      string(window.MainAction.ID),
				AllowedAction: allowed,
				TargetIndex:   targetIndexFromAction(window.MainAction),
				Eligible:      isEligible,
				TimeoutMs:     p.cfg.CounterTimeout.Milliseconds(),
				Prompt:        "play a counter or pass",
			}),
		}); err != nil {
			p.unregister(reqID)
			return nil, err
		}
	}
	out := make(chan engine.CounterDecision)
	go func() {
		defer close(out)
		defer p.unregister(reqID)
		defer p.clearCounterWindow(reqID)
		defer p.broadcastTurnTimer("resume", window.MainAction.SourceIndex, turnNumber, p.cfg.TurnTimeout)
		defer func() {
			for _, player := range playerIndices {
				_ = p.send(player, Envelope{
					Type:      MsgPromptClosed,
					RequestID: reqID,
					Payload:   mustJSON(PromptClosedPayload{Reason: "window_closed"}),
				})
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-p.errCh:
				if err != nil {
					return
				}
			case resp := <-ch:
				if resp.env.Type != MsgCounter {
					p.notifyError(fmt.Errorf("unexpected counter response %q", resp.env.Type))
					return
				}
				if _, ok := perPlayer[resp.sender]; !ok {
					continue
				}
				decision, err := parseCounterPayload(resp.env.Payload)
				if err != nil {
					p.notifyError(err)
					return
				}
				if decision.Pass {
					p.recordEvent(ctx, "counter_response", replay.VisibilityPublic, nil, map[string]any{
						"request_id":   reqID,
						"player_index": resp.sender,
						"pass":         true,
					})
				} else if decision.Command != nil {
					p.recordEvent(ctx, "counter_response", replay.VisibilityPublic, nil, map[string]any{
						"request_id": reqID,
						"action":     buildActionPayload(decision.Command),
					})
				}
				if decision.Pass {
					decision.PlayerIndex = resp.sender
					out <- decision
					continue
				}
				if decision.Command != nil && decision.Command.SourceIndex != resp.sender {
					p.notifyError(fmt.Errorf("counter sender %d does not match payload %d", resp.sender, decision.Command.SourceIndex))
					return
				}
				out <- decision
			}
		}
	}()
	return out, nil
}

func (p *sessionProvider) RequestStep(ctx context.Context, req state.StepRequest) (any, error) {
	if err := p.waitIfPaused(ctx); err != nil {
		return nil, err
	}
	reqID, ch := p.register()
	actor := req.Actor
	p.setStepWindow(reqID, req)
	defer p.clearStepWindow(reqID)
	p.recordEvent(ctx, "step_request", replay.VisibilityPrivate, &actor, map[string]any{
		"request_id": reqID,
		"action_id":  req.ActionID,
		"kind":       req.Kind,
		"context":    req.Context,
		"count":      req.Count,
		"options":    req.Options,
	})
	if err := p.send(req.Actor, Envelope{
		Type:      MsgRequestStep,
		RequestID: reqID,
		Payload: mustJSON(RequestStepPayload{
			Step:   req,
			Prompt: "select cards",
		}),
	}); err != nil {
		p.unregister(reqID)
		return nil, err
	}

	for {
		select {
		case <-ctx.Done():
			p.unregister(reqID)
			return nil, ctx.Err()
		case err := <-p.errCh:
			if err != nil {
				p.unregister(reqID)
				return nil, err
			}
		case resp := <-ch:
			p.unregister(reqID)
			if resp.sender != req.Actor {
				return nil, fmt.Errorf("unexpected step sender %d", resp.sender)
			}
			if resp.env.Type != MsgStepResult {
				return nil, fmt.Errorf("unexpected step response %q", resp.env.Type)
			}
			result, err := parseStepPayload(req, resp.env.Payload)
			if err == nil {
				p.recordEvent(ctx, "step_response", replay.VisibilityPrivate, &actor, map[string]any{
					"request_id": reqID,
					"result":     result,
				})
			}
			_ = p.send(req.Actor, Envelope{
				Type:      MsgPromptClosed,
				RequestID: reqID,
				Payload:   mustJSON(PromptClosedPayload{Reason: "step_submitted"}),
			})
			return result, err
		}
	}
}

func (p *sessionProvider) dispatch(sender int, env Envelope) bool {
	p.mu.RLock()
	ch := p.pending[env.RequestID]
	p.mu.RUnlock()
	if ch == nil {
		return false
	}

	// If the session is paused, buffer responses by blocking dispatch until resume.
	// Note: callers that must remain responsive (e.g. sessionRunner's message loop)
	// should avoid calling this directly while paused; see hasPending + async dispatch.
	for p.isPaused() {
		p.pauseMu.Lock()
		resumeCh := p.resumeCh
		p.pauseMu.Unlock()
		if resumeCh == nil {
			break
		}
		<-resumeCh
	}

	select {
	case ch <- responseEnvelope{sender: sender, env: env}:
	default:
		p.notifyError(fmt.Errorf("response buffer full for %s", env.RequestID))
	}
	return true
}

func (p *sessionProvider) hasPending(requestID string) bool {
	p.mu.RLock()
	_, ok := p.pending[requestID]
	p.mu.RUnlock()
	return ok
}

func (p *sessionProvider) register() (string, chan responseEnvelope) {
	id := p.nextRequestID()
	ch := make(chan responseEnvelope, 16)
	p.mu.Lock()
	p.pending[id] = ch
	p.mu.Unlock()
	return id, ch
}

func (p *sessionProvider) unregister(id string) {
	p.mu.Lock()
	delete(p.pending, id)
	p.mu.Unlock()
}

func (p *sessionProvider) send(player int, env Envelope) error {
	p.mu.RLock()
	client := p.players[player]
	p.mu.RUnlock()
	if client == nil {
		return fmt.Errorf("player %d not connected", player)
	}
	client.sendMu.Lock()
	defer client.sendMu.Unlock()
	return client.conn.WriteJSON(env)
}

func (p *sessionProvider) broadcast(env Envelope) {
	p.mu.RLock()
	clients := make([]*clientConn, 0, len(p.players))
	for _, client := range p.players {
		if client != nil {
			clients = append(clients, client)
		}
	}
	p.mu.RUnlock()

	for _, client := range clients {
		client.sendMu.Lock()
		_ = client.conn.WriteJSON(env)
		client.sendMu.Unlock()
	}
}

func (p *sessionProvider) broadcastTurnTimer(state string, actor int, turn int, duration time.Duration) {
	if duration <= 0 {
		return
	}
	p.broadcast(Envelope{
		Type: MsgTurnTimer,
		Payload: mustJSON(TurnTimerPayload{
			ActivePlayerIndex: actor,
			TurnNumber:        turn,
			DurationMs:        duration.Milliseconds(),
			State:             state,
		}),
	})
}

func (p *sessionProvider) notifyError(err error) {
	select {
	case p.errCh <- err:
	default:
	}
}

func (p *sessionProvider) Pause() {
	p.pauseMu.Lock()
	defer p.pauseMu.Unlock()
	if p.paused {
		return
	}
	p.paused = true
	close(p.pauseCh)
	p.resumeCh = make(chan struct{})
}

func (p *sessionProvider) Resume() {
	p.pauseMu.Lock()
	defer p.pauseMu.Unlock()
	if !p.paused {
		return
	}
	p.paused = false
	if p.resumeCh != nil {
		close(p.resumeCh)
		p.resumeCh = nil
	}
	p.pauseCh = make(chan struct{})
}

func (p *sessionProvider) isPaused() bool {
	p.pauseMu.Lock()
	defer p.pauseMu.Unlock()
	return p.paused
}

func (p *sessionProvider) waitIfPaused(ctx context.Context) error {
	p.pauseMu.Lock()
	if !p.paused {
		p.pauseMu.Unlock()
		return nil
	}
	resumeCh := p.resumeCh
	p.pauseMu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-resumeCh:
		return nil
	}
}

func (p *sessionProvider) attach(index int, client *clientConn) {
	p.mu.Lock()
	p.players[index] = client
	p.mu.Unlock()
}

func (p *sessionProvider) detach(index int, client *clientConn) {
	p.mu.Lock()
	current := p.players[index]
	if client == nil || current == nil || current == client {
		delete(p.players, index)
	}
	p.mu.Unlock()
}

func (p *sessionProvider) nextRequestID() string {
	id := atomic.AddUint64(&p.nextID, 1)
	return fmt.Sprintf("req-%d", id)
}

func allowedMainActions(gs *state.GameState, actor int) []models.ActionID {
	ids := make([]models.ActionID, 0)
	for id, def := range actions.ActionRegistry {
		if def.Kind != models.MainAction {
			continue
		}
		if id == models.Pass {
			continue
		}
		ids = append(ids, id)
	}
	allowed := affordableActions(gs, actor, ids)
	return eligibleMainActions(gs, actor, allowed)
}

func affordableActions(gs *state.GameState, actor int, ids []models.ActionID) []models.ActionID {
	if gs == nil || actor < 0 || actor >= len(gs.Players) {
		return ids
	}
	coins := gs.Players[actor].Coins
	allowed := make([]models.ActionID, 0, len(ids))
	for _, id := range ids {
		cost := actions.ActionCost(id)
		if cost > 0 && coins < cost {
			continue
		}
		allowed = append(allowed, id)
	}
	return allowed
}

func eligibleMainActions(gs *state.GameState, actor int, ids []models.ActionID) []models.ActionID {
	if gs == nil || actor < 0 || actor >= len(gs.Players) {
		return ids
	}
	hasTarget := hasEligibleTarget(gs, actor, 0)
	hasStealTarget := hasEligibleTarget(gs, actor, 2)
	hasTaxable := hasTaxablePlayer(gs)
	allowed := make([]models.ActionID, 0, len(ids))
	for _, id := range ids {
		switch id {
		case models.Coup, models.Accuse, models.Assassinate, models.Investigate:
			if !hasTarget {
				continue
			}
		case models.Steal:
			if !hasStealTarget {
				continue
			}
		case models.Tax:
			if !hasTaxable {
				continue
			}
		}
		allowed = append(allowed, id)
	}
	return allowed
}

func hasEligibleTarget(gs *state.GameState, actor int, minCoins int) bool {
	for i, player := range gs.Players {
		if i == actor {
			continue
		}
		if len(player.Hand) == 0 {
			continue
		}
		if minCoins > 0 && player.Coins < minCoins {
			continue
		}
		return true
	}
	return false
}

func hasTaxablePlayer(gs *state.GameState) bool {
	for _, player := range gs.Players {
		if len(player.Hand) == 0 {
			continue
		}
		if player.Coins >= 7 {
			return true
		}
	}
	return false
}

func parseActionPayload(raw json.RawMessage) (*models.PlayerAction, error) {
	var payload ActionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	if payload.Pass {
		return nil, fmt.Errorf("pass not allowed for actions")
	}
	if payload.ID == "" {
		return nil, fmt.Errorf("missing action id")
	}
	if payload.SourceIndex == nil {
		return nil, fmt.Errorf("missing source_index")
	}
	if payload.Guess != nil && payload.TargetIndex == nil {
		return nil, fmt.Errorf("guess requires target_index")
	}
	cmd := &models.PlayerAction{
		ID:          payload.ID,
		SourceIndex: *payload.SourceIndex,
	}
	if payload.MainAction != nil {
		cmd.MainAction = *payload.MainAction
	}
	if payload.TargetIndex != nil && payload.Guess != nil {
		role, err := models.ParseRole(*payload.Guess)
		if err != nil {
			return nil, err
		}
		cmd.Payload = models.AccusePayload{TargetIndex: *payload.TargetIndex, Guess: role}
	} else if payload.TargetIndex != nil {
		cmd.Payload = models.TargetPayload{TargetIndex: *payload.TargetIndex}
	}
	return cmd, nil
}

func parseCounterPayload(raw json.RawMessage) (engine.CounterDecision, error) {
	var payload ActionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return engine.CounterDecision{}, err
	}
	if payload.Pass {
		return engine.CounterDecision{Pass: true}, nil
	}
	if payload.ID == "" {
		return engine.CounterDecision{}, fmt.Errorf("missing action id")
	}
	if payload.SourceIndex == nil {
		return engine.CounterDecision{}, fmt.Errorf("missing source_index")
	}
	if payload.Guess != nil && payload.TargetIndex == nil {
		return engine.CounterDecision{}, fmt.Errorf("guess requires target_index")
	}
	cmd := &models.PlayerAction{
		ID:          payload.ID,
		SourceIndex: *payload.SourceIndex,
	}
	if payload.MainAction != nil {
		cmd.MainAction = *payload.MainAction
	}
	if payload.TargetIndex != nil && payload.Guess != nil {
		role, err := models.ParseRole(*payload.Guess)
		if err != nil {
			return engine.CounterDecision{}, err
		}
		cmd.Payload = models.AccusePayload{TargetIndex: *payload.TargetIndex, Guess: role}
	} else if payload.TargetIndex != nil {
		cmd.Payload = models.TargetPayload{TargetIndex: *payload.TargetIndex}
	}
	return engine.CounterDecision{Command: cmd}, nil
}

func parseChallengePayload(raw json.RawMessage) (engine.ChallengeDecision, error) {
	var payload ChallengePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return engine.ChallengeDecision{}, err
	}
	if payload.ChallengerIndex == nil {
		return engine.ChallengeDecision{}, fmt.Errorf("missing challenger_index")
	}
	return engine.ChallengeDecision{
		ChallengerIndex:        *payload.ChallengerIndex,
		ChallengerDiscardIndex: payload.ChallengerDiscardIndex,
		ActorDiscardIndex:      payload.ActorDiscardIndex,
		ActorProvingCardIndex:  payload.ActorProvingCardIndex,
		Pass:                   payload.Pass,
	}, nil
}

func parseStepPayload(req state.StepRequest, raw json.RawMessage) (any, error) {
	if req.Kind == state.StepCardSelect {
		var indices []int
		if err := json.Unmarshal(raw, &indices); err != nil {
			return nil, err
		}
		return indices, nil
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func targetIndexFromAction(cmd *models.PlayerAction) *int {
	if cmd == nil {
		return nil
	}
	switch payload := cmd.Payload.(type) {
	case models.TargetPayload:
		return &payload.TargetIndex
	case *models.TargetPayload:
		if payload == nil {
			return nil
		}
		return &payload.TargetIndex
	case models.AccusePayload:
		return &payload.TargetIndex
	case *models.AccusePayload:
		if payload == nil {
			return nil
		}
		return &payload.TargetIndex
	default:
		return nil
	}
}

func isAllowedPlayer(players []int, candidate int) bool {
	for _, player := range players {
		if player == candidate {
			return true
		}
	}
	return false
}

func (p *sessionProvider) playerIndices() []int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	indices := make([]int, 0, len(p.players))
	for idx := range p.players {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	return indices
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
