package models

import "fmt"

// Role is the "Vocabulary" of the game.
// Everything in El Makina—from the deck to the challenge logic—refers back to these types.
// We use strings here intentionally: it makes the JSON that goes over the wire
// human-readable and easy to debug, while still giving us a type-safe way
// to handle logic in the engine.
//
// See: engine/state/deck.go for role assignment.
// See: actions/registry.go for role-action mapping.
// See: orchestrator_turn.go:79 for challenge validation.
type Role string

const (
	Businesswoman Role = "Businesswoman"
	TaxCollector  Role = "TaxCollector"
	Policewoman   Role = "Policewoman"
	Colonel       Role = "Colonel"
	Terrorist     Role = "Terrorist"
	Thief         Role = "Thief"
	Politician    Role = "Politician"
)

// Roles slice enables iteration and validation. Used by ParseRole() and when
// dealing cards from the deck.
var Roles = []Role{
	Businesswoman,
	TaxCollector,
	Policewoman,
	Colonel,
	Terrorist,
	Thief,
	Politician,
}

// ParseRole validates and converts a string to Role. Called during WebSocket
// message parsing to reject invalid role values before they reach the engine.
func ParseRole(value string) (Role, error) {
	role := Role(value)
	for _, validRole := range Roles {
		if role == validRole {
			return role, nil
		}
	}
	return "", fmt.Errorf("invalid role: %s", value)
}
