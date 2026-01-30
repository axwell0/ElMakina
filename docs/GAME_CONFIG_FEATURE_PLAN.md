# Game Configuration & Extend Feature Implementation Plan

**Status:** Draft - Ready for Review  
**Date:** January 30, 2026  
**Feature:** Per-Game Configuration System + Extend Button + Team Mode  
**Game:** El Makina (NOT Coup - uses El Makina terminology)

---

## Document Changelog

- **v1.0** (Initial): Basic configuration + extend feature
- **v1.1** (Current): Added Team Mode, 35+ configuration options, 8 presets, El Makina terminology

---

## 1. Executive Summary

This document outlines the implementation of a comprehensive per-game configuration system that allows lobby leaders to customize game rules, timeouts, and action costs before starting a match. Additionally, it introduces an "Extend" button mechanic where players can extend challenge/counter windows a limited number of times per game.

---

## 2. Goals

1. **Per-Game Customization:** Allow each lobby to have unique game rules
2. **Frontend-Driven:** Configuration set via UI before game start
3. **Validation:** Server validates all configurations for sanity
4. **Extend Mechanic:** Players can extend time windows 3 times (configurable) per game
5. **Backward Compatible:** Default values match current behavior
6. **Replay Support:** Configurations stored for replay accuracy

---

## 3. Architecture Overview

### 3.1 Current State

- Timeouts: Global env vars (`ELMAKINA_CHALLENGE_TIMEOUT`, etc.)
- Action Costs: Hardcoded in `backend/actions/registry.go`
- Game Config: Only sends available roles to frontend
- No per-game configuration exists

### 3.2 Target State

```
Lobby Creation → Configure Rules → Start Game
                      ↓
              Rules stored in GameSession
                      ↓
              Passed to engine.Game
                      ↓
              Used throughout game lifecycle
```

---

## 4. Data Models

### 4.1 GameRules Configuration

**File:** `backend/engine/gamerules.go` (NEW)

```go
package engine

import "time"

// GameRules contains all customizable game parameters.
// These are set per-game by the lobby leader before match start.
type GameRules struct {
    // Timing Configuration
    ChallengeTimeout time.Duration `json:"challenge_timeout_ms"` // e.g., 15000
    CounterTimeout   time.Duration `json:"counter_timeout_ms"`   // e.g., 15000
    TurnTimeout      time.Duration `json:"turn_timeout_ms"`      // e.g., 20000
    
    // Extend Feature
    ExtendUsesPerPlayer int `json:"extend_uses_per_player"` // default: 3
    ExtendDuration      time.Duration `json:"extend_duration_ms"` // e.g., 15000
    
    // Economy Configuration
    ActionCosts   map[string]int `json:"action_costs"`   // action_id -> coin cost
    StartingCoins int            `json:"starting_coins"` // default: 2
    MaxCoins      int            `json:"max_coins"`      // default: 12
    IncomeAmount  int            `json:"income_amount"`  // default: 1
    
    // Deck Configuration
    CardsPerRole int `json:"cards_per_role"` // default: 3
    
    // Gameplay Variants
    AllowSelfTarget      bool `json:"allow_self_target"`       // default: false
    RevealCardOnDeath    bool `json:"reveal_card_on_death"`    // default: true
    AnonymousChallenges  bool `json:"anonymous_challenges"`    // default: false
    ForeignAidBlockable  bool `json:"foreign_aid_blockable"`   // default: true
    
    // Advanced Variants
    TeamMode             bool   `json:"team_mode"`              // default: false
    Teams                []Team `json:"teams,omitempty"`        // only if team_mode=true
    DoubleActions        bool   `json:"double_actions"`         // default: false - two actions per turn
    RandomizedCosts      bool   `json:"randomized_costs"`       // default: false - shuffle costs each game
    PeekingAllowed       bool   `json:"peeking_allowed"`        // default: false - look at top deck card
    BountySystem         bool   `json:"bounty_system"`          // default: false - reward for successful challenges
    SuddenDeathTurn      int    `json:"sudden_death_turn"`      // default: 0 (disabled) - after this turn, all costs = 1
}

type Team struct {
    ID      string   `json:"id"`
    Name    string   `json:"name"`
    Color   string   `json:"color"`   // hex color
    Players []string `json:"players"` // player IDs
}

// DefaultGameRules returns the standard game rules matching current behavior.
func DefaultGameRules() GameRules {
    return GameRules{
        // Timing
        ChallengeTimeout:    15 * time.Second,
        CounterTimeout:      15 * time.Second,
        TurnTimeout:         20 * time.Second,
        
        // Extend
        ExtendUsesPerPlayer: 3,
        ExtendDuration:      15 * time.Second,
        
        // Economy
        ActionCosts: map[string]int{
            "income":       0,
            "foreign_aid":  0,
            "coup":         7,
            "tax":          0,
            "assassinate":  3,
            "exchange":     0,
            "steal":        0,
            "business":     0,
            "investigate":  0,
            "accuse":       4,
            "escape":       9,
        },
        StartingCoins:   2,
        MaxCoins:        12,
        IncomeAmount:    1,
        
        // Deck
        CardsPerRole:    3,
        
        // Core Variants
        AllowSelfTarget:      false,
        RevealCardOnDeath:    true,
        AnonymousChallenges:  false,
        ForeignAidBlockable:  true,
        
        // Advanced Variants (all disabled by default)
        TeamMode:             false,
        Teams:                nil,
        DoubleActions:        false,
        RandomizedCosts:      false,
        PeekingAllowed:       false,
        BountySystem:         false,
        SuddenDeathTurn:      0,
    }
}

// Validate checks that rules are within acceptable bounds.
func (r GameRules) Validate() error {
    // Challenge/counter/turn timeouts: 5s - 120s
    // Extend uses: 0 - 10
    // Extend duration: 5s - 60s
    // Action costs: 0 - 20
    // Starting coins: 0 - 10
    // Max coins: 5 - 30
    // Cards per role: 1 - 5
    return nil
}

// ToTurnConfig converts GameRules to the existing TurnConfig.
func (r GameRules) ToTurnConfig() TurnConfig {
    return TurnConfig{
        ChallengeTimeout: r.ChallengeTimeout,
        CounterTimeout:   r.CounterTimeout,
        TurnTimeout:      r.TurnTimeout,
    }
}
```

