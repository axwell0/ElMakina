// Type Generation Script for ElMakina
// Generates TypeScript types from Go structs for frontend-backend contract alignment

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ElMakina/backend/models"
	"ElMakina/backend/server/ws"
	"github.com/skia-dev/go2ts"
)

func main() {
	generator := go2ts.New()

	// ============================================================================
	// CORE ENUMS (from backend/models)
	// ============================================================================

	// ActionID - All possible player actions
	generator.Add(models.Business)
	generator.Add(models.BlockForeignAid)
	generator.Add(models.Income)
	generator.Add(models.ForeignAid)
	generator.Add(models.Coup)
	generator.Add(models.Tax)
	generator.Add(models.TaxBusinessWoman)
	generator.Add(models.Investigate)
	generator.Add(models.BlockPolice)
	generator.Add(models.Accuse)
	generator.Add(models.BlockTerrorist)
	generator.Add(models.Assassinate)
	generator.Add(models.Steal)
	generator.Add(models.BlockSteal)
	generator.Add(models.Exchange)
	generator.Add(models.Escape)
	generator.Add(models.Pass)

	// Role - Character roles
	generator.Add(models.Businesswoman)
	generator.Add(models.TaxCollector)
	generator.Add(models.Policewoman)
	generator.Add(models.Colonel)
	generator.Add(models.Terrorist)
	generator.Add(models.Thief)
	generator.Add(models.Politician)

	// ActionKind - Main vs Counter actions
	generator.Add(models.MainAction)
	generator.Add(models.CounterAction)

	// ============================================================================
	// PAYLOAD TYPES (from backend/models)
	// ============================================================================

	generator.Add(models.TargetPayload{})
	generator.Add(models.AccusePayload{})
	generator.Add(models.NoPayload{})
	generator.Add(models.PlayerAction{})

	// ============================================================================
	// WEBSOCKET MESSAGE PAYLOADS (from backend/server/ws)
	// ============================================================================

	// Lobby payloads
	generator.Add(ws.HelloPayload{})
	generator.Add(ws.HelloAckPayload{})
	generator.Add(ws.HelloErrorPayload{})
	generator.Add(ws.LobbyJoinPayload{})
	generator.Add(ws.LobbyStartPayload{})
	generator.Add(ws.LobbyCreatedPayload{})
	generator.Add(ws.LobbyListPayload{})
	generator.Add(ws.LobbyStatePayload{})
	generator.Add(ws.LobbySummaryPayload{})
	generator.Add(ws.LobbyStartedPayload{})
	generator.Add(ws.ChatMessagePayload{})

	// Game action payloads
	generator.Add(ws.ActionPayload{})
	generator.Add(ws.ChallengePayload{})
	generator.Add(ws.RequestActionPayload{})
	generator.Add(ws.RequestStepPayload{})
	generator.Add(ws.ChallengeWindowPayload{})
	generator.Add(ws.CounterWindowPayload{})

	// Game state payloads
	generator.Add(ws.GameLogPayload{})
	generator.Add(ws.GameOverPayload{})
	generator.Add(ws.PlayerStatePayload{})
	generator.Add(ws.GameStatePayload{})
	generator.Add(ws.GameConfigPayload{})
	generator.Add(ws.PromptClosedPayload{})
	generator.Add(ws.InvestigateResultPayload{})
	generator.Add(ws.HandStatePayload{})
	generator.Add(ws.PlayerEliminatedPayload{})
	generator.Add(ws.TurnTimerPayload{})

	// Pause/Vote payloads
	generator.Add(ws.GamePausedPayload{})
	generator.Add(ws.GameResumedPayload{})
	generator.Add(ws.KickVotePayload{})
	generator.Add(ws.KickVoteUpdatePayload{})
	generator.Add(ws.PlayerKickedPayload{})

	// Card discard payload
	generator.Add(ws.CardDiscardedPayload{})

	// ============================================================================
	// RENDER OUTPUT
	// ============================================================================

	var buf bytes.Buffer
	if err := generator.Render(&buf); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating types: %v\n", err)
		os.Exit(1)
	}

	// Add header comment
	header := `// Generated TypeScript types from Go structs
// DO NOT EDIT MANUALLY - Run 'npm run generate:types' to regenerate
// 
// Source: backend/models/*.go, backend/server/ws/messages.go
// Tool: github.com/skia-dev/go2ts
// 
// These types ensure frontend-backend contract alignment

`

	// Post-process to fix any issues
	processed := postProcessTypes(buf.String())

	// Determine output path
	rootDir, err := findProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding project root: %v\n", err)
		os.Exit(1)
	}

	outputPath := filepath.Join(rootDir, "web", "src", "types", "generated.ts")

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Write file
	fullContent := header + processed
	if err := os.WriteFile(outputPath, []byte(fullContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Generated TypeScript types: %s\n", outputPath)
}

// postProcessTypes applies any necessary transformations to the generated output
func postProcessTypes(input string) string {
	// Fix any known issues with go2ts output
	output := input

	// Ensure consistent naming (go2ts should handle this, but let's be safe)
	output = strings.ReplaceAll(output, "__", "_")

	return output
}

// findProjectRoot finds the project root directory (contains both backend/ and web/)
func findProjectRoot() (string, error) {
	// Start from current directory and walk up
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		// Check if this directory has both backend/ and web/ subdirectories
		backendDir := filepath.Join(dir, "backend")
		webDir := filepath.Join(dir, "web")

		if _, err := os.Stat(backendDir); err == nil {
			if _, err := os.Stat(webDir); err == nil {
				return dir, nil
			}
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding project
			return "", fmt.Errorf("could not find project root (looking for backend/ and web/ directories)")
		}
		dir = parent
	}
}
