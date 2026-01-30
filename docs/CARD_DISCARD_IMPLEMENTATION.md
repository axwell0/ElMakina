# Card Discard Animation Implementation Plan

## Overview
Add visual card discard animations with popup showing "X discarded Y for Z reason" and player elimination fizzle-out effect.

## Architecture

**Go-to-TypeScript Generation:**
- Uses `go2ts` tool via `npm run generate:types`
- Generates `web/src/types/generated.ts` from Go structs
- Must add new payload types to `backend/scripts/generate-types/main.go`

**Queue System:**
- Multiple discards can happen in rapid succession
- Queue-based system with auto-dismiss (3s)
- One discard shown at a time, then next in queue

**Elimination Animation:**
- When last card is discarded, trigger player fizzle-out
- Player avatar pulses and grays out
- Can still spectate (not removed from UI)

---

## Implementation Commits

### Commit 1: `feat(ws): add CardDiscardedPayload message type`
**Files:**
- `backend/server/ws/messages.go` - Add payload struct
- `backend/server/ws/session_runner.go` - Add broadcast logic
- `backend/scripts/generate-types/main.go` - Add to go2ts generator
- `Makefile` - Add gen-types and check-types targets

**Backend Code:**
```go
// CardDiscardedPayload notifies all players when a card is discarded
type CardDiscardedPayload struct {
    PlayerIndex   int    `json:"player_index"`
    PlayerName    string `json:"player_name"`
    CardRole      string `json:"card_role"`      // e.g., "Colonel", "Terrorist"
    Reason        string `json:"reason"`         // "challenge_lost", "coup", "assassinate", "accuse", "exchange"
    Turn          int    `json:"turn"`
    IsElimination bool   `json:"is_elimination"` // true if this was their last card
}

const MsgCardDiscarded = "card_discarded"
```

**Broadcast Logic (in session_runner.go):**
After processing turn result with discard events, broadcast structured payload instead of just text log.

**Makefile Addition:**
```makefile
gen-types:
	cd backend && go run scripts/generate-types/main.go

check-types:
	cd backend && go run scripts/generate-types/main.go
	git diff --exit-code web/src/types/generated.ts

gen-all: gen-schema gen-types docs-generate
```

---

### Commit 2: `feat(state): add discard queue management`
**Files:**
- `web/src/state/types.ts` - Add CardDiscardEvent type and update GameState
- `web/src/state/slices/game/index.ts` - Add CARD_DISCARDED and DISMISS_DISCARD actions
- `web/src/state/slices/index.ts` - Add card_discarded WebSocket handler
- `web/src/state/slices/validation.ts` - Add validator registration

**TypeScript Types:**
```typescript
export interface CardDiscardEvent {
    playerIndex: number;
    playerName: string;
    cardRole: string;
    reason: string;
    turn: number;
    isElimination: boolean;
    timestamp: number;
}

export interface GameState {
    // ... existing fields
    discardQueue: CardDiscardEvent[];
    currentDiscard: CardDiscardEvent | null;
    eliminatingPlayer: number | null;
}
```

**Action Handlers:**
- CARD_DISCARDED: Add to queue, set current if empty
- DISMISS_DISCARD: Remove current, show next in queue, update eliminatingPlayer if was elimination

---

### Commit 3: `feat(ui): add CardDiscardedModal component`
**Files:**
- `web/src/components/CardDiscardedModal.tsx` - New modal component
- `web/src/components/GameView.tsx` - Add modal to render tree

**Component Features:**
- Reuses RevealModal CSS/animations (card flip)
- Shows: Player name, card role, reason, elimination flag
- Auto-dismiss: 3 seconds
- Click to dismiss early
- Progress bar animation
- Queue indicator ("Showing 1 of 3")

**Reason Labels:**
```typescript
const REASON_LABELS: Record<string, string> = {
    'challenge_lost': 'Lost a challenge',
    'coup': 'Target of Coup',
    'assassinate': 'Assassination victim',
    'accuse': 'Accused correctly',
    'exchange': 'Exchanged away',
};
```

---

### Commit 4: `feat(ui): add player elimination animation`
**Files:**
- `web/src/components/game/PlayerRing.tsx` - Add elimination animation class
- `web/src/app/globals.css` - Add elimination keyframes