### 4.2 Updated GameSession

**File:** `backend/server/session.go`

```go
type GameSession struct {
    MatchID       string
    LobbyID       string
    Game          *engine.Game
    PlayerIDs     []string
    IndexByPlayer map[string]int
    PlayerAvatars []string
    RNGSeed       string
    Rules         engine.GameRules        // NEW: per-game configuration
    ExtendUsed    map[string]int          // NEW: player_id -> times used
}
```

### 4.3 Updated Game State

**File:** `backend/engine/game.go`

```go
type Game struct {
    CurrentState state.GameState
    TurnState    state.TurnState
    History      []state.GameState
    ActiveFlow   *actions.FlowRunner
    rng          *rand.Rand
    Rules        engine.GameRules        // NEW: rules for this game
}
```

---

## 5. WebSocket Protocol Changes

### 5.1 New Message Types

**File:** `backend/server/ws/messages.go`

```go
// Client → Server: Configure lobby before start
type LobbyConfigurePayload struct {
    Rules engine.GameRules `json:"rules"`
}

// Client → Server: Request time extension
type ExtendRequestPayload struct {
    WindowType string `json:"window_type"` // "challenge" or "counter"
}

// Server → Client: Broadcast when someone extends
type ExtendUsedPayload struct {
    PlayerID      string `json:"player_id"`
    PlayerName    string `json:"player_name"`
    WindowType    string `json:"window_type"`
    NewDurationMs int    `json:"new_duration_ms"`
    RemainingUses int    `json:"remaining_uses"`
}

// Server → Client: Enhanced game config with rules
type GameConfigPayload struct {
    Roles       []string       `json:"roles"`
    Rules       engine.GameRules `json:"rules"`
    ExtendLeft  map[string]int `json:"extend_left"` // player_id -> uses remaining
}

// Server → Client: Current timer state with extension info
type TurnTimerPayload struct {
    Type         string `json:"type"`          // "challenge_window" | "counter_window" | "turn"
    DurationMs   int    `json:"duration_ms"`
    RemainingMs  int    `json:"remaining_ms"`
    CanExtend    bool   `json:"can_extend"`    // NEW
    ExtendsLeft  int    `json:"extends_left"`  // NEW
}
```

### 5.2 Message Flow

```
1. Lobby Created
   Server → All: lobby_state { status: "open", can_configure: true }

2. Leader Configures
   Client → Server: lobby_configure { rules: {...} }
   Server validates rules
   Server → All: lobby_state { status: "open", rules: {...} }

3. Game Starts
   Server → All: game_config { roles, rules, extend_left: {player1: 3, ...} }

4. Challenge Window Opens
   Server → All: challenge_window { duration_ms: 15000, can_extend: true, extends_left: 3 }

5. Player Extends
   Client → Server: extend_request { window_type: "challenge" }
   Server validates (has extends left, window still open)
   Server → All: extend_used { player_id, new_duration_ms: 30000, remaining_uses: 2 }
   Server → All: Updated turn_timer with new remaining_ms

6. Game Ends
   Server → All: game_over { winner, stats }
```

