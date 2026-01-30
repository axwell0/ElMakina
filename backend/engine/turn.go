package engine

import (
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
)

const NoPlayer = -1 // Sentinel value for "no player selected"

// ChallengeDecision encapsulates the outcome of a challenge window.
// Either a player challenges (with discard indices) or everyone passes.
type ChallengeDecision struct {
	ChallengerIndex        int  // Index of challenging player, or NoPlayer if no challenge
	ChallengerDiscardIndex int  // Card index challenger discards if they lose
	ActorDiscardIndex      int  // Card index actor discards if they bluffed
	ActorProvingCardIndex  int  // Card index actor reveals and swaps if they prove role
	Pass                   bool // Deprecated: use ChallengerIndex == NoPlayer instead
}

// ChallengeOutcome represents the possible final states of a challenge resolution.
type ChallengeOutcome string

const (
	OutcomeActorProved  ChallengeOutcome = "actor_proved"  // Actor had the card, challenger loses
	OutcomeActorBluffed ChallengeOutcome = "actor_bluffed" // Actor was lying, actor loses
	OutcomeNoChallenge  ChallengeOutcome = "no_challenge"  // Window closed without any challenges
)

// ChallengeResult is the outcome of resolving a challenge. Contains logs,
// who lost a card, and whether the action proceeds.
type ChallengeResult struct {
	Success         bool             // True if actor was bluffing (challenger won)
	ActionAllowed   bool             // Whether the challenged action can proceed
	ActorIndex      int              // Player who was challenged
	ChallengerIndex int              // Player who challenged (or NoPlayer)
	ClaimedRole     models.Role      // Role that was claimed
	Outcome         ChallengeOutcome // Result of the challenge resolution
	Logs            []string         // Public log lines for this challenge
	LostCard        *state.Card      // Card that was discarded (nil if none)
	LostPlayerIndex int              // Index of player who lost card (-1 if none)
}

// CounterDecision is a player's response to a counter window.
// Either pass or provide a counter action command.
type CounterDecision struct {
	PlayerIndex int                  // Player making this decision
	Pass        bool                 // True if player passes
	Command     *models.PlayerAction // Counter action if not passing
}

// TurnResult is the complete output of a turn execution. Contains logs,
// action outcomes, and challenge/counter results.
type TurnResult struct {
	Logs             []string             // Public log lines
	PrivateLogs      map[int][]string     // Per-player private logs (investigate reveals, etc.)
	Main             *models.PlayerAction // Main action that was requested
	MainApplied      bool                 // True if main action executed
	MainCanceled     bool                 // True if main action was blocked
	CounterResults   []CounterResult      // Outcomes of all counter attempts
	ChallengeResults []ChallengeResult    // Outcomes of all challenges
}

// CounterResult tracks the outcome of a single counter action.
type CounterResult struct {
	Action    *models.PlayerAction // Counter action command
	Applied   bool                 // True if counter executed
	Canceled  bool                 // True if counter was challenged successfully
	Challenge *ChallengeResult     // Challenge outcome if counter was challenged
}
