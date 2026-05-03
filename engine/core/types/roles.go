package types

import "fmt"

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

// ParseRole takes in a string and resolves it to a role
func ParseRole(value string) (Role, error) {
	role := Role(value)
	for _, validRole := range Roles {
		if role == validRole {
			return role, nil
		}
	}
	return "", fmt.Errorf("invalid role: %s", value)
}