---

## 6. Backend Implementation Details

### 6.1 Engine Changes

**File:** `backend/engine/game.go`

```go
// Updated constructor signature
func NewGame(playerNames []string, rng *rand.Rand, rules GameRules) (*Game, error) {
    // ... existing validation
    
    // Use rules.StartingCoins instead of hardcoded 2
    // Use rules.CardsPerRole when creating deck
    // Store rules in Game struct
    
    return &Game{
        CurrentState: gs,
        TurnState:    state.TurnState{},
        History:      make([]state.GameState, 0),
        rng:          rng,
        Rules:        rules,  // Store for runtime access
    }, nil
}

// GetActionCost retrieves cost from rules, fallback to 0
func (g *Game) GetActionCost(actionID models.ActionID) int {
    if cost, ok := g.Rules.ActionCosts[string(actionID)]; ok {
        return cost
    }
    return 0
}
```

**File:** `backend/engine/state/deck.go`

```go
// Updated to accept cards per role from rules
func NewDeck(cardsPerRole int, rng *rand.Rand) Deck {
    // Use cardsPerRole parameter instead of hardcoded 3
}
```

**File:** `backend/engine/orchestrator_turn.go`

```go
// Use g.Rules.ChallengeTimeout, g.Rules.CounterTimeout instead of cfg param
// Consider removing TurnConfig entirely in favor of GameRules
```

### 6.2 Server Changes

**File:** `backend/server/lobby.go`

```go
// Add to Lobby struct
type Lobby struct {
    // ... existing fields
    Rules      engine.GameRules  // Configuration for this lobby
    ExtendUsed map[string]int    // Track extends per player
}

// New method: ConfigureLobby
func (m *LobbyManager) ConfigureLobby(lobbyID, leaderID string, rules engine.GameRules) error {
    // 1. Validate lobby exists and is open
    // 2. Verify requester is leader
    // 3. Validate rules using rules.Validate()
    // 4. Store rules in lobby
    // 5. Broadcast updated lobby state
}

// Update StartLobby to use configured rules
func (m *LobbyManager) StartLobby(lobbyID, leaderID string) (*GameSession, error) {
    // ... existing validation
    
    // Get rules from lobby or use defaults
    rules := lobby.Rules
    if rules.ChallengeTimeout == 0 {
        rules = engine.DefaultGameRules()
    }
    
    // Pass rules to NewGame
    game, err := engine.NewGame(names, matchRNG, rules)
    
    // Create session with rules and extend tracking
    session := &GameSession{
        // ... existing fields
        Rules:      rules,
        ExtendUsed: make(map[string]int),
    }
    
    // Initialize extend counts
    for _, pid := range playerIDs {
        session.ExtendUsed[pid] = rules.ExtendUsesPerPlayer
    }
}
```

**File:** `backend/server/ws/session_runner.go`

```go
// Add extend handling
func (sr *sessionRunner) handleExtendRequest(playerID string, req ExtendRequestPayload) error {
    // 1. Check if player has extends left
    if sr.session.ExtendUsed[playerID] <= 0 {
        return fmt.Errorf("no extends remaining")
    }
    
    // 2. Check if window is currently open
    if !sr.isWindowOpen(req.WindowType) {
        return fmt.Errorf("no active %s window", req.WindowType)
    }
    
    // 3. Decrement extend count
    sr.session.ExtendUsed[playerID]--
    
    // 4. Extend current timeout
    sr.extendCurrentWindow(sr.session.Rules.ExtendDuration)
    
    // 5. Broadcast to all players
    sr.broadcastExtendUsed(playerID, req.WindowType)
    
    return nil
}
```

### 6.3 WebSocket Handler Changes

**File:** `backend/server/ws/server.go`

```go
// New handler methods
func (s *Server) handleLobbyConfigure(client *Client, payload []byte) error {
    var req LobbyConfigurePayload
    if err := json.Unmarshal(payload, &req); err != nil {
        return err
    }
    
    lobby := s.lobbies[client.LobbyID]
    return s.manager.ConfigureLobby(lobby.ID, client.ID, req.Rules)
}

func (s *Server) handleExtendRequest(client *Client, payload []byte) error {
    var req ExtendRequestPayload
    if err := json.Unmarshal(payload, &req); err != nil {
        return err
    }
    
    session := s.sessions[client.SessionID]
    return session.handleExtendRequest(client.ID, req)
}
```

---

## 7. Frontend Implementation (High Level)

