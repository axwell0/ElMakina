package api

import coreconfig "github.com/axwell0/elmakina/engine/core/config"

// MessageType identifies the kind of websocket message being sent or received.
type MessageType string

const (
	TypeRoomSync    MessageType = "room.sync"
	TypePrompt      MessageType = "prompt"
	TypeTurnResult  MessageType = "turn.result"
	TypeGameOver    MessageType = "game.over"
	TypeCommand     MessageType = "command"
	TypeSubmitReady MessageType = "room.ready"
	TypeSubmitStart MessageType = "room.start"
)

// CreateRoomRequest is the POST body the browser sends when a new hosted room is created.
type CreateRoomRequest struct {
	// LeaderName is the display name assigned to the first player who creates the room.
	LeaderName string `json:"leaderName"`
	// PlayerCount is the total number of seats the room should reserve.
	PlayerCount int `json:"playerCount"`
	// DeviceID identifies the browser device so the session can be resumed later.
	DeviceID   string                 `json:"deviceId"`
	GameConfig *coreconfig.GameConfig `json:"gameConfig,omitempty"`
}

// CreateRoomResponse is the creation result returned to the first player.
type CreateRoomResponse struct {
	// RoomID is the room identifier used by every server endpoint.
	RoomID string `json:"roomId"`
	// RoomCode is the shareable short code shown to players in the browser.
	RoomCode string `json:"roomCode"`
	// JoinURL is the full browser URL that opens the room join page.
	JoinURL string `json:"joinUrl"`
	// SessionID is the newly created player session ID.
	SessionID string `json:"sessionId"`
	// ResumeToken is the bearer token used to authenticate reconnects.
	ResumeToken string `json:"resumeToken"`
	// PlayerIndex is the zero-based seat assigned to the creator.
	PlayerIndex int `json:"playerIndex"`
	// PlayerName is the final display name recorded for the created session.
	PlayerName string `json:"playerName"`
	// Resumed reports whether the session was reattached instead of freshly created.
	Resumed bool `json:"resumed"`
}

// JoinRoomRequest is the POST body used when another player joins an existing room.
type JoinRoomRequest struct {
	// PlayerName is the display name requested for the joining player.
	PlayerName string `json:"playerName"`
	// DeviceID identifies the browser so reconnects can find the same session later.
	DeviceID string `json:"deviceId"`
}

// JoinRoomResponse mirrors the create-room response for players entering an existing room.
type JoinRoomResponse struct {
	// RoomID is the room identifier used by every server endpoint.
	RoomID string `json:"roomId"`
	// RoomCode is the shareable short code shown to players in the browser.
	RoomCode string `json:"roomCode"`
	// JoinURL is the full browser URL that opens the room join page.
	JoinURL string `json:"joinUrl"`
	// SessionID is the newly created player session ID.
	SessionID string `json:"sessionId"`
	// ResumeToken is the bearer token used to authenticate reconnects.
	ResumeToken string `json:"resumeToken"`
	// PlayerIndex is the zero-based seat assigned to the joining player.
	PlayerIndex int `json:"playerIndex"`
	// PlayerName is the final display name recorded for the joined session.
	PlayerName string `json:"playerName"`
	// Resumed reports whether the session was reattached instead of freshly created.
	Resumed bool `json:"resumed"`
}

// PlayerSnapshot is the public row shown for one player in a room snapshot.
type PlayerSnapshot struct {
	// PlayerIndex is the zero-based seat number for the player.
	PlayerIndex int `json:"playerIndex"`
	// Name is the public display name shown in the lobby and summary views.
	Name string `json:"name"`
	// Coins is the current coin count visible to all players.
	Coins int `json:"coins"`
	// Cards is the number of cards the player still holds.
	Cards int `json:"cards"`
	// Eliminated marks whether the player has been knocked out of the match.
	Eliminated bool `json:"eliminated"`
}

// MatchSnapshot is the shared match-wide snapshot embedded in room responses and websocket syncs.
type MatchSnapshot struct {
	// TurnNumber is the current turn counter for the match.
	TurnNumber int `json:"turnNumber"`
	// CurrentPlayerIndex identifies the active player for the turn.
	CurrentPlayerIndex int `json:"currentPlayerIndex"`
	// Players contains the public summary for every seat in the room.
	Players []PlayerSnapshot `json:"players"`
}

// PrivateCardView reveals one card to its owner.
type PrivateCardView struct {
	ID   int    `json:"id"`
	Role string `json:"role"`
}

// RoomView is the public room-state payload that drives the lobby UI.
type RoomView struct {
	// Phase is the current room lifecycle state.
	Phase string `json:"phase"`
	// LeaderPlayerIndex identifies the player who controls lobby actions.
	LeaderPlayerIndex int `json:"leaderPlayerIndex"`
	// PlayerCount is the number of seats reserved for the room.
	PlayerCount int `json:"playerCount"`
	// JoinedPlayers is the count of seats that have been filled so far.
	JoinedPlayers int `json:"joinedPlayers"`
	// ConnectedPlayers lists the seats that currently have a live websocket.
	ConnectedPlayers []int `json:"connectedPlayers"`
	// ReadyPlayers lists the seats that have marked themselves ready.
	ReadyPlayers []int `json:"readyPlayers"`
	// CanStart reports whether the leader can begin the match.
	CanStart bool `json:"canStart"`
	// CanRestart reports whether the leader can restart a finished match.
	CanRestart bool `json:"canRestart"`
	// CanDelete reports whether the room can be deleted right now.
	CanDelete  bool                   `json:"canDelete"`
	GameConfig *coreconfig.GameConfig `json:"gameConfig,omitempty"`
}

