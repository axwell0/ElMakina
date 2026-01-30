package ws

import (
	"encoding/json"

	"ElMakina/backend/models"
)

// Envelope is the shared message container for lobby and game traffic.
// request_id is required for any message that expects a response.
type Envelope struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

const (
	MsgHello       = "hello"
	MsgHelloAck    = "hello_ack"
	MsgHelloError  = "hello_error"
	MsgLobbyCreate = "lobby_create"
	MsgLobbyList   = "lobby_list"
	MsgLobbyJoin   = "lobby_join"
	MsgLobbyStart  = "lobby_start"

	MsgLobbyCreated = "lobby_created"
	MsgLobbyJoined  = "lobby_joined"
	MsgLobbyListRes = "lobby_list_result"
	MsgLobbyStarted = "lobby_started"
	MsgLobbyState   = "lobby_state"
	MsgLobbyError   = "lobby_error"

	MsgRequestAction    = "request_action"
	MsgRequestStep      = "request_step"
	MsgChallengeWindow  = "challenge_window"
	MsgCounterWindow    = "counter_window"
	MsgAction           = "action"
	MsgChallenge        = "challenge"
	MsgCounter          = "counter"
	MsgStepResult       = "step_result"
	MsgGameLog          = "game_log"
	MsgGameOver         = "game_over"
	MsgGameState        = "game_state"
	MsgGameConfig       = "game_config"
	MsgPromptClosed     = "prompt_closed"
	MsgInvestigateRes   = "investigate_result"
	MsgHandState        = "hand_state"
	MsgPlayerEliminated = "player_eliminated"
	MsgTurnTimer        = "turn_timer"
	MsgGamePaused       = "game_paused"
	MsgGameResumed      = "game_resumed"
	MsgKickVote         = "kick_vote"
	MsgKickVoteUpdate   = "kick_vote_update"
	MsgPlayerKicked     = "player_kicked"
	MsgChatMessage      = "chat_message"
	MsgCardDiscarded    = "card_discarded"
)

type HelloPayload struct {
	Nickname       string `json:"nickname"`
	ReconnectToken string `json:"reconnect_token,omitempty"`
	Avatar         string `json:"avatar,omitempty"`
}

type HelloAckPayload struct {
	PlayerID string `json:"player_id"`
	Token    string `json:"token"`
}

type HelloErrorPayload struct {
	Error string `json:"error"`
}

type LobbyJoinPayload struct {
	LobbyID string `json:"lobby_id"`
}

type LobbyStartPayload struct {
	LobbyID string `json:"lobby_id"`
}

type LobbyCreatedPayload struct {
	LobbyID string `json:"lobby_id"`
}

type LobbyListPayload struct {
	Lobbies []LobbySummaryPayload `json:"lobbies"`
}

type LobbyStatePayload struct {
	LobbyID       string   `json:"lobby_id"`
	LeaderNick    string   `json:"leader_nick"`
	LeaderID      string   `json:"leader_id"`
	PlayerNicks   []string `json:"player_nicks"`
	PlayerIDs     []string `json:"player_ids"`
	PlayerAvatars []string `json:"player_avatars,omitempty"`
	PlayerCount   int      `json:"player_count"`
	Status        string   `json:"status"`
}

type LobbySummaryPayload struct {
	ID            string   `json:"id"`
	LeaderNick    string   `json:"leader_nick"`
	LeaderID      string   `json:"leader_id"`
	PlayerCount   int      `json:"player_count"`
	PlayerNicks   []string `json:"player_nicks,omitempty"`
	PlayerIDs     []string `json:"player_ids,omitempty"`
	PlayerAvatars []string `json:"player_avatars,omitempty"`
	Status        string   `json:"status"`
}

type LobbyStartedPayload struct {
	LobbyID       string         `json:"lobby_id"`
	MatchID       string         `json:"match_id"`
	PlayerIndex   int            `json:"player_index"`
	PlayerCount   int            `json:"player_count"`
	PlayerNames   []string       `json:"player_names"`
	PlayerAvatars []string       `json:"player_avatars,omitempty"`
	IndexMapping  map[string]int `json:"index_mapping"`
}

type ChatMessagePayload struct {
	ID          string `json:"id"`
	SenderIndex int    `json:"senderIndex"`
	SenderName  string `json:"senderName"`
	Text        string `json:"text"`
	Timestamp   int64  `json:"timestamp"`
}

type ActionPayload struct {
	ID          models.ActionID  `json:"id"`
	SourceIndex *int             `json:"source_index"`
	TargetIndex *int             `json:"target_index,omitempty"`
	Guess       *string          `json:"guess,omitempty"`
	MainAction  *models.ActionID `json:"main_action,omitempty"`
	Pass        bool             `json:"pass,omitempty"`
}

