//nolint:err113,wrapcheck,gocognit,cyclop,gocritic,revive // Legacy session orchestration; lint debt tracked for staged cleanup.
package session

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	coreactions "github.com/axwell0/elmakina/engine/core/actions"
	coreconfig "github.com/axwell0/elmakina/engine/core/config"
	corestate "github.com/axwell0/elmakina/engine/core/state"
	coretypes "github.com/axwell0/elmakina/engine/core/types"
	runtimeturn "github.com/axwell0/elmakina/engine/runtime/turn"
	turnmodel "github.com/axwell0/elmakina/engine/turn"
	"github.com/axwell0/elmakina/internal/apperrors"
)

type activeTurn struct {
	done   chan struct{}
	result *turnmodel.TurnResult
	err    error
}

// GameSession owns the mutable match state and turn execution for a lobby.
type GameSession struct {
	id     string
	state  corestate.GameState
	config coreconfig.GameConfig

	mu             sync.RWMutex
	activeTurn     *activeTurn
	lastTurnResult *turnmodel.TurnResult
	lastTurnErr    error
	revision       int64
	currentPrompt  *Prompt
	promptEvents   chan *Prompt

	mainAction chan *coretypes.PlayerAction
	challenge  chan runtimeturn.ChallengeDecision
	counter    chan runtimeturn.CounterDecision
	step       chan StepResponse
}

func NewGameSessionWithConfig(id string, leaderName string, playerCount int, cfg coreconfig.GameConfig) (*GameSession, error) {
	if id == "" {
		return nil, fmt.Errorf("session id is required")
	}
	state, err := newGameState(playerCount, cfg)
	if err != nil {
		return nil, err
	}
	state.players[0].Name = leaderName
	return &GameSession{
		id:           id,
		state:        state,
		config:       cfg,
		promptEvents: make(chan *Prompt, 8),
		mainAction:   make(chan *coretypes.PlayerAction),
		challenge:    make(chan runtimeturn.ChallengeDecision),
		counter:      make(chan runtimeturn.CounterDecision),
		step:         make(chan StepResponse),
	}, nil
}

// ID returns the stable game session identifier.
func (s *GameSession) ID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.id
}

// Snapshot returns a deep copy of the current game state.
func (s *GameSession) Snapshot() corestate.GameState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return *s.state.Snapshot()
}

func (s *GameSession) Config() coreconfig.GameConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// TurnNumber returns the current turn counter.
func (s *GameSession) TurnNumber() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.TurnNumber
}

// PlayerCount returns the number of seats in the current game.
func (s *GameSession) PlayerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.state.players)
}

func (s *GameSession) setPlayerName(playerIndex int, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if playerIndex < 0 || playerIndex >= len(s.state.players) {
		return fmt.Errorf("player index %d out of range", playerIndex)
	}
	s.state.players[playerIndex].Name = name
	return nil
}

func (s *GameSession) copyPlayerNamesFrom(source *GameSession) error {
	if source == nil {
		return fmt.Errorf("session game is nil")
	}
	sourceState := source.Snapshot()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(sourceState.players) != len(s.state.players) {
		return fmt.Errorf("player count mismatch")
	}
	for i := range s.state.players {
		s.state.players[i].Name = sourceState.players[i].Name
	}
	return nil
}

// StartTurn launches one turn against the live game state.
func (s *GameSession) StartTurn(ctx context.Context, cfg runtimeturn.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeTurn != nil {
		return fmt.Errorf("turn already active")
	}

	working := s.state.Snapshot()
	turn := &activeTurn{
		done: make(chan struct{}),
	}
	s.activeTurn = turn

	go func() {
		defer close(turn.done)
		turn.result, turn.err = runtimeturn.NewService(working).Run(ctx, s, runtimeturn.RealClock{}, cfg)
		s.clearPrompt()
		s.mu.Lock()
		if turn.err == nil {
			s.state = *working
		}
		s.lastTurnResult = turn.result
		s.lastTurnErr = turn.err
		s.activeTurn = nil
		s.mu.Unlock()
	}()

	return nil
}

