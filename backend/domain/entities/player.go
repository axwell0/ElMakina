package entities

import (
	"fmt"
	"time"
)

// PlayerID is a type-safe identifier for players.
type PlayerID string

func (id PlayerID) String() string { return string(id) }

// Player represents a human participant in the game.
// This is a pure domain entity with no persistence concerns.
type Player struct {
	id        PlayerID
	nick      string
	token     string // Secret reconnection token
	avatar    string
	createdAt time.Time
	updatedAt time.Time
}

// NewPlayer creates a new player with validation.
func NewPlayer(id PlayerID, nick, token string) (*Player, error) {
	if id == "" {
		return nil, fmt.Errorf("player id is required: %w", ErrInvalidEntity)
	}
	if nick == "" {
		return nil, fmt.Errorf("player nick is required: %w", ErrInvalidEntity)
	}
	if token == "" {
		return nil, fmt.Errorf("player token is required: %w", ErrInvalidEntity)
	}
	if len(nick) > 32 {
		return nil, fmt.Errorf("player nick too long (max 32): %w", ErrInvalidEntity)
	}

	now := time.Now().UTC()
	return &Player{
		id:        id,
		nick:      nick,
		token:     token,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// ReconstitutePlayer reconstructs a player from persistence (no validation).
// Use only within repository mappers.
func ReconstitutePlayer(id PlayerID, nick, token, avatar string, createdAt, updatedAt time.Time) *Player {
	return &Player{
		id:        id,
		nick:      nick,
		token:     token,
		avatar:    avatar,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

// ID returns the player's unique identifier.
func (p *Player) ID() PlayerID { return p.id }

// Nick returns the player's display name.
func (p *Player) Nick() string { return p.nick }

// Token returns the player's secret reconnection token.
func (p *Player) Token() string { return p.token }

// Avatar returns the player's avatar identifier.
func (p *Player) Avatar() string { return p.avatar }

// CreatedAt returns the creation timestamp.
func (p *Player) CreatedAt() time.Time { return p.createdAt }

// UpdatedAt returns the last update timestamp.
func (p *Player) UpdatedAt() time.Time { return p.updatedAt }

// SetAvatar updates the player's avatar.
func (p *Player) SetAvatar(avatar string) {
	p.avatar = avatar
	p.updatedAt = time.Now().UTC()
}

// CanAuthenticate verifies if the provided token matches.
func (p *Player) CanAuthenticate(token string) bool {
	return p.token == token
}
