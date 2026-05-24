package game

import (
	"errors"
	"fmt"
	"slices"
)

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

// Roles is the set of all available roles in the game.
var Roles = []Role{
	Businesswoman,
	TaxCollector,
	Policewoman,
	Colonel,
	Terrorist,
	Thief,
	Politician,
}

// ParseRole converts a string to a Role, validating it against Roles.
func ParseRole(value string) (Role, error) {
	role := Role(value)
	if slices.Contains(Roles, role) {
		return role, nil
	}
	return "", fmt.Errorf("%w: %s", errors.New("invalid role"), value)
}
