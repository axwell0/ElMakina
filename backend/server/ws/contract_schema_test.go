package ws

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func TestEnvelopeSchemaExamples(t *testing.T) {
	schema := loadEnvelopeSchema(t)
	table := []Envelope{
		{Type: MsgHello, Payload: mustJSON(HelloPayload{Nickname: "Ada"})},
		{Type: MsgHelloAck, Payload: mustJSON(HelloAckPayload{PlayerID: "p1", Token: "t1"})},
		{Type: MsgHelloError, Payload: mustJSON(HelloErrorPayload{Error: "bad"})},

		{Type: MsgLobbyCreate, RequestID: "req-1"},
		{Type: MsgLobbyList, RequestID: "req-2"},
		{Type: MsgLobbyJoin, RequestID: "req-3", Payload: mustJSON(LobbyJoinPayload{LobbyID: "l1"})},
		{Type: MsgLobbyStart, RequestID: "req-4", Payload: mustJSON(LobbyStartPayload{LobbyID: "l1"})},

		{Type: MsgLobbyCreated, RequestID: "req-1", Payload: mustJSON(LobbyCreatedPayload{LobbyID: "l1"})},
		{Type: MsgLobbyJoined, RequestID: "req-3", Payload: mustJSON(LobbyJoinPayload{LobbyID: "l1"})},
		{Type: MsgLobbyListRes, Payload: mustJSON(LobbyListPayload{Lobbies: []LobbySummaryPayload{
			{ID: "l1", LeaderNick: "Ada", LeaderID: "p1", PlayerCount: 2, PlayerNicks: []string{"Ada", "Ben"}, PlayerIDs: []string{"p1", "p2"}, Status: "open"},
		}})},
		{Type: MsgLobbyStarted, Payload: mustJSON(LobbyStartedPayload{LobbyID: "l1", MatchID: "m1", PlayerIndex: 0, PlayerCount: 2, PlayerNames: []string{"Ada", "Ben"}, IndexMapping: map[string]int{"p1": 0, "p2": 1}})},
		{Type: MsgLobbyState, Payload: mustJSON(LobbyStatePayload{LobbyID: "l1", LeaderNick: "Ada", LeaderID: "p1", PlayerNicks: []string{"Ada", "Ben"}, PlayerIDs: []string{"p1", "p2"}, PlayerCount: 2, Status: "open"})},
		{Type: MsgLobbyError, RequestID: "req-3", Payload: mustJSON(map[string]string{"error": "not allowed"})},

		{Type: MsgRequestAction, RequestID: "req-10", Payload: mustJSON(RequestActionPayload{ActorIndex: 0, AllowedAction: []models.ActionID{models.Income}, Prompt: "choose your action"})},
		{Type: MsgChallengeWindow, RequestID: "req-11", Payload: mustJSON(ChallengeWindowPayload{ActorIndex: 0, ActionID: "steal", ClaimedRole: "captain", Kind: "main", Eligible: true, Prompt: "challenge or pass"})},
		{Type: MsgCounterWindow, RequestID: "req-12", Payload: mustJSON(CounterWindowPayload{ActorIndex: 0, ActionID: "steal", AllowedAction: []models.ActionID{models.BlockSteal}, Eligible: true})},
		{Type: MsgRequestStep, RequestID: "req-13", Payload: mustJSON(RequestStepPayload{Step: state.StepRequest{ActionID: "exchange", Kind: state.StepCardSelect, Actor: 0, Count: 1, Context: state.ContextDiscard, Options: []string{"Card 1"}}, Prompt: "select cards"})},

		{Type: MsgAction, RequestID: "req-10", Payload: mustJSON(ActionPayload{ID: "income", SourceIndex: intPtr(0)})},
		{Type: MsgChallenge, RequestID: "req-11", Payload: mustJSON(ChallengePayload{ChallengerIndex: intPtr(1), ChallengerDiscardIndex: 0, ActorDiscardIndex: 0, ActorProvingCardIndex: 0, Pass: true})},
		{Type: MsgCounter, RequestID: "req-12", Payload: mustJSON(ActionPayload{SourceIndex: intPtrSchema(1), MainAction: actionIDPtr(models.Steal), Pass: true})},
		{Type: MsgStepResult, RequestID: "req-13", Payload: json.RawMessage(`[0]`)},

		{Type: MsgGameLog, Payload: mustJSON(GameLogPayload{Turn: 1, Scope: "public", Message: "log"})},
		{Type: MsgGameState, Payload: mustJSON(GameStatePayload{TurnNumber: 1, ActivePlayerIndex: 0, Players: []PlayerStatePayload{{Index: 0, Name: "Ada", Coins: 2, CardCount: 2, Alive: true}}})},
		{Type: MsgGameConfig, Payload: mustJSON(GameConfigPayload{Roles: []string{"duke"}})},
		{Type: MsgGameOver, Payload: mustJSON(GameOverPayload{WinnerIndex: 0, WinnerName: "Ada"})},
		{Type: MsgPromptClosed, RequestID: "req-10", Payload: mustJSON(PromptClosedPayload{Reason: "action_submitted"})},
		{Type: MsgInvestigateRes, Payload: mustJSON(InvestigateResultPayload{TargetName: "Ben", Role: "duke"})},
		{Type: MsgHandState, Payload: mustJSON(HandStatePayload{PlayerIndex: 0, Hand: []string{"duke"}})},
		{Type: MsgPlayerEliminated, Payload: mustJSON(PlayerEliminatedPayload{PlayerIndex: 1, Reason: "lost_last_card", Turn: 2})},
		{Type: MsgTurnTimer, Payload: mustJSON(TurnTimerPayload{ActivePlayerIndex: 0, TurnNumber: 1, DurationMs: 15000, State: "start"})},
		{Type: MsgGamePaused, Payload: mustJSON(GamePausedPayload{PausedByPlayerID: "p2", PausedByIndex: 1, PausedByName: "Ben", DeadlineMs: 123, DurationMs: 60000, PauseReason: "disconnect", EligibleVoters: []int{0, 2}, KickVotes: []int{0}})},
		{Type: MsgGameResumed, Payload: mustJSON(GameResumedPayload{ResumedByPlayerID: "p2", ResumedByIndex: 1, ResumedByName: "Ben", ResumeReason: "reconnected"})},
		{Type: MsgKickVote, Payload: mustJSON(KickVotePayload{TargetIndex: 1})},
		{Type: MsgKickVoteUpdate, Payload: mustJSON(KickVoteUpdatePayload{EligibleVoters: []int{0, 2}, KickVotes: []int{0}})},
		{Type: MsgPlayerKicked, Payload: mustJSON(PlayerKickedPayload{PlayerIndex: 1, Reason: "timeout"})},
		{Type: MsgChatMessage, Payload: mustJSON(ChatMessagePayload{ID: "c1", SenderIndex: 0, SenderName: "Ada", Text: "hi", Timestamp: 1234567890})},
	}

	for i, env := range table {
		if err := validateEnvelope(schema, env); err != nil {
			t.Fatalf("schema validation failed for case %d (%s): %v", i, env.Type, err)
		}
	}
}

func loadEnvelopeSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	path := filepath.Join("..", "..", "..", "shared", "schemas", "envelope.json")
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open schema: %v", err)
	}
	defer file.Close()
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("envelope.json", file); err != nil {
		t.Fatalf("add schema resource: %v", err)
	}
	schema, err := compiler.Compile("envelope.json")
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return schema
}

func validateEnvelope(schema *jsonschema.Schema, env Envelope) error {
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}
	var v any
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return err
	}
	return schema.Validate(v)
}

func intPtrSchema(v int) *int { return &v }

func actionIDPtr(v models.ActionID) *models.ActionID { return &v }