### 7.1 Lobby Configuration UI

**New Component:** `web/src/components/lobby/GameConfigPanel.tsx`

```typescript
interface GameConfigPanelProps {
  isLeader: boolean;
  currentRules: GameRules;
  onSave: (rules: GameRules) => void;
}

// Sections:
// 1. Timing: Sliders for challenge/counter/turn timeouts (5s - 60s)
// 2. Economy: Number inputs for each action cost
// 3. Extend: Slider for uses per player (0 - 5), extend duration
// 4. Variants: Toggle for allow self-target, etc.
// 5. Presets: Buttons for "Default", "Speed Mode", "Chaos Mode"
```

### 7.2 Extend Button Component

**New Component:** `web/src/components/game/ExtendButton.tsx`

```typescript
interface ExtendButtonProps {
  windowType: 'challenge' | 'counter';
  canExtend: boolean;
  extendsLeft: number;
  onExtend: () => void;
  theme: 'challenge' | 'counter'; // Themed styling
}

// Shows:
// - Hourglass icon + "EXTEND" text
// - Badge with remaining uses
// - Disabled state when uses = 0
// - Themed colors (red for challenge, blue for counter)
```

### 7.3 State Management

**Update:** `web/src/state/gameStore.ts`

```typescript
interface GameState {
  // ... existing fields
  rules: GameRules;
  extendLeft: Record<string, number>; // player_id -> uses left
  currentWindow: {
    type: 'challenge' | 'counter' | null;
    canExtend: boolean;
    extendsLeft: number;
  };
}
```

---

## 8. Database Schema (For Replays)

**File:** `backend/server/replay/` (existing)

```sql
-- Add to replay_games table or create new table
CREATE TABLE game_configurations (
    match_id UUID PRIMARY KEY REFERENCES replay_games(match_id),
    
    -- Timing
    challenge_timeout_ms INTEGER NOT NULL,
    counter_timeout_ms INTEGER NOT NULL,
    turn_timeout_ms INTEGER NOT NULL,
    
    -- Extend
    extend_uses_per_player INTEGER NOT NULL,
    extend_duration_ms INTEGER NOT NULL,
    
    -- Economy
    action_costs JSONB NOT NULL, -- {"income": 0, "coup": 7, ...}
    starting_coins INTEGER NOT NULL,
    max_coins INTEGER NOT NULL,
    income_amount INTEGER NOT NULL DEFAULT 1,
    
    -- Deck
    cards_per_role INTEGER NOT NULL,
    
    -- Core Variants
    allow_self_target BOOLEAN NOT NULL DEFAULT false,
    reveal_card_on_death BOOLEAN NOT NULL DEFAULT true,
    anonymous_challenges BOOLEAN NOT NULL DEFAULT false,
    foreign_aid_blockable BOOLEAN NOT NULL DEFAULT true,
    
    -- Advanced Variants
    team_mode BOOLEAN NOT NULL DEFAULT false,
    teams JSONB, -- [{"id": "red", "name": "Team A", ...}]
    double_actions BOOLEAN NOT NULL DEFAULT false,
    randomized_costs BOOLEAN NOT NULL DEFAULT false,
    peeking_allowed BOOLEAN NOT NULL DEFAULT false,
    bounty_system BOOLEAN NOT NULL DEFAULT false,
    sudden_death_turn INTEGER DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT NOW()
);

-- Index for team mode queries
CREATE INDEX idx_game_configs_team_mode ON game_configurations(team_mode) WHERE team_mode = true;
```

---

## 9. Validation Rules

All configurations must pass validation before game start:

### 9.1 Core Timing & Economy

| Field | Min | Max | Default | Notes |
|-------|-----|-----|---------|-------|
| ChallengeTimeout | 5s | 120s | 15s | |
| CounterTimeout | 5s | 120s | 15s | |
| TurnTimeout | 10s | 300s | 20s | Must be > challenge+counter |
| ExtendUsesPerPlayer | 0 | 10 | 3 | 0 = disabled |
| ExtendDuration | 5s | 60s | 15s | |
| ActionCosts (each) | 0 | 20 | varies | Income should stay 0 |
| StartingCoins | 0 | 10 | 2 | |
| MaxCoins | 5 | 30 | 12 | Must be >= starting + max income |
| IncomeAmount | 1 | 5 | 1 | Affects economic pacing |
| CardsPerRole | 1 | 5 | 3 | Affects deck size |

### 9.2 Boolean Variants

