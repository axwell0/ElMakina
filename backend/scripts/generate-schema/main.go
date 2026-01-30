package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"ElMakina/backend/engine/state"
	"ElMakina/backend/server/ws"

	"github.com/invopop/jsonschema"
)

func main() {
	r := &jsonschema.Reflector{
		ExpandedStruct: true,
		DoNotReference: false,
	}

	schema := buildEnvelopeSchema(r)

	out, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
		os.Exit(1)
	}

	outPath := filepath.Join("shared", "schemas", "envelope.json")
	if err := os.WriteFile(outPath, out, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s\n", outPath)
}

func buildEnvelopeSchema(r *jsonschema.Reflector) map[string]any {
	defs := make(map[string]any)

	// Generate schemas for all payload types using pointers
	addDef(r, defs, "HelloPayload", &ws.HelloPayload{})
	addDef(r, defs, "HelloAckPayload", &ws.HelloAckPayload{})
	addDef(r, defs, "HelloErrorPayload", &ws.HelloErrorPayload{})
	addDef(r, defs, "LobbyJoinPayload", &ws.LobbyJoinPayload{})
	addDef(r, defs, "LobbyStartPayload", &ws.LobbyStartPayload{})
	addDef(r, defs, "LobbyCreatedPayload", &ws.LobbyCreatedPayload{})
	addDef(r, defs, "LobbyListPayload", &ws.LobbyListPayload{})
	addDef(r, defs, "LobbyStatePayload", &ws.LobbyStatePayload{})
	addDef(r, defs, "LobbySummaryPayload", &ws.LobbySummaryPayload{})
	addDef(r, defs, "LobbyStartedPayload", &ws.LobbyStartedPayload{})
	addDef(r, defs, "ActionPayload", &ws.ActionPayload{})
	addDef(r, defs, "ChallengePayload", &ws.ChallengePayload{})
	addDef(r, defs, "RequestActionPayload", &ws.RequestActionPayload{})
	addDef(r, defs, "RequestStepPayload", &ws.RequestStepPayload{})
	addDef(r, defs, "ChallengeWindowPayload", &ws.ChallengeWindowPayload{})
	addDef(r, defs, "CounterWindowPayload", &ws.CounterWindowPayload{})
	addDef(r, defs, "GameLogPayload", &ws.GameLogPayload{})
	addDef(r, defs, "GameOverPayload", &ws.GameOverPayload{})
	addDef(r, defs, "PlayerStatePayload", &ws.PlayerStatePayload{})
	addDef(r, defs, "GameStatePayload", &ws.GameStatePayload{})
	addDef(r, defs, "GameConfigPayload", &ws.GameConfigPayload{})
	addDef(r, defs, "PromptClosedPayload", &ws.PromptClosedPayload{})
	addDef(r, defs, "InvestigateResultPayload", &ws.InvestigateResultPayload{})
	addDef(r, defs, "HandStatePayload", &ws.HandStatePayload{})
	addDef(r, defs, "PlayerEliminatedPayload", &ws.PlayerEliminatedPayload{})
	addDef(r, defs, "TurnTimerPayload", &ws.TurnTimerPayload{})
	addDef(r, defs, "GamePausedPayload", &ws.GamePausedPayload{})
	addDef(r, defs, "GameResumedPayload", &ws.GameResumedPayload{})
	addDef(r, defs, "KickVotePayload", &ws.KickVotePayload{})
	addDef(r, defs, "KickVoteUpdatePayload", &ws.KickVoteUpdatePayload{})
	addDef(r, defs, "PlayerKickedPayload", &ws.PlayerKickedPayload{})
	addDef(r, defs, "ChatMessagePayload", &ws.ChatMessagePayload{})
	addDef(r, defs, "StepRequest", &state.StepRequest{})

	// LobbyErrorPayload defined inline (no Go type exists)
	defs["LobbyErrorPayload"] = map[string]any{
		"type":     "object",
		"required": []string{"error"},
		"properties": map[string]any{
			"error": map[string]any{"type": "string"},
		},
		"additionalProperties": false,
	}

	// EnvelopeBase
	defs["EnvelopeBase"] = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"type":       map[string]any{"type": "string"},
			"request_id": map[string]any{"type": "string"},
			"payload":    map[string]any{},
		},
		"additionalProperties": false,
	}

	// StepResultPayload
	defs["StepResultPayload"] = map[string]any{
		"oneOf": []any{
			map[string]any{"type": "array", "items": map[string]any{"type": "integer"}},
			map[string]any{"type": "object"},
			map[string]any{"type": "string"},
			map[string]any{"type": "number"},
			map[string]any{"type": "integer"},
			map[string]any{"type": "boolean"},
			map[string]any{"type": "null"},
		},
	}

	// Message type definitions
	messageTypes := []struct {
		Name    string
		MsgType string
		Payload string
	}{
		{"Hello", ws.MsgHello, "HelloPayload"},
		{"HelloAck", ws.MsgHelloAck, "HelloAckPayload"},
		{"HelloError", ws.MsgHelloError, "HelloErrorPayload"},
		{"LobbyCreate", ws.MsgLobbyCreate, ""},
		{"LobbyList", ws.MsgLobbyList, ""},
		{"LobbyJoin", ws.MsgLobbyJoin, "LobbyJoinPayload"},
		{"LobbyStart", ws.MsgLobbyStart, "LobbyStartPayload"},
		{"LobbyCreated", ws.MsgLobbyCreated, "LobbyCreatedPayload"},
		{"LobbyJoined", ws.MsgLobbyJoined, "LobbyJoinPayload"},
		{"LobbyListResult", ws.MsgLobbyListRes, "LobbyListPayload"},
		{"LobbyStarted", ws.MsgLobbyStarted, "LobbyStartedPayload"},
		{"LobbyState", ws.MsgLobbyState, "LobbyStatePayload"},
		{"LobbyError", ws.MsgLobbyError, "LobbyErrorPayload"},
		{"RequestAction", ws.MsgRequestAction, "RequestActionPayload"},
		{"RequestStep", ws.MsgRequestStep, "RequestStepPayload"},
		{"ChallengeWindow", ws.MsgChallengeWindow, "ChallengeWindowPayload"},
		{"CounterWindow", ws.MsgCounterWindow, "CounterWindowPayload"},
		{"Action", ws.MsgAction, "ActionPayload"},
		{"Challenge", ws.MsgChallenge, "ChallengePayload"},
		{"Counter", ws.MsgCounter, "ActionPayload"},
		{"StepResult", ws.MsgStepResult, "StepResultPayload"},
		{"GameLog", ws.MsgGameLog, "GameLogPayload"},
		{"GameOver", ws.MsgGameOver, "GameOverPayload"},
		{"GameState", ws.MsgGameState, "GameStatePayload"},
		{"GameConfig", ws.MsgGameConfig, "GameConfigPayload"},
		{"PromptClosed", ws.MsgPromptClosed, "PromptClosedPayload"},
		{"InvestigateResult", ws.MsgInvestigateRes, "InvestigateResultPayload"},
		{"HandState", ws.MsgHandState, "HandStatePayload"},
		{"PlayerEliminated", ws.MsgPlayerEliminated, "PlayerEliminatedPayload"},
		{"TurnTimer", ws.MsgTurnTimer, "TurnTimerPayload"},
		{"GamePaused", ws.MsgGamePaused, "GamePausedPayload"},
		{"GameResumed", ws.MsgGameResumed, "GameResumedPayload"},
		{"KickVote", ws.MsgKickVote, "KickVotePayload"},
		{"KickVoteUpdate", ws.MsgKickVoteUpdate, "KickVoteUpdatePayload"},
		{"PlayerKicked", ws.MsgPlayerKicked, "PlayerKickedPayload"},
		{"ChatMessage", ws.MsgChatMessage, "ChatMessagePayload"},
	}

	var oneOf []any
	for _, mt := range messageTypes {
		defs[mt.Name] = buildMessageDef(mt.MsgType, mt.Payload)
		oneOf = append(oneOf, map[string]any{"$ref": "#/$defs/" + mt.Name})
	}

	return map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id":     "https://elmakina.example/schemas/ws-envelope.json",
		"title":   "ElMakina WebSocket Envelope",
		"type":    "object",
		"oneOf":   oneOf,
		"$defs":   defs,
	}
}

func addDef(r *jsonschema.Reflector, defs map[string]any, name string, v any) {
	s := r.Reflect(v)
	defs[name] = schemaToMap(s)
}

func buildMessageDef(msgType, payloadType string) map[string]any {
	props := map[string]any{
		"type": map[string]any{"const": msgType},
	}
	required := []string{"type"}

	if payloadType != "" {
		props["payload"] = map[string]any{"$ref": "#/$defs/" + payloadType}
		required = append(required, "payload")
	}

	return map[string]any{
		"allOf": []any{
			map[string]any{"$ref": "#/$defs/EnvelopeBase"},
			map[string]any{
				"type":       "object",
				"required":   required,
				"properties": props,
			},
		},
	}
}

func schemaToMap(s *jsonschema.Schema) map[string]any {
	b, _ := json.Marshal(s)
	var m map[string]any
	json.Unmarshal(b, &m)

	// Clean up generated schema
	delete(m, "$schema")
	delete(m, "$id")

	if m["type"] == "object" {
		m["additionalProperties"] = false
	}

	return m
}
