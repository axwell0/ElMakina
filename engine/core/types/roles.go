package types

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

var Roles = []Role{
	Businesswoman,
	TaxCollector,
	Policewoman,
	Colonel,
	Terrorist,
	Thief,
	Politician,
}

var ErrInvalidRole = errors.New("invalid role")

// ParseRole takes in a string and resolves it to a role.
func ParseRole(value string) (Role, error) {
	role := Role(value)
	if slices.Contains(Roles, role) {
		return role, nil
	}
	return "", fmt.Errorf("%w: %s", ErrInvalidRole, value)
}