| Field | Default | Notes |
|-------|---------|-------|
| AllowSelfTarget | false | Suicide mechanic |
| RevealCardOnDeath | true | Information meta |
| AnonymousChallenges | false | Social dynamics |
| ForeignAidBlockable | true | Tax Collector power |
| TeamMode | false | Requires team configuration |
| DoubleActions | false | Complex interactions |
| RandomizedCosts | false | Chaos mode |
| PeekingAllowed | false | Information edge |
| BountySystem | false | Challenge incentive |

### 9.3 Integer Variants

| Field | Min | Max | Default | Notes |
|-------|-----|-----|---------|-------|
| SuddenDeathTurn | 0 | 50 | 0 | 0 = disabled |
| TimeBankSeconds | 0 | 600 | 0 | 0 = disabled |

### 9.4 Team Mode Validation

When TeamMode = true:
- Teams array must be non-empty
- All players must be assigned to exactly one team
- Teams should be balanced (within 1 player difference)
- Minimum 2 teams
- Team names must be unique
- Team colors must be unique

**Validation Logic:**
- Turn timeout must accommodate at least one challenge + counter cycle
- Max coins must be >= starting coins + max single-turn income
- At least one action must have cost 0 (income/foreign aid)
- If DoubleActions = true, TurnTimeout should be increased by 50%
- If TeamMode = true, validate team configuration thoroughly
- SuddenDeathTurn must be > 0 if enabled, and reasonable for game length

---

## 10. Additional Fun Configuration Options

### 10.1 Tier 1: Core Configurations (Already in Model)

1. **Starting Coins** (0-10, default: 2) - Higher starts = faster early aggression
2. **Cards Per Role** (1-5, default: 3) - Fewer cards = more predictable deck
3. **Income Amount** (1-5, default: 1) - Higher income = faster economic game
4. **Max Coins Cap** (5-30, default: 12) - Lower cap = forces spending
5. **Allow Self-Target** (bool, default: false) - Enable suicide mechanic

### 10.2 Tier 2: Information & Social

6. **Reveal Card On Death** (bool, default: true) - Show lost card to all players
7. **Anonymous Challenges** (bool, default: false) - Hide who issued the challenge
8. **Foreign Aid Blockable** (bool, default: true) - Can Tax Collector block Foreign Aid?
9. **Show Remaining Deck Count** (bool, default: true) - Display cards left in deck
10. **Role Reveal On Elimination** (bool, default: true) - Show all cards when player dies

### 10.3 Tier 3: Advanced Gameplay Variants

11. **Team Mode** (bool, default: false) - Players form teams, team wins when all opponents eliminated
    - **Team Configuration**: 2v2, 3v3, 2v2v2, 3v3v3 (must be even teams)
    - **Shared Victory**: Team wins together
    - **Team Chat**: Private team communication channel
    - **Revival Mechanic**: Teammate can sacrifice card to revive ally (optional)
    - **See Section 10.4 for full Team Mode specification**

12. **Double Actions** (bool, default: false) - Each turn, player can perform 2 actions
    - Action costs still apply
    - Challenge/counter windows happen for each action separately
    - Creates combo opportunities (e.g., Steal + Investigate)

13. **Randomized Costs** (bool, default: false) - Shuffle all action costs at game start
    - Each game has different economy
    - Players see costs in game_config
    - Creates "economic chaos" mode

14. **Peeking Allowed** (bool, default: false) - Once per turn, look at top deck card
    - Adds information edge
    - Must be declared publicly
    - Creates new mind games

15. **Bounty System** (bool, default: false) - Successful challenger gets reward
    - 1-2 coins for successful challenge
    - Creates incentive to challenge bluffs
    - Rewards reading opponents

16. **Sudden Death Mode** (turn number, default: 0=disabled) - After N turns, all action costs = 1
    - Accelerates endgame
    - Forces action
    - Prevents infinite stalls

17. **Handicap Mode** (map[player]int, default: none) - Some players start with extra coins/cards
    - Balances skill differences
    - Leader can assign handicaps
    - Great for teaching new players

18. **Mercenary Mode** (bool, default: false) - Defeated players become "mercenaries"
    - Can still influence game with 1 action per turn
    - No cards, only coins (earn from challenges)
    - Can be hired by living players

19. **Draft Mode** (bool, default: false) - Players draft their starting cards
    - Take turns picking from face-up selection
    - More strategic deck building
    - Slower setup but higher skill ceiling

20. **Kingmaker Mode** (bool, default: false) - Last eliminated player chooses winner
    - Creates social dynamics
    - Revenge mechanic
    - Prevents kingmaking by strong players

21. **Time Bank** (int seconds, default: 0) - Each player has total time pool for entire game
    - Extends come from time bank
    - Run out = auto-pass on your turns
    - Adds strategic time management