// Prompt returns the currently answerable prompt, if the turn is waiting on one.
func (s *GameSession) Prompt() *Prompt {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentPrompt
}

// PromptEvents streams openings and clears for the active prompt.
func (s *GameSession) PromptEvents() <-chan *Prompt {
	return s.promptEvents
}

// AwaitTurn waits for the current turn to finish or returns the last result.
func (s *GameSession) AwaitTurn(ctx context.Context) (*turnmodel.TurnResult, error) {
	s.mu.RLock()
	turn := s.activeTurn
	lastResult := s.lastTurnResult
	lastErr := s.lastTurnErr
	s.mu.RUnlock()

	if turn == nil {
		return lastResult, lastErr
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-turn.done:
		return turn.result, turn.err
	}
}

// submitCommandFromSession validates a command envelope against the active prompt.
func (s *GameSession) submitCommandFromSession(command CommandEnvelope, caller *ClientSession) error {
	prompt := s.Prompt()
	if prompt == nil {
		return apperrors.ErrNoPrompt
	}
	if prompt.ID != command.PromptID {
		return fmt.Errorf("command %q does not match active prompt %q: %w", command.PromptID, prompt.ID, apperrors.ErrStalePrompt)
	}

	if caller == nil {
		return fmt.Errorf("submit command for session %q in room %q: %w", command.SessionID, s.ID(), apperrors.ErrSessionNotFound)
	}
	if !slices.Contains(prompt.Recipients, caller.PlayerIndex) {
		return fmt.Errorf("session %q is not an allowed recipient for prompt %q", command.SessionID, prompt.ID)
	}

	switch command.Kind {
	case CommandMainAction:
		payload, ok := command.Payload.(MainActionCommand)
		if !ok {
			return fmt.Errorf("main action command payload has unexpected type %T", command.Payload)
		}
		action, err := s.actionFromMainActionCommand(payload)
		if err != nil {
			return err
		}
		return s.SubmitMainAction(command.SessionID, action)
	case CommandChallengeDecision:
		payload, ok := command.Payload.(ChallengeDecisionCommand)
		if !ok {
			return fmt.Errorf("challenge command payload has unexpected type %T", command.Payload)
		}
		decision := runtimeturn.ChallengeDecision{
			ChallengerIndex: payload.ChallengerIndex,
			Pass:            payload.Pass,
		}
		if payload.ChallengerDiscardIndex != nil {
			decision.ChallengerDiscardIndex = *payload.ChallengerDiscardIndex
		}
		if payload.ActorDiscardIndex != nil {
			decision.ActorDiscardIndex = *payload.ActorDiscardIndex
		}
		if payload.ActorProvingCardIndex != nil {
			decision.ActorProvingCardIndex = *payload.ActorProvingCardIndex
		}
		return s.SubmitChallengeDecision(command.SessionID, decision)
	case CommandCounterDecision:
		payload, ok := command.Payload.(CounterDecisionCommand)
		if !ok {
			return fmt.Errorf("counter command payload has unexpected type %T", command.Payload)
		}
		decision := runtimeturn.CounterDecision{
			PlayerIndex: payload.PlayerIndex,
			Pass:        payload.Pass,
		}
		if payload.Command != nil {
			counterAction, err := s.actionFromCounterActionCommand(*payload.Command)
			if err != nil {
				return err
			}
			decision.Command = counterAction
		}
		return s.SubmitCounterDecision(command.SessionID, decision)
	case CommandCardSelection:
		payload, ok := command.Payload.(CardSelectionCommand)
		if !ok {
			return fmt.Errorf("card selection command payload has unexpected type %T", command.Payload)
		}
		return s.SubmitStepResponse(command.SessionID, StepResponse{
			Kind: coreactions.StepCardSelection,
			CardSelection: &coreactions.CardSelectionResponse{
				SelectedCardIndices: slices.Clone(payload.SelectedIndices),
			},
		})
	default:
		return fmt.Errorf("unsupported command kind %q", command.Kind)
	}
}