type ChallengePayload struct {
	ChallengerIndex        *int `json:"challenger_index"`
	ChallengerDiscardIndex int  `json:"challenger_discard_index"`
	ActorDiscardIndex      int  `json:"actor_discard_index"`
	ActorProvingCardIndex  int  `json:"actor_proving_card_index"`
	Pass                   bool `json:"pass,omitempty"`
}

type RequestActionPayload struct {
	ActorIndex    int               `json:"actor_index"`
	AllowedAction []models.ActionID `json:"allowed_actions,omitempty"`
	Prompt        string            `json:"prompt,omitempty"`
}

type RequestStepPayload struct {
	Step   any    `json:"step"`
	Prompt string `json:"prompt,omitempty"`
}

type ChallengeWindowPayload struct {
	ActorIndex  int    `json:"actor_index"`
	ActionID    string `json:"action_id"`
	ClaimedRole string `json:"claimed_role"`
	Kind        string `json:"kind"`
	TargetIndex *int   `json:"target_index,omitempty"`
	Eligible    bool   `json:"eligible"`
	TimeoutMs   int64  `json:"timeout_ms,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
}

type CounterWindowPayload struct {
	ActorIndex    int               `json:"actor_index"`
	ActionID      string            `json:"action_id"`
	AllowedAction []models.ActionID `json:"allowed_actions,omitempty"`
	TargetIndex   *int              `json:"target_index,omitempty"`
	Eligible      bool              `json:"eligible"`
	TimeoutMs     int64             `json:"timeout_ms,omitempty"`
	Prompt        string            `json:"prompt,omitempty"`
}

// GameLogPayload delivers a single public or private log entry.
type GameLogPayload struct {
	Turn        int    `json:"turn"`
	Scope       string `json:"scope"`
	Message     string `json:"message"`
	PlayerIndex *int   `json:"player_index,omitempty"`
}

type GameOverPayload struct {
	WinnerIndex int    `json:"winner_index"`
	WinnerName  string `json:"winner_name"`
}

type PlayerStatePayload struct {
	Index     int    `json:"index"`
	Name      string `json:"name"`
	Coins     int    `json:"coins"`
	CardCount int    `json:"card_count"`
	Alive     bool   `json:"alive"`
	Avatar    string `json:"avatar,omitempty"`
}

type GameStatePayload struct {
	TurnNumber        int                  `json:"turn_number"`
	ActivePlayerIndex int                  `json:"active_player_index"`
	Players           []PlayerStatePayload `json:"players"`
}

type GameConfigPayload struct {
	Roles []string `json:"roles"`
}

type PromptClosedPayload struct {
	Reason string `json:"reason"`
}

type InvestigateResultPayload struct {
	TargetName string `json:"target_name"`
	Role       string `json:"role"`
}

type HandStatePayload struct {
	PlayerIndex int      `json:"player_index"`
	Hand        []string `json:"hand"`
}

type PlayerEliminatedPayload struct {
	PlayerIndex int    `json:"player_index"`
	Reason      string `json:"reason"`
	Turn        int    `json:"turn"`
}

type TurnTimerPayload struct {
	ActivePlayerIndex int    `json:"active_player_index"`
	TurnNumber        int    `json:"turn_number"`
	DurationMs        int64  `json:"duration_ms"`
	State             string `json:"state"`
}

type GamePausedPayload struct {
	PausedByPlayerID string `json:"paused_by_player_id"`
	PausedByIndex    int    `json:"paused_by_index"`
	PausedByName     string `json:"paused_by_name"`
	DeadlineMs       int64  `json:"deadline_ms"`
	DurationMs       int64  `json:"duration_ms"`
	PauseReason      string `json:"pause_reason"`
	EligibleVoters   []int  `json:"eligible_voters"`
	KickVotes        []int  `json:"kick_votes"`
}

type GameResumedPayload struct {
	ResumedByPlayerID string `json:"resumed_by_player_id"`
	ResumedByIndex    int    `json:"resumed_by_index"`
	ResumedByName     string `json:"resumed_by_name"`
	ResumeReason      string `json:"resume_reason"`
}

type KickVotePayload struct {
	TargetIndex int `json:"target_index"`
}

type KickVoteUpdatePayload struct {
	EligibleVoters []int `json:"eligible_voters"`
	KickVotes      []int `json:"kick_votes"`
}

type PlayerKickedPayload struct {
	PlayerIndex int    `json:"player_index"`
	Reason      string `json:"reason"`
}

// CardDiscardedPayload notifies all players when a card is discarded
type CardDiscardedPayload struct {
	PlayerIndex   int    `json:"player_index"`
	PlayerName    string `json:"player_name"`
	CardRole      string `json:"card_role"`
	Reason        string `json:"reason"`
	Turn          int    `json:"turn"`
	IsElimination bool   `json:"is_elimination"`
}