22. **Sudden Revelation** (bool, default: false) - At random turn, all cards are revealed
    - Creates mid-game information shift
    - Configurable: turn 3, 5, or random
    - Bluffers must survive until revelation

23. **Economy Shock** (interval, default: 0) - Every N turns, random cost change
    - Market volatility mechanic
    - Forces adaptation
    - High chaos factor

24. **Tournament Mode** (bool, default: false) - Strict competitive settings
    - Locked to default rules
    - No extends allowed
    - Standard timing
    - For ranked play

25. **Wildcard Role** (bool, default: false) - Add "Wildcard" role with random ability
    - Draw random card each time it's revealed
    - High variance
    - Fun/casual mode

26. **Chain Challenges** (bool, default: false) - Successfully challenged player can challenge back
    - Escalation mechanic
    - Risk/reward chain
    - Dramatic reversals

27. **Card Trading** (bool, default: false) - Players can trade cards with each other
    - Requires both players to agree
    - Adds negotiation layer
    - Team mode synergy

28. **Open Discourse** (bool, default: false) - No restrictions on table talk
    - Allow alliances mid-game
    - Verbal bluffing encouraged
    - Social game intensifier

### 10.4 Team Mode Specification

**Overview:**
Team Mode transforms El Makina from a free-for-all into a team-based strategy game where coordination and trust matter.

**Configuration:**
```go
TeamMode: true
Teams: [
  {ID: "red", Name: "The Syndicate", Color: "#FF4444", Players: ["p1", "p2"]},
  {ID: "blue", Name: "The Coalition", Color: "#4444FF", Players: ["p3", "p4"]}
]
TeamSettings: {
  SharedVictory: true,        // Team wins together
  PrivateChat: true,          // Team-only WebSocket channel
  RevivalEnabled: false,      // Can revive teammates?
  TeamVision: false,          // See teammate's cards?
  CrossTeamAssistance: false  // Can help other teams?
}
```

**Gameplay Changes:**
1. **Win Condition**: Team wins when all opposing team players are eliminated
2. **Targeting**: Cannot target own team members (unless AllowSelfTarget=true)
3. **Information Sharing**: Optional "Team Vision" to see teammate's cards
4. **Team Chat**: Private WebSocket channel for coordination
5. **Revival Mechanic** (optional): Teammate can sacrifice a card to bring eliminated ally back with 1 card

**UI Changes:**
- Color-coded player names by team
- Team indicator in player list
- Team chat tab separate from global chat
- Victory screen shows team celebration

**Strategic Depth:**
- Coordinate bluffs with teammates
- Protect weaker teammates
- Trade information via team chat
- Sacrifice plays (take hit for teammate)
- Feints and coordinated attacks

**Variants:**
- **2v2**: Classic team mode
- **3v3**: Larger team coordination
- **2v2v2**: Three teams, diplomacy matters
- **1v1v1v1v1v1**: Teams of 1 (individual but can form temporary alliances)
- **Captain Mode**: One "captain" per team makes all decisions

### 10.5 Action-Specific Configurations

**Per-Action Toggles:**
29. **Accuse Reveal** (bool, default: true) - Does accuser reveal their card?
30. **Investigate Reveal** (bool, default: false) - Does target reveal investigated card to all?
31. **Steal Maximum** (int, default: 2) - Max coins that can be stolen
32. **Tax Collector Guarantee** (bool, default: false) - Tax Collector always gets coins even if blocked
33. **Escape Cost Scaling** (bool, default: false) - Escape cost increases each use (9, 11, 13...)
34. **Double Business** (bool, default: false) - Business action earns 2 coins instead of 1
35. **Exchange Count** (int, default: 2) - How many cards drawn during Exchange (2-5)

---

## 11. Implementation Phases

### Phase 1: Foundation (Week 1)
- [ ] Create `GameRules` struct and validation
- [ ] Update `GameSession` to store rules
- [ ] Update `engine.NewGame()` signature
- [ ] Update `TurnConfig` to use `GameRules`
- [ ] Update deck creation to use `CardsPerRole`
- [ ] Update action cost lookups to use rules

### Phase 2: WebSocket Protocol (Week 1-2)
- [ ] Add new message types
- [ ] Implement `lobby_configure` handler
- [ ] Update `game_config` broadcast
- [ ] Implement `extend_request` handler
- [ ] Add `extend_used` broadcast
- [ ] Update `turn_timer` with extend info

### Phase 3: Lobby Integration (Week 2)
- [ ] Add `ConfigureLobby` method to LobbyManager
- [ ] Initialize extend counts on game start
- [ ] Track extend usage per player
- [ ] Wire up extend button to session runner
- [ ] Update session provider to handle extends