**Animation:**
```css
@keyframes elimination-pulse {
    0%, 100% { transform: scale(1); filter: grayscale(0%); }
    50% { transform: scale(1.1); filter: grayscale(50%); }
}

.animate-elimination-pulse {
    animation: elimination-pulse 0.5s ease-in-out 3;
}
```

**PlayerRing Integration:**
- Check if player is being eliminated
- Apply animation class during elimination
- Transition to grayscale/50% opacity after animation

---

### Commit 5: `feat(validation): add card_discarded payload validation`
**Files:**
- `web/src/validation/payloads/game.ts` - Add validateCardDiscardedPayload
- `web/src/validation/types.ts` - Add to PayloadTypeMap
- `web/src/state/slices/validation.ts` - Register validator

**Validator:**
```typescript
export const validateCardDiscardedPayload = createPayloadValidator<PayloadTypeMap["card_discarded"]>([
    validateRequired('player_index'),
    validateRequired('player_name'),
    validateRequired('card_role'),
    validateRequired('reason'),
    validateRequired('turn'),
    validateRequired('is_elimination'),
]);
```

---

### Commit 6: `chore(types): regenerate TypeScript types from Go`
**Command:**
```bash
npm run generate:types
```

**Result:**
- Updates `web/src/types/generated.ts` with new CardDiscardedPayload type
- Auto-generated, no manual edits

---

### Commit 7: `feat(ws): determine discard reason from turn context`
**File:**
- `backend/server/ws/session_runner.go` - Helper function to determine reason

**Logic:**
```go
func determineDiscardReason(result *engine.TurnResult, playerIndex int) string {
    // Check challenge results
    for _, cr := range result.ChallengeResults {
        if cr.LostPlayerIndex == playerIndex {
            return "challenge_lost"
        }
    }
    
    // Check main action
    if result.Main != nil {
        switch result.Main.ID {
        case models.Coup:
            if result.Main.TargetIndex == playerIndex {
                return "coup"
            }
        case models.Assassinate:
            if result.Main.TargetIndex == playerIndex {
                return "assassinate"
            }
        case models.Accuse:
            // Check if target lost card
            return "accuse"
        case models.Exchange:
            return "exchange"
        }
    }
    
    return "unknown"
}
```

---

## File Changes Summary

### Backend (Go)
1. `backend/server/ws/messages.go` - Add CardDiscardedPayload
2. `backend/server/ws/session_runner.go` - Broadcast logic + reason detection
3. `backend/scripts/generate-types/main.go` - Add to go2ts
4. `Makefile` - Add gen-types and check-types targets

### Frontend (TypeScript/React)
5. `web/src/types/generated.ts` - Auto-generated
6. `web/src/state/types.ts` - Add CardDiscardEvent type
7. `web/src/state/slices/game/index.ts` - Action handlers
8. `web/src/state/slices/index.ts` - WebSocket handler
9. `web/src/state/slices/validation.ts` - Validator registration
10. `web/src/validation/payloads/game.ts` - Validator function
11. `web/src/components/CardDiscardedModal.tsx` - New component
12. `web/src/components/GameView.tsx` - Add modal to render tree
13. `web/src/components/game/PlayerRing.tsx` - Elimination animation
14. `web/src/app/globals.css` - Keyframe animations

---

## Testing Checklist

- [ ] Discard from challenge loss shows popup with correct card
- [ ] Discard from coup shows popup
- [ ] Discard from assassinate shows popup
- [ ] Discard from accuse shows popup
- [ ] Discard from exchange shows popup
- [ ] Last card discard triggers elimination animation
- [ ] Multiple discards queue correctly (show 1, then 2, then 3)
- [ ] Auto-dismiss works (3s)
- [ ] Click to dismiss works
- [ ] Progress bar animates correctly
- [ ] Eliminated player grays out and stays visible (spectator mode)
- [ ] Reduced motion respected
- [ ] Types regenerate correctly
- [ ] Validation catches malformed payloads

---

## Worktree Setup

```bash
# Create worktree
git worktree add -b feat/card-discard-animation ../elmakina-card-discard

# Switch to worktree
cd ../elmakina-card-discard

# Verify branch
git branch --show-current  # Should show: feat/card-discard-animation

# Make changes...
# Then rebase onto main when done
git checkout main
git rebase feat/card-discard-animation
```

---

**Document End**
