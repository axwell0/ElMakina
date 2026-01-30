package state

import "ElMakina/backend/models"

// Player represents a single player in the game.
type Player struct {
	ID    int    // Unique player identifier (index in GameState.Players)
	Name  string // Display name
	Hand  []Card // Role cards currently held
	Coins int    // Current coin count
}

// Card represents a role card in a player's hand or the deck.
type Card struct {
	ID   int         // Unique card identifier
	Role models.Role // Role type on this card
}