### Phase 4: Frontend UI (Week 2-3)
- [ ] Create GameConfigPanel component
- [ ] Add timing sliders
- [ ] Add action cost inputs
- [ ] Add extend configuration
- [ ] Create ExtendButton component
- [ ] Add themed styling (challenge vs counter)
- [ ] Update game store with extend state

### Phase 5: Persistence (Week 3)
- [ ] Add game_configuration table to DB
- [ ] Store rules when game starts
- [ ] Load rules for replays
- [ ] Add extend usage to replay events

### Phase 6: Testing & Polish (Week 4)
- [ ] Unit tests for GameRules validation
- [ ] Integration tests for extend feature
- [ ] Test various configuration combinations
- [ ] UI/UX polish
- [ ] Add preset configurations (Speed, Chaos)

---

## 12. Open Questions

### 12.1 Core Questions

1. **Should presets be hardcoded or user-savable?**
   - MVP: Hardcoded (Default, Speed, Chaos)
   - Future: User-created presets stored in localStorage

2. **Should we allow mid-game rule changes?**
   - Recommendation: No, only before start
   - Exception: Maybe allow leader to pause and adjust?

3. **How do we handle invalid configurations gracefully?**
   - Option A: Reject with detailed error
   - Option B: Clamp to valid range with warning
   - Recommendation: Option A for clarity

4. **Should extend count reset per window or global?**
   - Current plan: Global per player (3 total)
   - Alternative: 3 per window type (3 challenge + 3 counter)
   - Recommendation: Global for simplicity

5. **Visual indicator for low extends?**
   - Show warning when 1 left?
   - Show total used in end-game stats?

### 12.2 Team Mode Questions

6. **Team chat: WebSocket channel or just UI filtering?**
   - Separate channel: cleaner, but more connections
   - UI filtering: simpler, but messages still technically visible

7. **Can teams have uneven numbers?**
   - 3v2 allowed? Creates handicap
   - Recommendation: Require balance within 1 player

8. **Team formation: Random or manual assignment?**
   - Leader assigns teams
   - Random fair teams
   - Players pick teams

9. **What happens when teammate disconnects?**
   - Auto-eliminate both?
   - Allow replacement player?
   - Pause game for reconnect?

10. **Team victory celebration?**
    - Shared victory screen
    - Individual stats within team
    - Team MMR/rating?

### 12.3 Advanced Variant Questions

11. **Randomized costs: How random?**
    - Shuffle existing costs
    - Pure random 0-10 range
    - Ensure at least one free action

12. **Double actions: Sequential or parallel?**
    - One after another (challenge each)
    - Both declared, resolved together

13. **Bounty amount: Fixed or scaling?**
    - Always 1 coin
    - Based on turn number
    - Based on target's coins

14. **Sudden Death: Warning turn?**
    - Announce at turn N-1
    - Surprise sudden death
    - Configurable warning

15. **Peeking: Public or private?**
    - Only peeker sees
    - Broadcast "Player X peeked" (but not result)
    - Full public reveal

16. **Handicap mode: Who assigns?**
    - Lobby leader assigns before start
    - Detect "new" players automatically
    - Players self-declare skill level

---

## 13. Success Metrics

### 13.1 Technical Performance
- Configuration panel loads in <500ms
- Rules validation completes in <100ms
- Extend request processed in <50ms
- Zero crashes with extreme configurations (0s timeout, 20 costs)
- Database query for game config <20ms

### 13.2 User Engagement
- Players use extend feature in 30%+ of games
- 50%+ of games use non-default configuration
- Team Mode used in 20%+ of 4+ player games
- Average 2.5 different presets tried per player
- Configuration panel viewed by 80%+ of lobby leaders

### 13.3 Game Balance
- No single configuration has >60% win rate for any strategy
- Sudden Death games end 20% faster than normal
- Team Mode games have <5% abandonment rate
- Randomized Costs games report "fun" in 70%+ of post-game surveys

### 13.4 Stability
- Zero configuration-related game crashes
- <1% invalid configuration rejection rate
- 100% replay accuracy with custom rules
- Team chat latency <100ms

