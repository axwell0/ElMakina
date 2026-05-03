package state

import (
	models "github.com/your-org/elmakina/engine/core/types"
)

type Player struct {
	ID    int
	Name  string
	Hand  []Card
	Coins int
}

type Card struct {
	ID   int
	Role models.Role
}