// SubmitMainAction delivers the main action to the running turn.
func (s *GameSession) SubmitMainAction(sessionID string, action *coretypes.PlayerAction) error {
	if action == nil {
		return fmt.Errorf("main action is nil")
	}
	payload, err := promptPayload[MainActionPrompt](s, PromptMainAction)
	if err != nil {
		return err
	}
	if payload.ActorIndex != action.ActorIndex {
		return fmt.Errorf("player %d is not allowed to answer %s", action.ActorIndex, PromptMainAction)
	}
	return submitTurnValue(s.mainAction, action, PromptMainAction)
}

// SubmitChallengeDecision delivers the challenge response to the running turn.
func (s *GameSession) SubmitChallengeDecision(sessionID string, decision runtimeturn.ChallengeDecision) error {
	payload, err := promptPayload[ChallengePrompt](s, PromptChallenge)
	if err != nil {
		return err
	}
	if !slices.Contains(payload.AllowedChallengers, decision.ChallengerIndex) {
		return fmt.Errorf("player %d is not allowed to answer %s", decision.ChallengerIndex, PromptChallenge)
	}
	return submitTurnValue(s.challenge, decision, PromptChallenge)
}

// SubmitCounterDecision delivers the counter response to the running turn.
func (s *GameSession) SubmitCounterDecision(sessionID string, decision runtimeturn.CounterDecision) error {
	payload, err := promptPayload[CounterPrompt](s, PromptCounter)
	if err != nil {
		return err
	}
	if !slices.Contains(payload.AllowedPlayers, decision.PlayerIndex) {
		return fmt.Errorf("player %d is not allowed to answer %s", decision.PlayerIndex, PromptCounter)
	}
	return submitTurnValue(s.counter, decision, PromptCounter)
}

// SubmitStepResponse delivers the follow-up step response to the running turn.
func (s *GameSession) SubmitStepResponse(sessionID string, response StepResponse) error {
	if _, err := promptPayload[CardSelectionPrompt](s, PromptCardSelect); err != nil {
		return err
	}
	if response.Kind != coreactions.StepCardSelection {
		return fmt.Errorf("step kind %q does not match expected kind %q", response.Kind, coreactions.StepCardSelection)
	}
	return submitTurnValue(s.step, response, PromptCardSelect)
}

func (s *GameSession) RequestAction(ctx context.Context, actor int, gs *corestate.GameState) (*coretypes.PlayerAction, error) {
	prompt := s.nextPrompt(mainActionPrompt(gs, actor))
	s.openPrompt(prompt)
	defer s.clearPromptByID(prompt.ID)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case action := <-s.mainAction:
		return action, nil
	}
}

func (s *GameSession) ChallengeResponses(ctx context.Context, window runtimeturn.ChallengeWindow) (<-chan runtimeturn.ChallengeDecision, error) {
	prompt, err := challengePrompt(window)
	if err != nil {
		return nil, err
	}
	prompt = s.nextPrompt(prompt)
	s.openPrompt(prompt)
	go func() {
		<-ctx.Done()
		s.clearPromptByID(prompt.ID)
	}()
	return s.challenge, nil
}

func (s *GameSession) CounterResponses(ctx context.Context, window runtimeturn.CounterWindow) (<-chan runtimeturn.CounterDecision, error) {
	prompt, err := counterPrompt(window)
	if err != nil {
		return nil, err
	}
	prompt = s.nextPrompt(prompt)
	s.openPrompt(prompt)
	go func() {
		<-ctx.Done()
		s.clearPromptByID(prompt.ID)
	}()
	return s.counter, nil
}

func (s *GameSession) RequestCardSelection(ctx context.Context, req coreactions.StepRequest) (coreactions.CardSelectionResponse, error) {
	prompt, err := cardSelectionPrompt(req)
	if err != nil {
		return coreactions.CardSelectionResponse{}, err
	}
	prompt = s.nextPrompt(prompt)
	s.openPrompt(prompt)
	defer s.clearPromptByID(prompt.ID)

	select {
	case <-ctx.Done():
		return coreactions.CardSelectionResponse{}, ctx.Err()
	case response := <-s.step:
		if response.CardSelection == nil {
			return coreactions.CardSelectionResponse{}, fmt.Errorf("step response missing card selection payload")
		}
		return *response.CardSelection, nil
	}
}