---

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Unbalanced configurations break game | High | Medium | Strict validation, cost constraints, presets |
| Extend feature abused (spam) | Medium | Low | Limited uses, server-side enforcement |
| Frontend/backend desync on rules | Medium | High | Single source of truth, server validates all |
| Database migration issues | Low | High | Test thoroughly, rollback plan, migrations |
| Complex UI confuses users | Medium | Medium | Presets, tooltips, progressive disclosure |
| Team Mode complexity (coordination) | High | High | Extensive testing, clear UI indicators |
| Double Actions timing issues | Medium | Medium | Increase timeout recommendations, buffer time |
| Randomized Costs break balance | High | Low | Validation prevents extreme randomization |
| Sudden Death anti-climactic | Medium | Low | Configurable trigger turn, clear warning |
| Bounty System economy inflation | Medium | Medium | Cap bounty amount, limit frequency |
| Card Trading abuse (collusion) | Medium | Medium | Only allow fair trades, log all trades |
| Mercenary Mode confusing | Medium | Low | Clear UI, tutorial, optional feature |

---

## 15. Appendix: Preset Configurations

### Default (Current Behavior)
```json
{
  "challenge_timeout_ms": 15000,
  "counter_timeout_ms": 15000,
  "turn_timeout_ms": 20000,
  "extend_uses_per_player": 3,
  "extend_duration_ms": 15000,
  "action_costs": {
    "income": 0,
    "foreign_aid": 0,
    "coup": 7,
    "tax": 0,
    "assassinate": 3,
    "exchange": 0,
    "steal": 0,
    "business": 0,
    "investigate": 0,
    "accuse": 4,
    "escape": 9
  },
  "starting_coins": 2,
  "max_coins": 12,
  "income_amount": 1,
  "cards_per_role": 3,
  "allow_self_target": false,
  "reveal_card_on_death": true,
  "anonymous_challenges": false,
  "foreign_aid_blockable": true
}
```

### Speed Mode
```json
{
  "challenge_timeout_ms": 5000,
  "counter_timeout_ms": 5000,
  "turn_timeout_ms": 10000,
  "extend_uses_per_player": 1,
  "extend_duration_ms": 5000,
  "starting_coins": 5,
  "max_coins": 20,
  "income_amount": 2,
  "cards_per_role": 2,
  "action_costs": {
    "coup": 5,
    "assassinate": 2,
    "accuse": 2,
    "escape": 5
  }
}
```

### Chaos Mode (Randomized)
```json
{
  "challenge_timeout_ms": 20000,
  "counter_timeout_ms": 20000,
  "extend_uses_per_player": 5,
  "randomized_costs": true,
  "allow_self_target": true,
  "anonymous_challenges": true,
  "double_actions": true,
  "peeking_allowed": true,
  "bounty_system": true,
  "starting_coins": 5,
  "max_coins": 25
}
```

### Minimalist (Predictable)
```json
{
  "cards_per_role": 1,
  "action_costs": {
    "coup": 5,
    "assassinate": 2,
    "accuse": 2
  },
  "starting_coins": 5,
  "income_amount": 2,
  "reveal_card_on_death": true
}
```

### Team Mode (2v2)
```json
{
  "challenge_timeout_ms": 20000,
  "counter_timeout_ms": 20000,
  "extend_uses_per_player": 3,
  "team_mode": true,
  "teams": [
    {"id": "red", "name": "The Syndicate", "color": "#FF4444", "players": ["p1", "p2"]},
    {"id": "blue", "name": "The Coalition", "color": "#4444FF", "players": ["p3", "p4"]}
  ],
  "starting_coins": 3,
  "cards_per_role": 3,
  "allow_self_target": false
}
```

### Tournament Mode (Competitive)
```json
{
  "challenge_timeout_ms": 15000,
  "counter_timeout_ms": 15000,
  "turn_timeout_ms": 20000,
  "extend_uses_per_player": 0,
  "tournament_mode": true,
  "anonymous_challenges": false,
  "reveal_card_on_death": true,
  "foreign_aid_blockable": true,
  "starting_coins": 2,
  "max_coins": 12,
  "cards_per_role": 3
}
```

### Sudden Death Mode
```json
{
  "challenge_timeout_ms": 15000,
  "counter_timeout_ms": 15000,
  "turn_timeout_ms": 20000,
  "extend_uses_per_player": 2,
  "sudden_death_turn": 10,
  "starting_coins": 2,
  "max_coins": 12,
  "action_costs": {
    "coup": 7,
    "assassinate": 3,
    "accuse": 4
  }
}
```

### Teaching Mode (Beginner Friendly)
```json
{
  "challenge_timeout_ms": 30000,
  "counter_timeout_ms": 30000,
  "turn_timeout_ms": 45000,
  "extend_uses_per_player": 5,
  "extend_duration_ms": 20000,
  "starting_coins": 5,
  "income_amount": 2,
  "max_coins": 15,
  "cards_per_role": 4,
  "reveal_card_on_death": true,
  "anonymous_challenges": false,
  "handicap_mode": {
    "enabled": true,
    "new_player_bonus": 2
  }
}
```

---

**Document End**
