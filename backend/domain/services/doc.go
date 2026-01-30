// Package services provides example usage of the PlayerService.
//
// Example: Register a new player
//
//	player, err := playerService.RegisterPlayer(ctx, "Alice")
//	if err != nil {
//	    log.Fatalf("Failed to register player: %v", err)
//	}
//	fmt.Printf("Registered: %s (ID: %s, Token: %s)\n",
//	    player.Nick(), player.ID(), player.Token())
//
// Example: Reconnect using token
//
//	player, err := playerService.ReconnectPlayer(ctx, token)
//	if err != nil {
//	    if errors.Is(err, entities.ErrInvalidToken) {
//	        // Handle invalid token
//	    }
//	    log.Fatalf("Reconnect failed: %v", err)
//	}
//
// Example: Unregister (fails if in lobby)
//
//	if err := playerService.UnregisterPlayer(ctx, playerID); err != nil {
//	    if errors.Is(err, entities.ErrPlayerInLobby) {
//	        // Cannot unregister - player is in a lobby
//	    }
//	}
//
// Example: Prune inactive players (run as scheduled job)
//
//	removedIDs, err := playerService.PruneInactivePlayers(ctx, 24*time.Hour)
//	if err != nil {
//	    log.Fatalf("Failed to prune: %v", err)
//	}
//	fmt.Printf("Removed %d inactive players\n", len(removedIDs))
package services