func (s *GameSession) nextPrompt(prompt Prompt) Prompt {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revision++
	prompt.ID = fmt.Sprintf("%s:%s:%d", s.id, prompt.Kind, s.revision)
	return prompt
}

func (s *GameSession) openPrompt(next Prompt) {
	s.mu.Lock()
	s.currentPrompt = &next
	snapshot := s.currentPrompt
	s.mu.Unlock()
	s.publishPrompt(snapshot)
}

func (s *GameSession) clearPromptByID(id string) {
	s.mu.Lock()
	if s.currentPrompt != nil && s.currentPrompt.ID == id {
		s.currentPrompt = nil
		s.mu.Unlock()
		s.publishPrompt(nil)
		return
	}
	s.mu.Unlock()
}

func (s *GameSession) clearPrompt() {
	s.mu.Lock()
	if s.currentPrompt == nil {
		s.mu.Unlock()
		return
	}
	s.currentPrompt = nil
	s.mu.Unlock()
	s.publishPrompt(nil)
}

func (s *GameSession) publishPrompt(prompt *Prompt) {
	select {
	case s.promptEvents <- prompt:
	default:
		for {
			select {
			case <-s.promptEvents:
			default:
				s.promptEvents <- prompt
				return
			}
		}
	}
}

func promptPayload[T any](s *GameSession, kind PromptKind) (T, error) {
	var zero T
	current := s.Prompt()
	if current == nil || current.Kind != kind {
		return zero, fmt.Errorf("prompt %q is not currently open", kind)
	}
	payload, ok := current.Payload.(T)
	if !ok {
		return zero, fmt.Errorf("prompt payload has unexpected type %T", current.Payload)
	}
	return payload, nil
}

func submitTurnValue[T any](ch chan T, value T, kind PromptKind) error {
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	select {
	case ch <- value:
		return nil
	case <-timer.C:
		return fmt.Errorf("prompt %q is not currently accepting submissions", kind)
	}
}

// Winner returns the surviving player when the match is over.
func (s *GameSession) Winner() *corestate.Player {
	activePlayers := 0
	var lastPlayer *corestate.Player
	state := s.Snapshot()
	for i := range state.players {
		if len(state.players[i].Hand) > 0 {
			activePlayers++
			lastPlayer = &state.players[i]
		}
	}
	if activePlayers == 1 {
		return lastPlayer
	}
	return nil
}

// IncrementTurn advances the turn counter and skips eliminated players.
func (s *GameSession) IncrementTurn() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.state.players) == 0 {
		return nil
	}

	start := s.state.CurrentPlayerIndex
	for {
		s.state.CurrentPlayerIndex = (s.state.CurrentPlayerIndex + 1) % len(s.state.players)
		if len(s.state.players[s.state.CurrentPlayerIndex].Hand) > 0 {
			break
		}
		if s.state.CurrentPlayerIndex == start {
			break
		}
	}

	s.state.TurnNumber++
	return nil
}

func (s *GameSession) hasActiveTurn() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeTurn != nil
}

func (s *GameSession) actionFromMainActionCommand(command MainActionCommand) (*coretypes.PlayerAction, error) {
	actionCommand := actionCommand(command)
	action, err := s.actionFromActionCommand(actionCommand)
	if err != nil {
		return nil, err
	}
	if _, ok := coreactions.MainAction(action.Action); !ok {
		return nil, fmt.Errorf("action %q is not a main action", action.Action)
	}
	return action, nil
}

func actionFromMainActionCommand(command MainActionCommand) (*coretypes.PlayerAction, error) {
	return (&GameSession{}).actionFromMainActionCommand(command)
}