// RoomSyncView is the full client snapshot sent whenever room state changes.
type RoomSyncView struct {
	Type              MessageType       `json:"type"`
	RoomID            string            `json:"roomId"`
	PlayerIndex       int               `json:"playerIndex"`
	PlayerName        string            `json:"playerName"`
	StatusTitle       string            `json:"statusTitle,omitempty"`
	StatusDescription string            `json:"statusDescription,omitempty"`
	Room              RoomView          `json:"room"`
	Summary           MatchSnapshot     `json:"summary"`
	Hand              []PrivateCardView `json:"hand,omitempty"`
}

// GameOverView tells clients who won the match.
type GameOverView struct {
	Type        MessageType   `json:"type"`
	RoomID      string        `json:"roomId"`
	WinnerIndex int           `json:"winnerIndex"`
	WinnerName  string        `json:"winnerName"`
	Summary     MatchSnapshot `json:"summary"`
}

// ActionPayloadView is the client-facing action payload.
type ActionPayloadView struct {
	TargetIndex *int   `json:"targetIndex,omitempty"`
	Guess       string `json:"guess,omitempty"`
}

// ActionView is the wire format for a player action.
type ActionView struct {
	ID         string            `json:"id"`
	ActorIndex int               `json:"actorIndex"`
	Payload    ActionPayloadView `json:"payload,omitempty"`
}

// ChallengeResultView reports the outcome of a challenge to the UI.
type ChallengeResultView struct {
	Success         bool     `json:"success"`
	ActionAllowed   bool     `json:"actionAllowed"`
	ActorIndex      int      `json:"actorIndex"`
	ChallengerIndex int      `json:"challengerIndex"`
	ClaimedRole     string   `json:"claimedRole"`
	Outcome         string   `json:"outcome"`
	Logs            []string `json:"logs,omitempty"`
	LostCardRole    string   `json:"lostCardRole,omitempty"`
	LostPlayerIndex *int     `json:"lostPlayerIndex,omitempty"`
}

// CounterResultView reports the outcome of a counter action.
type CounterResultView struct {
	Action    *ActionView          `json:"action,omitempty"`
	Applied   bool                 `json:"applied"`
	Canceled  bool                 `json:"canceled"`
	Challenge *ChallengeResultView `json:"challenge,omitempty"`
}

// TurnResultView is the full payload sent when a turn finishes.
type TurnResultView struct {
	Type             MessageType           `json:"type"`
	RoomID           string                `json:"roomId"`
	TurnNumber       int                   `json:"turnNumber"`
	Main             *ActionView           `json:"main,omitempty"`
	MainApplied      bool                  `json:"mainApplied"`
	MainCanceled     bool                  `json:"mainCanceled"`
	Logs             []string              `json:"logs,omitempty"`
	PrivateLogs      []string              `json:"privateLogs,omitempty"`
	CounterResults   []CounterResultView   `json:"counterResults,omitempty"`
	ChallengeResults []ChallengeResultView `json:"challengeResults,omitempty"`
	Summary          MatchSnapshot         `json:"summary"`
	Hand             []PrivateCardView     `json:"hand,omitempty"`
}

// PromptView is the websocket shape for a session prompt.
type PromptView struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Recipients []int  `json:"recipients"`
	Payload    any    `json:"payload"`
}

// RoomSnapshotResponse combines the room view and match summary for HTTP GET endpoints.
type RoomSnapshotResponse struct {
	// RoomID is the room identifier used by every server endpoint.
	RoomID string `json:"roomId"`
	// Room contains the public lobby or match state.
	Room RoomView `json:"room"`
	// Summary contains the match-wide player and turn snapshot.
	Summary MatchSnapshot `json:"summary"`
}

// ListRoomsResponse wraps a list of room snapshots for the room index endpoint.
type ListRoomsResponse struct {
	// Rooms contains every active room snapshot returned by the server.
	Rooms []RoomSnapshotResponse `json:"rooms"`
}

// RoomMutationRequest carries the session ID used for restart/delete actions.
type RoomMutationRequest struct {
	// SessionID identifies the authorized session that is requesting the mutation.
	SessionID string `json:"sessionId"`
}

// ErrorResponse is the standard JSON error body used by HTTP handlers.
type ErrorResponse struct {
	// Error contains the human-readable reason the request failed.
	Error string `json:"error"`
}

// ResumableGameSnapshot is the browser-facing summary for one reconnectable session.
type ResumableGameSnapshot struct {
	// RoomID is the room identifier used by every server endpoint.
	RoomID string `json:"roomId"`
	// RoomCode is the shareable short code shown to players in the browser.
	RoomCode string `json:"roomCode"`
	// JoinURL is the full browser URL that opens the room join page.
	JoinURL string `json:"joinUrl"`
	// SessionID is the reconnectable player session ID.
	SessionID string `json:"sessionId"`
	// PlayerName is the recorded display name for the session.
	PlayerName string `json:"playerName"`
	// PlayerIndex is the zero-based seat assigned to the session.
	PlayerIndex int `json:"playerIndex"`
	// Phase is the current room lifecycle state.
	Phase string `json:"phase"`
	// Connected reports whether the session currently has a live websocket.
	Connected bool `json:"connected"`
	// CanReconnect reports whether the browser may resume this session.
	CanReconnect bool `json:"canReconnect"`
	// LastSeenAt records the last activity timestamp in RFC3339 format.
	LastSeenAt string `json:"lastSeenAt"`
}

// ListResumableGamesResponse wraps all resumable sessions for one browser device.
type ListResumableGamesResponse struct {
	// Games contains every session this browser device can attempt to resume.
	Games []ResumableGameSnapshot `json:"games"`
}