func (s *GameSession) actionFromCounterActionCommand(command CounterActionCommand) (*coretypes.PlayerAction, error) {
	action, err := s.actionFromActionCommand(actionCommand{
		Action:     command.Action,
		ActorIndex: command.ActorIndex,
	})
	if err != nil {
		return nil, err
	}
	if _, ok := coreactions.CounterAction(action.Action); !ok {
		return nil, fmt.Errorf("action %q is not a counter action", action.Action)
	}
	return action, nil
}

func actionFromCounterActionCommand(command CounterActionCommand) (*coretypes.PlayerAction, error) {
	return (&GameSession{}).actionFromCounterActionCommand(command)
}

type actionCommand struct {
	Action      string
	ActorIndex  int
	TargetIndex *int
	Guess       string
}

func (s *GameSession) actionFromActionCommand(command actionCommand) (*coretypes.PlayerAction, error) {
	if command.Action == "" {
		return nil, fmt.Errorf("action is required")
	}
	actionID := coretypes.ActionID(command.Action)
	presentation, ok := coreactions.ActionPresentationByID(actionID)
	if !ok {
		return nil, fmt.Errorf("unknown action: %s", actionID)
	}
	action := &coretypes.PlayerAction{
		Action:     actionID,
		ActorIndex: command.ActorIndex,
	}
	switch presentation.Payload.(type) {
	case coretypes.AccusePayload:
		if command.TargetIndex == nil {
			return nil, fmt.Errorf("%s action requires a target index", actionID)
		}
		if command.Guess == "" {
			return nil, fmt.Errorf("%s action requires a role guess", actionID)
		}
		role, err := coretypes.ParseRole(command.Guess)
		if err != nil {
			return nil, err
		}
		action.Payload = coretypes.AccusePayload{
			TargetIndex: *command.TargetIndex,
			Guess:       role,
		}
	case coretypes.TargetPayload:
		if command.Guess != "" {
			return nil, fmt.Errorf("role guess is not allowed for action %q", actionID)
		}
		if command.TargetIndex == nil {
			return nil, fmt.Errorf("%s action requires a target index", actionID)
		}
		action.Payload = coretypes.TargetPayload{TargetIndex: *command.TargetIndex}
	case nil:
		if command.TargetIndex != nil {
			return nil, fmt.Errorf("target index is not allowed for action %q", actionID)
		}
		if command.Guess != "" {
			return nil, fmt.Errorf("role guess is not allowed for action %q", actionID)
		}
	default:
		return nil, fmt.Errorf("unsupported payload template %T for action %q", presentation.Payload, actionID)
	}
	return action, nil
}

func newGameState(playerCount int, cfg coreconfig.GameConfig) (corestate.GameState, error) {
	if playerCount < 2 {
		return corestate.GameState{}, fmt.Errorf("at least 2 players required")
	}
	if playerCount > 9 {
		return corestate.GameState{}, fmt.Errorf("max 9 players allowed")
	}

	cardsPerPlayer := cfg.Setup.CardsPerPlayer
	if cardsPerPlayer == 0 {
		cardsPerPlayer = 2
		if playerCount <= 4 {
			cardsPerPlayer = 3
		}
	}
	deck := corestate.NewDeck(cfg.Setup.DeckCopies)

	players := make([]corestate.Player, playerCount)
	for i := range players {
		players[i] = corestate.Player{
			ID:    i,
			Name:  placeholderPlayerName(i),
			Coins: cfg.Economy.StartingCoins,
		}
		hand, err := deck.Draw(cardsPerPlayer)
		if err != nil {
			return corestate.GameState{}, err
		}
		players[i].Hand = hand
	}

	return corestate.GameState{
		TurnNumber:         1,
		players:            players,
		Deck:               deck,
		CurrentPlayerIndex: 0,
		MaxCoins:           cfg.Economy.MaxCoins,
	}, nil
}

// placeholderPlayerName gives unclaimed lobby slots a readable label before players join.
func placeholderPlayerName(index int) string {
	return fmt.Sprintf("Open slot %d", index+1)
}
