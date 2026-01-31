/**
 * Game slice tests
 *
 * Comprehensive tests for game state management including match state,
 * player tracking, hand management, prompts, targeting, and game flow.
 */

import { describe, it, expect } from "vitest";
import {
  gameReducer,
  gameActions,
  initialGameSliceState,
  type GameSliceState,
  type GameAction,
} from "@/state/slices/game";
import type { GameIdentity, Prompt, CardDiscardEvent } from "@/state/types";

describe("Game Slice", () => {
  describe("Initial State", () => {
    it("should have correct initial values", () => {
      expect(initialGameSliceState.currentMatch).toBeNull();
      expect(initialGameSliceState.identity).toBeNull();
      expect(initialGameSliceState.players).toEqual([]);
      expect(initialGameSliceState.roles).toEqual([]);
      expect(initialGameSliceState.hand).toEqual([]);
      expect(initialGameSliceState.activePlayerIndex).toBeNull();
      expect(initialGameSliceState.pendingPrompt).toBeNull();
      expect(initialGameSliceState.promptClosedReason).toBeNull();
      expect(initialGameSliceState.targeting).toBeNull();
      expect(initialGameSliceState.turnTimer).toBeNull();
      expect(initialGameSliceState.pause).toEqual({ status: "inactive" });
      expect(initialGameSliceState.gameOver).toBeNull();
      expect(initialGameSliceState.discardQueue).toEqual([]);
      expect(initialGameSliceState.currentDiscard).toBeNull();
      expect(initialGameSliceState.eliminatingPlayer).toBeNull();
    });
  });

  describe("LOBBY_STARTED action", () => {
    it("should initialize game from lobby with all players", () => {
      const payload = {
        lobby_id: "lobby-123",
        match_id: "match-456",
        player_index: 0,
        player_count: 4,
        player_names: ["Alice", "Bob", "Charlie", "David"],
        player_avatars: ["avatar1", "avatar2", "avatar3", "avatar4"],
      };
      const action = gameActions.lobbyStarted(payload, "player-abc", null);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.identity).toEqual({
        playerId: "player-abc",
        playerIndex: 0,
        playerNames: ["Alice", "Bob", "Charlie", "David"],
      });
      expect(newState.players).toHaveLength(4);
      expect(newState.players[0]).toEqual({
        index: 0,
        name: "Alice",
        alive: true,
        coins: null,
        cardCount: null,
        avatar: "avatar1",
      });
      expect(newState.currentMatch).toEqual({
        matchId: "match-456",
        lobbyId: "lobby-123",
        playerNames: ["Alice", "Bob", "Charlie", "David"],
        participantIds: [],
      });
      expect(newState.activePlayerIndex).toBe(0);
      expect(newState.hand).toEqual([]);
      expect(newState.roles).toEqual([]);
    });

    it("should use lobby_id as match_id when match_id not provided", () => {
      const payload = {
        lobby_id: "lobby-123",
        player_index: 1,
        player_count: 2,
        player_names: ["Alice", "Bob"],
      };
      const action = gameActions.lobbyStarted(payload, "player-xyz", null);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.currentMatch?.matchId).toBe("lobby-123");
    });

    it("should handle missing player_avatars", () => {
      const payload = {
        lobby_id: "lobby-123",
        player_index: 0,
        player_count: 2,
        player_names: ["Alice", "Bob"],
      };
      const action = gameActions.lobbyStarted(payload, "player-1", null);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.players[0].avatar).toBe("");
      expect(newState.players[1].avatar).toBe("");
    });

    it("should use index_mapping for participantIds when available", () => {
      const payload = {
        lobby_id: "lobby-123",
        player_index: 0,
        player_count: 2,
        player_names: ["Alice", "Bob"],
        index_mapping: { "id-1": 0, "id-2": 1 },
      };
      const action = gameActions.lobbyStarted(payload, "player-1", null);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.currentMatch?.participantIds).toEqual(["id-1", "id-2"]);
    });

    it("should use currentLobby.playerIds when index_mapping not available", () => {
      const payload = {
        lobby_id: "lobby-123",
        player_index: 0,
        player_count: 2,
        player_names: ["Alice", "Bob"],
      };
      const currentLobby = {
        lobbyId: "lobby-123",
        playerNicks: ["Alice", "Bob"],
        playerCount: 2,
        playerIds: ["pid-1", "pid-2"],
      };
      const action = gameActions.lobbyStarted(payload, "player-1", currentLobby);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.currentMatch?.participantIds).toEqual(["pid-1", "pid-2"]);
    });

    it("should handle null currentPlayerId", () => {
      const payload = {
        lobby_id: "lobby-123",
        player_index: 0,
        player_count: 2,
        player_names: ["Alice", "Bob"],
      };
      const action = gameActions.lobbyStarted(payload, null, null);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.identity?.playerId).toBe("unknown");
    });

    it("should reset game state fields on lobby start", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        gameOver: { winnerIndex: 0, winnerName: "Alice" },
        pendingPrompt: { kind: "action", requestId: "req-1", allowedActions: [] },
        targeting: { active: true, actionId: "action-1", requestId: "req-1", selectedTarget: null },
      };
      const payload = {
        lobby_id: "lobby-123",
        player_index: 0,
        player_count: 2,
        player_names: ["Alice", "Bob"],
      };
      const action = gameActions.lobbyStarted(payload, "player-1", null);
      const newState = gameReducer(state, action);

      expect(newState.gameOver).toBeNull();
      expect(newState.pendingPrompt).toBeNull();
      expect(newState.targeting).toBeNull();
    });

    it("should create currentLobby from payload when currentLobby is null", () => {
      const payload = {
        lobby_id: "lobby-123",
        player_index: 0,
        player_count: 3,
        player_names: ["Alice", "Bob", "Charlie"],
      };
      const action = gameActions.lobbyStarted(payload, "player-1", null);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.currentMatch?.participantIds).toEqual([]);
    });
  });

  describe("GAME_CONFIG action", () => {
    it("should set roles from payload", () => {
      const payload = { roles: ["Duke", "Assassin", "Captain", "Contessa", "Ambassador"] };
      const action = gameActions.gameConfig(payload);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.roles).toEqual(["Duke", "Assassin", "Captain", "Contessa", "Ambassador"]);
    });

    it("should handle undefined payload", () => {
      const action = gameActions.gameConfig(undefined);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.roles).toEqual([]);
    });

    it("should handle payload without roles", () => {
      const action = gameActions.gameConfig({});
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.roles).toEqual([]);
    });

    it("should overwrite existing roles", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        roles: ["OldRole1", "OldRole2"],
      };
      const payload = { roles: ["Duke", "Assassin"] };
      const action = gameActions.gameConfig(payload);
      const newState = gameReducer(state, action);

      expect(newState.roles).toEqual(["Duke", "Assassin"]);
    });

    it("should preserve other state when setting roles", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        players: [{ index: 0, name: "Alice", alive: true, coins: 2, cardCount: 2, avatar: "" }],
      };
      const payload = { roles: ["Duke"] };
      const action = gameActions.gameConfig(payload);
      const newState = gameReducer(state, action);

      expect(newState.players).toEqual(state.players);
      expect(newState.roles).toEqual(["Duke"]);
    });
  });

  describe("GAME_STATE action", () => {
    it("should update player states with coins and card counts", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        players: [
          { index: 0, name: "Alice", alive: true, coins: null, cardCount: null, avatar: "" },
          { index: 1, name: "Bob", alive: true, coins: null, cardCount: null, avatar: "" },
        ],
      };
      const payload = {
        players: [
          { index: 0, name: "Alice", alive: true, coins: 3, card_count: 2 },
          { index: 1, name: "Bob", alive: true, coins: 2, card_count: 2 },
        ],
        active_player_index: 1,
      };
      const action = gameActions.gameState(payload, null);
      const newState = gameReducer(state, action);

      expect(newState.players[0].coins).toBe(3);
      expect(newState.players[0].cardCount).toBe(2);
      expect(newState.players[1].coins).toBe(2);
      expect(newState.activePlayerIndex).toBe(1);
    });

    it("should track prevCoins for animation", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        players: [
          { index: 0, name: "Alice", alive: true, coins: 2, cardCount: 2, avatar: "" },
        ],
      };
      const payload = {
        players: [
          { index: 0, name: "Alice", alive: true, coins: 5, card_count: 2 },
        ],
        active_player_index: 0,
      };
      const action = gameActions.gameState(payload, null);
      const newState = gameReducer(state, action);

      expect(newState.players[0].coins).toBe(5);
      expect(newState.players[0].prevCoins).toBe(2);
    });

    it("should use current coins as prevCoins when no previous state", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        players: [
          { index: 0, name: "Alice", alive: true, coins: null, cardCount: null, avatar: "" },
        ],
      };
      const payload = {
        players: [
          { index: 0, name: "Alice", alive: true, coins: 3, card_count: 2 },
        ],
        active_player_index: 0,
      };
      const action = gameActions.gameState(payload, null);
      const newState = gameReducer(state, action);

      expect(newState.players[0].prevCoins).toBe(3);
    });

    it("should preserve avatars from previous state", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        players: [
          { index: 0, name: "Alice", alive: true, coins: 2, cardCount: 2, avatar: "alice-avatar" },
        ],
      };
      const payload = {
        players: [
          { index: 0, name: "Alice", alive: true, coins: 3, card_count: 2 },
        ],
        active_player_index: 0,
      };
      const action = gameActions.gameState(payload, null);
      const newState = gameReducer(state, action);

      expect(newState.players[0].avatar).toBe("alice-avatar");
    });

    it("should use avatar from payload when available", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        players: [
          { index: 0, name: "Alice", alive: true, coins: 2, cardCount: 2, avatar: "old-avatar" },
        ],
      };
      const payload = {
        players: [
          { index: 0, name: "Alice", alive: true, coins: 3, card_count: 2, avatar: "new-avatar" },
        ],
        active_player_index: 0,
      };
      const action = gameActions.gameState(payload, null);
      const newState = gameReducer(state, action);

      expect(newState.players[0].avatar).toBe("new-avatar");
    });

    it("should handle undefined payload", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        players: [
          { index: 0, name: "Alice", alive: true, coins: 2, cardCount: 2, avatar: "" },
        ],
        activePlayerIndex: 0,
      };
      const action = gameActions.gameState(undefined, null);
      const newState = gameReducer(state, action);

      expect(newState.players).toEqual([]);
      expect(newState.activePlayerIndex).toBe(0);
    });

    it("should handle empty players array", () => {
      const payload = {
        players: [],
        active_player_index: 0,
      };
      const action = gameActions.gameState(payload, null);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.players).toEqual([]);
    });
  });

  describe("REQUEST_ACTION action", () => {
    it("should set action prompt for current player", () => {
      const identity: GameIdentity = { playerId: "p1", playerIndex: 0, playerNames: ["Alice", "Bob"] };
      const payload = {
        actor_index: 0,
        allowed_actions: ["income", "foreign_aid", "coup", "duke"],
      };
      const action = gameActions.requestAction(payload, "req-123", identity);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.pendingPrompt).toEqual({
        kind: "action",
        requestId: "req-123",
        allowedActions: ["income", "foreign_aid", "coup", "duke"],
      });
      expect(newState.activePlayerIndex).toBe(0);
      expect(newState.promptClosedReason).toBeNull();
    });

    it("should not set prompt when action is for different player", () => {
      const identity: GameIdentity = { playerId: "p1", playerIndex: 1, playerNames: ["Alice", "Bob"] };
      const payload = {
        actor_index: 0,
        allowed_actions: ["income", "foreign_aid"],
      };
      const action = gameActions.requestAction(payload, "req-123", identity);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.pendingPrompt).toBeNull();
      expect(newState.activePlayerIndex).toBe(0);
    });

    it("should update activePlayerIndex even when not current player", () => {
      const identity: GameIdentity = { playerId: "p1", playerIndex: 2, playerNames: ["Alice", "Bob", "Charlie"] };
      const payload = {
        actor_index: 1,
        allowed_actions: ["income"],
      };
      const action = gameActions.requestAction(payload, "req-123", identity);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.activePlayerIndex).toBe(1);
    });

    it("should handle null currentIdentity", () => {
      const payload = {
        actor_index: 0,
        allowed_actions: ["income"],
      };
      const action = gameActions.requestAction(payload, "req-123", null);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.pendingPrompt).toBeNull();
      expect(newState.activePlayerIndex).toBe(0);
    });

    it("should handle missing allowed_actions", () => {
      const identity: GameIdentity = { playerId: "p1", playerIndex: 0, playerNames: ["Alice"] };
      const payload = {
        actor_index: 0,
      };
      const action = gameActions.requestAction(payload, "req-123", identity);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.pendingPrompt?.kind === "action" ? newState.pendingPrompt.allowedActions : undefined).toEqual([]);
    });
  });

  describe("CHALLENGE_WINDOW action", () => {
    it("should set challenge prompt with all fields", () => {
      const payload = {
        action_id: "action-123",
        actor_index: 1,
        claimed_role: "Duke",
        eligible: true,
        kind: "main",
        prompt: "Challenge Duke?",
        target_index: 2,
        timeout_ms: 10000,
      };
      const action = gameActions.challengeWindow(payload, "req-456");
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.pendingPrompt).toEqual({
        kind: "challenge",
        requestId: "req-456",
        actorIndex: 1,
        actionId: "action-123",
        claimedRole: "Duke",
        challengeKind: "main",
        targetIndex: 2,
        eligible: true,
        timeoutMs: 10000,
      });
    });

    it("should handle counter challenge kind", () => {
      const payload = {
        action_id: "action-123",
        actor_index: 0,
        claimed_role: "Captain",
        eligible: false,
        kind: "counter",
      };
      const action = gameActions.challengeWindow(payload, "req-789");
      const newState = gameReducer(initialGameSliceState, action);

      const pp = newState.pendingPrompt;
      expect(pp?.kind === "challenge" ? pp.challengeKind : undefined).toBe("counter");
      expect(pp?.kind === "challenge" ? pp.eligible : undefined).toBe(false);
    });

    it("should handle unknown challenge kind", () => {
      const payload = {
        action_id: "action-123",
        actor_index: 0,
        claimed_role: "Ambassador",
        eligible: true,
        kind: "unknown",
      };
      const action = gameActions.challengeWindow(payload, "req-000");
      const newState = gameReducer(initialGameSliceState, action);

      const pp = newState.pendingPrompt;
      expect(pp?.kind === "challenge" ? pp.challengeKind : undefined).toBeUndefined();
    });

    it("should handle missing target_index", () => {
      const payload = {
        action_id: "action-123",
        actor_index: 0,
        claimed_role: "Duke",
        eligible: true,
        kind: "main",
      };
      const action = gameActions.challengeWindow(payload, "req-111");
      const newState = gameReducer(initialGameSliceState, action);

      const pp = newState.pendingPrompt;
      expect(pp?.kind === "challenge" ? pp.targetIndex : undefined).toBeUndefined();
    });

    it("should clear promptClosedReason", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        promptClosedReason: "timeout",
      };
      const payload = {
        action_id: "action-123",
        actor_index: 0,
        claimed_role: "Duke",
        eligible: true,
        kind: "main",
      };
      const action = gameActions.challengeWindow(payload, "req-222");
      const newState = gameReducer(state, action);

      expect(newState.promptClosedReason).toBeNull();
    });
  });

  describe("COUNTER_WINDOW action", () => {
    it("should set counter prompt with all fields", () => {
      const payload = {
        action_id: "action-123",
        actor_index: 1,
        allowed_actions: ["block_foreign_aid", "pass"],
        eligible: true,
        prompt: "Block Foreign Aid?",
        target_index: 0,
        timeout_ms: 8000,
      };
      const action = gameActions.counterWindow(payload, "req-333");
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.pendingPrompt).toEqual({
        kind: "counter",
        requestId: "req-333",
        actorIndex: 1,
        allowedActions: ["block_foreign_aid", "pass"],
        actionId: "action-123",
        targetIndex: 0,
        eligible: true,
        timeoutMs: 8000,
      });
    });

    it("should use activePlayerIndex when actor_index is not a number", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        activePlayerIndex: 2,
      };
      const payload = {
        action_id: "action-123",
        actor_index: undefined as unknown as number,
        allowed_actions: ["pass"],
        eligible: true,
      };
      const action = gameActions.counterWindow(payload, "req-444");
      const newState = gameReducer(state, action);

      const pp = newState.pendingPrompt;
      expect(pp?.kind === "counter" ? pp.actorIndex : undefined).toBe(2);
    });

    it("should use -1 when actor_index missing and no activePlayerIndex", () => {
      const payload = {
        action_id: "action-123",
        actor_index: undefined as unknown as number,
        allowed_actions: ["pass"],
        eligible: true,
      };
      const action = gameActions.counterWindow(payload, "req-555");
      const newState = gameReducer(initialGameSliceState, action);

      const pp = newState.pendingPrompt;
      expect(pp?.kind === "counter" ? pp.actorIndex : undefined).toBe(-1);
    });

    it("should handle missing allowed_actions", () => {
      const payload = {
        action_id: "action-123",
        actor_index: 0,
        eligible: false,
      };
      const action = gameActions.counterWindow(payload, "req-666");
      const newState = gameReducer(initialGameSliceState, action);

      const pp = newState.pendingPrompt;
      expect(pp?.kind === "counter" ? pp.allowedActions : undefined).toEqual([]);
    });
  });

  describe("REQUEST_STEP action", () => {
    it("should set step prompt", () => {
      const payload = {
        prompt: "Choose a card to keep",
        step: {
          context: "exchange",
          count: 1,
          options: ["Duke", "Assassin", "Ambassador"],
        },
      };
      const action = gameActions.requestStep(payload, "req-777");
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.pendingPrompt).toEqual({
        kind: "step",
        requestId: "req-777",
        context: "exchange",
        count: 1,
        options: ["Duke", "Assassin", "Ambassador"],
      });
    });

    it("should clear promptClosedReason", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        promptClosedReason: "completed",
      };
      const payload = {
        step: {
          context: "discard",
          count: 1,
          options: ["Contessa", "Captain"],
        },
      };
      const action = gameActions.requestStep(payload, "req-888");
      const newState = gameReducer(state, action);

      expect(newState.promptClosedReason).toBeNull();
    });
  });

  describe("HAND_STATE action", () => {
    it("should update hand for current player", () => {
      const identity: GameIdentity = { playerId: "p1", playerIndex: 0, playerNames: ["Alice", "Bob"] };
      const payload = {
        hand: ["Duke", "Assassin"],
        player_index: 0,
      };
      const action = gameActions.handState(payload, identity);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.hand).toHaveLength(2);
      expect(newState.hand[0]).toEqual({ id: "0-0-Duke", role: "Duke" });
      expect(newState.hand[1]).toEqual({ id: "0-1-Assassin", role: "Assassin" });
    });

    it("should not update hand for different player", () => {
      const identity: GameIdentity = { playerId: "p1", playerIndex: 0, playerNames: ["Alice", "Bob"] };
      const payload = {
        hand: ["Captain", "Contessa"],
        player_index: 1,
      };
      const action = gameActions.handState(payload, identity);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.hand).toEqual([]);
    });

    it("should handle empty hand", () => {
      const identity: GameIdentity = { playerId: "p1", playerIndex: 0, playerNames: ["Alice"] };
      const payload = {
        hand: [],
        player_index: 0,
      };
      const action = gameActions.handState(payload, identity);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.hand).toEqual([]);
    });

    it("should handle null currentIdentity", () => {
      const payload = {
        hand: ["Duke"],
        player_index: 0,
      };
      const action = gameActions.handState(payload, null);
      const newState = gameReducer(initialGameSliceState, action);

      // When currentIdentity is null, hand should be updated (no player_index check)
      expect(newState.hand).toEqual([{ id: "0-0-Duke", role: "Duke" }]);
    });

    it("should generate unique card IDs", () => {
      const identity: GameIdentity = { playerId: "p1", playerIndex: 2, playerNames: ["A", "B", "C"] };
      const payload = {
        hand: ["Duke", "Duke", "Assassin"],
        player_index: 2,
      };
      const action = gameActions.handState(payload, identity);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.hand[0].id).toBe("2-0-Duke");
      expect(newState.hand[1].id).toBe("2-1-Duke");
      expect(newState.hand[2].id).toBe("2-2-Assassin");
    });
  });

  describe("PROMPT_CLOSED action", () => {
    it("should clear prompt when requestId matches current prompt", () => {
      const currentPrompt: Prompt = { kind: "action", requestId: "req-123", allowedActions: [] };
      const state: GameSliceState = {
        ...initialGameSliceState,
        pendingPrompt: currentPrompt,
        targeting: { active: true, actionId: "a1", requestId: "req-123", selectedTarget: null },
      };
      const payload = { reason: "completed" };
      const action = gameActions.promptClosed(payload, "req-123", currentPrompt);
      const newState = gameReducer(state, action);

      expect(newState.pendingPrompt).toBeNull();
      expect(newState.promptClosedReason).toBe("completed");
      expect(newState.targeting).toBeNull();
    });

    it("should not clear prompt when requestId does not match", () => {
      const currentPrompt: Prompt = { kind: "action", requestId: "req-123", allowedActions: [] };
      const state: GameSliceState = {
        ...initialGameSliceState,
        pendingPrompt: currentPrompt,
      };
      const payload = { reason: "timeout" };
      const action = gameActions.promptClosed(payload, "req-999", currentPrompt);
      const newState = gameReducer(state, action);

      expect(newState.pendingPrompt).toEqual(currentPrompt);
    });

    it("should not clear prompt when currentPrompt is null", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        pendingPrompt: { kind: "action", requestId: "req-123", allowedActions: [] },
      };
      const payload = { reason: "completed" };
      const action = gameActions.promptClosed(payload, "req-123", null);
      const newState = gameReducer(state, action);

      expect(newState.pendingPrompt).not.toBeNull();
    });

    it("should handle null reason", () => {
      const currentPrompt: Prompt = { kind: "action", requestId: "req-123", allowedActions: [] };
      const state: GameSliceState = {
        ...initialGameSliceState,
        pendingPrompt: currentPrompt,
      };
      const payload = { reason: null as unknown as string };
      const action = gameActions.promptClosed(payload, "req-123", currentPrompt);
      const newState = gameReducer(state, action);

      expect(newState.promptClosedReason).toBeNull();
    });
  });

  describe("TURN_TIMER action", () => {
    it("should set turn timer with start state", () => {
      const payload = {
        active_player_index: 1,
        duration_ms: 30000,
        state: "start",
        turn_number: 5,
      };
      const action = gameActions.turnTimer(payload);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.turnTimer?.activePlayerIndex).toBe(1);
      expect(newState.turnTimer?.durationMs).toBe(30000);
      expect(newState.turnTimer?.running).toBe(true);
      expect(newState.turnTimer?.paused).toBe(false);
      expect(newState.turnTimer?.key).toMatch(/^5-\d+$/);
    });

    it("should set paused state", () => {
      const payload = {
        active_player_index: 0,
        duration_ms: 15000,
        state: "paused",
        turn_number: 3,
      };
      const action = gameActions.turnTimer(payload);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.turnTimer?.running).toBe(false);
      expect(newState.turnTimer?.paused).toBe(true);
    });

    it("should handle unknown state as not running", () => {
      const payload = {
        active_player_index: 0,
        duration_ms: 30000,
        state: "unknown",
        turn_number: 1,
      };
      const action = gameActions.turnTimer(payload);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.turnTimer?.running).toBe(false);
    });

    it("should generate unique key based on turn number and timestamp", () => {
      const payload = {
        active_player_index: 0,
        duration_ms: 30000,
        state: "start",
        turn_number: 10,
      };
      const action = gameActions.turnTimer(payload);
      const newState1 = gameReducer(initialGameSliceState, action);

      // Wait a tick to ensure different timestamp
      const start = Date.now();
      while (Date.now() === start) {
        // Busy wait for millisecond to change
      }
      const newState2 = gameReducer(initialGameSliceState, action);

      expect(newState1.turnTimer?.key).not.toBe(newState2.turnTimer?.key);
      expect(newState1.turnTimer?.key).toMatch(/^10-/);
    });

    it("should handle undefined turn_number", () => {
      const payload = {
        active_player_index: 0,
        duration_ms: 30000,
        state: "start",
        turn_number: undefined as unknown as number,
      };
      const action = gameActions.turnTimer(payload);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.turnTimer?.key).toMatch(/^0-\d+$/);
    });
  });

  describe("CLEAR_PROMPT action", () => {
    it("should clear pending prompt", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        pendingPrompt: { kind: "action", requestId: "req-123", allowedActions: [] },
      };
      const action = gameActions.clearPrompt();
      const newState = gameReducer(state, action);

      expect(newState.pendingPrompt).toBeNull();
    });

    it("should handle clearing when no prompt exists", () => {
      const action = gameActions.clearPrompt();
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.pendingPrompt).toBeNull();
    });
  });

  describe("SET_TARGETING action", () => {
    it("should activate targeting mode", () => {
      const action = gameActions.setTargeting("action-123", "req-456");
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.targeting).toEqual({
        active: true,
        actionId: "action-123",
        requestId: "req-456",
        selectedTarget: null,
      });
    });
  });

  describe("SET_TARGET_SELECTED action", () => {
    it("should set selected target", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        targeting: {
          active: true,
          actionId: "action-123",
          requestId: "req-456",
          selectedTarget: null,
        },
      };
      const action = gameActions.setTargetSelected(2);
      const newState = gameReducer(state, action);

      expect(newState.targeting?.selectedTarget).toBe(2);
    });

    it("should not modify state when targeting is null", () => {
      const action = gameActions.setTargetSelected(1);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.targeting).toBeNull();
    });

    it("should preserve other targeting fields", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        targeting: {
          active: true,
          actionId: "action-123",
          requestId: "req-456",
          selectedTarget: null,
        },
      };
      const action = gameActions.setTargetSelected(3);
      const newState = gameReducer(state, action);

      expect(newState.targeting?.active).toBe(true);
      expect(newState.targeting?.actionId).toBe("action-123");
      expect(newState.targeting?.requestId).toBe("req-456");
    });
  });

  describe("CLEAR_TARGETING action", () => {
    it("should clear targeting state", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        targeting: {
          active: true,
          actionId: "action-123",
          requestId: "req-456",
          selectedTarget: 2,
        },
      };
      const action = gameActions.clearTargeting();
      const newState = gameReducer(state, action);

      expect(newState.targeting).toBeNull();
    });

    it("should handle clearing when targeting is null", () => {
      const action = gameActions.clearTargeting();
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.targeting).toBeNull();
    });
  });

  describe("GAME_OVER action", () => {
    it("should set game over state with winner", () => {
      const action = gameActions.gameOver(1, "Bob");
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.gameOver).toEqual({
        winnerIndex: 1,
        winnerName: "Bob",
      });
    });

    it("should clear pending prompt on game over", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        pendingPrompt: { kind: "action", requestId: "req-123", allowedActions: [] },
      };
      const action = gameActions.gameOver(0, "Alice");
      const newState = gameReducer(state, action);

      expect(newState.pendingPrompt).toBeNull();
    });

    it("should clear targeting on game over", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        targeting: { active: true, actionId: "a1", requestId: "r1", selectedTarget: null },
      };
      const action = gameActions.gameOver(2, "Charlie");
      const newState = gameReducer(state, action);

      expect(newState.targeting).toBeNull();
    });
  });

  describe("PLAYER_ELIMINATED action", () => {
    it("should mark player as eliminated", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        players: [
          { index: 0, name: "Alice", alive: true, coins: 5, cardCount: 2, avatar: "" },
          { index: 1, name: "Bob", alive: true, coins: 3, cardCount: 2, avatar: "" },
        ],
      };
      const payload = {
        player_index: 1,
        reason: "lost_influence",
        turn: 5,
      };
      const action = gameActions.playerEliminated(payload);
      const newState = gameReducer(state, action);

      expect(newState.players[1].alive).toBe(false);
      expect(newState.players[1].cardCount).toBe(0);
      expect(newState.players[0].alive).toBe(true);
    });

    it("should handle eliminating non-existent player index", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        players: [
          { index: 0, name: "Alice", alive: true, coins: 5, cardCount: 2, avatar: "" },
        ],
      };
      const payload = {
        player_index: 5,
        reason: "coup",
        turn: 3,
      };
      const action = gameActions.playerEliminated(payload);
      const newState = gameReducer(state, action);

      expect(newState.players).toHaveLength(1);
      expect(newState.players[0].alive).toBe(true);
    });
  });

  describe("GAME_PAUSED action", () => {
    it("should set paused state with all fields", () => {
      const payload = {
        paused_by_player_id: "player-123",
        paused_by_index: 0,
        paused_by_name: "Alice",
        deadline_ms: 60000,
        duration_ms: 300000,
        pause_reason: "Player requested pause",
        eligible_voters: [0, 1, 2],
        kick_votes: [1],
      };
      const action = gameActions.gamePaused(payload);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.pause).toEqual({
        status: "paused",
        pausedByPlayerId: "player-123",
        pausedByIndex: 0,
        pausedByName: "Alice",
        deadlineMs: 60000,
        durationMs: 300000,
        pauseReason: "Player requested pause",
        eligibleVoters: [0, 1, 2],
        kickVotes: [1],
      });
    });

    it("should handle empty kick votes", () => {
      const payload = {
        paused_by_player_id: "player-1",
        paused_by_index: 0,
        paused_by_name: "Alice",
        deadline_ms: 60000,
        duration_ms: 300000,
        pause_reason: "Pause",
        eligible_voters: [0, 1],
        kick_votes: [],
      };
      const action = gameActions.gamePaused(payload);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.pause.status).toBe("paused");
      const pause = newState.pause;
      expect(pause.status === "paused" ? pause.eligibleVoters : undefined).toEqual([0, 1]);
      expect(pause.status === "paused" ? pause.kickVotes : undefined).toEqual([]);
    });
  });

  describe("GAME_RESUMED action", () => {
    it("should set resumed state", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        pause: {
          status: "paused",
          pausedByPlayerId: "player-1",
          pausedByIndex: 0,
          pausedByName: "Alice",
          deadlineMs: 60000,
          durationMs: 300000,
          pauseReason: "Pause",
          eligibleVoters: [0, 1],
          kickVotes: [],
        },
      };
      const payload = {
        resumed_by_player_id: "player-2",
        resumed_by_index: 1,
        resumed_by_name: "Bob",
        resume_reason: "Player resumed",
      };
      const action = gameActions.gameResumed(payload);
      const newState = gameReducer(state, action);

      expect(newState.pause).toEqual({
        status: "resumed",
        resumedByPlayerId: "player-2",
        resumedByIndex: 1,
        resumedByName: "Bob",
        resumeReason: "Player resumed",
      });
    });
  });

  describe("KICK_VOTE_UPDATE action", () => {
    it("should update kick votes when paused", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        pause: {
          status: "paused",
          pausedByPlayerId: "player-1",
          pausedByIndex: 0,
          pausedByName: "Alice",
          deadlineMs: 60000,
          durationMs: 300000,
          pauseReason: "Pause",
          eligibleVoters: [0, 1, 2],
          kickVotes: [1],
        },
      };
      const payload = {
        eligible_voters: [0, 1, 2],
        kick_votes: [1, 2],
      };
      const action = gameActions.kickVoteUpdate(payload);
      const newState = gameReducer(state, action);

      const pause = newState.pause;
      expect(pause.status === "paused" ? pause.kickVotes : undefined).toEqual([1, 2]);
    });

    it("should not update when not in paused state", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        pause: { status: "inactive" },
      };
      const payload = {
        eligible_voters: [0, 1],
        kick_votes: [0],
      };
      const action = gameActions.kickVoteUpdate(payload);
      const newState = gameReducer(state, action);

      expect(newState.pause.status).toBe("inactive");
    });

    it("should not update when in resumed state", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        pause: {
          status: "resumed",
          resumedByPlayerId: "player-1",
          resumedByIndex: 0,
          resumedByName: "Alice",
          resumeReason: "Resumed",
        },
      };
      const payload = {
        eligible_voters: [0, 1],
        kick_votes: [0],
      };
      const action = gameActions.kickVoteUpdate(payload);
      const newState = gameReducer(state, action);

      expect(newState.pause.status).toBe("resumed");
    });
  });

  describe("PLAYER_KICKED action", () => {
    it("should mark kicked player as eliminated", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        players: [
          { index: 0, name: "Alice", alive: true, coins: 5, cardCount: 2, avatar: "" },
          { index: 1, name: "Bob", alive: true, coins: 3, cardCount: 2, avatar: "" },
        ],
        pause: {
          status: "paused",
          pausedByPlayerId: "player-1",
          pausedByIndex: 0,
          pausedByName: "Alice",
          deadlineMs: 60000,
          durationMs: 300000,
          pauseReason: "Pause",
          eligibleVoters: [0, 1],
          kickVotes: [0],
        },
      };
      const payload = {
        player_index: 1,
        reason: "voted_to_kick",
      };
      const action = gameActions.playerKicked(payload);
      const newState = gameReducer(state, action);

      expect(newState.players[1].alive).toBe(false);
      expect(newState.players[1].cardCount).toBe(0);
    });

    it("should reset pause state to inactive", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        players: [
          { index: 0, name: "Alice", alive: true, coins: 5, cardCount: 2, avatar: "" },
        ],
        pause: {
          status: "paused",
          pausedByPlayerId: "player-1",
          pausedByIndex: 0,
          pausedByName: "Alice",
          deadlineMs: 60000,
          durationMs: 300000,
          pauseReason: "Pause",
          eligibleVoters: [0],
          kickVotes: [],
        },
      };
      const payload = {
        player_index: 0,
        reason: "voted_to_kick",
      };
      const action = gameActions.playerKicked(payload);
      const newState = gameReducer(state, action);

      expect(newState.pause).toEqual({ status: "inactive" });
    });
  });

  describe("CARD_DISCARDED action", () => {
    it("should add discard event to queue", () => {
      const payload: CardDiscardEvent = {
        playerIndex: 0,
        playerName: "Alice",
        cardRole: "Duke",
        reason: "lost_influence",
        turn: 5,
        isElimination: false,
        timestamp: Date.now(),
      };
      const action = gameActions.cardDiscarded(payload);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.discardQueue).toHaveLength(1);
      expect(newState.discardQueue[0]).toEqual(payload);
    });

    it("should set currentDiscard when queue was empty", () => {
      const payload: CardDiscardEvent = {
        playerIndex: 1,
        playerName: "Bob",
        cardRole: "Assassin",
        reason: "challenge_failed",
        turn: 3,
        isElimination: false,
        timestamp: 12345,
      };
      const action = gameActions.cardDiscarded(payload);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.currentDiscard).toEqual(payload);
    });

    it("should preserve currentDiscard when queue not empty", () => {
      const existingDiscard: CardDiscardEvent = {
        playerIndex: 0,
        playerName: "Alice",
        cardRole: "Duke",
        reason: "coup",
        turn: 2,
        isElimination: false,
        timestamp: 10000,
      };
      const state: GameSliceState = {
        ...initialGameSliceState,
        discardQueue: [existingDiscard],
        currentDiscard: existingDiscard,
      };
      const newDiscard: CardDiscardEvent = {
        playerIndex: 1,
        playerName: "Bob",
        cardRole: "Contessa",
        reason: "assassinated",
        turn: 4,
        isElimination: false,
        timestamp: 20000,
      };
      const action = gameActions.cardDiscarded(newDiscard);
      const newState = gameReducer(state, action);

      expect(newState.currentDiscard).toEqual(existingDiscard);
      expect(newState.discardQueue).toHaveLength(2);
    });

    it("should set eliminatingPlayer when isElimination is true", () => {
      const payload: CardDiscardEvent = {
        playerIndex: 2,
        playerName: "Charlie",
        cardRole: "Captain",
        reason: "lost_last_influence",
        turn: 8,
        isElimination: true,
        timestamp: 30000,
      };
      const action = gameActions.cardDiscarded(payload);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.eliminatingPlayer).toBe(2);
    });

    it("should preserve eliminatingPlayer when not elimination", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        eliminatingPlayer: 1,
      };
      const payload: CardDiscardEvent = {
        playerIndex: 0,
        playerName: "Alice",
        cardRole: "Ambassador",
        reason: "exchange",
        turn: 5,
        isElimination: false,
        timestamp: 40000,
      };
      const action = gameActions.cardDiscarded(payload);
      const newState = gameReducer(state, action);

      expect(newState.eliminatingPlayer).toBe(1);
    });
  });

  describe("DISMISS_DISCARD action", () => {
    it("should remove first item from queue", () => {
      const discard1: CardDiscardEvent = {
        playerIndex: 0,
        playerName: "Alice",
        cardRole: "Duke",
        reason: "coup",
        turn: 2,
        isElimination: false,
        timestamp: 10000,
      };
      const discard2: CardDiscardEvent = {
        playerIndex: 1,
        playerName: "Bob",
        cardRole: "Assassin",
        reason: "challenge_failed",
        turn: 3,
        isElimination: false,
        timestamp: 20000,
      };
      const state: GameSliceState = {
        ...initialGameSliceState,
        discardQueue: [discard1, discard2],
        currentDiscard: discard1,
      };
      const action = gameActions.dismissDiscard();
      const newState = gameReducer(state, action);

      expect(newState.discardQueue).toHaveLength(1);
      expect(newState.discardQueue[0]).toEqual(discard2);
    });

    it("should update currentDiscard to next item", () => {
      const discard1: CardDiscardEvent = {
        playerIndex: 0,
        playerName: "Alice",
        cardRole: "Duke",
        reason: "coup",
        turn: 2,
        isElimination: false,
        timestamp: 10000,
      };
      const discard2: CardDiscardEvent = {
        playerIndex: 1,
        playerName: "Bob",
        cardRole: "Contessa",
        reason: "assassinated",
        turn: 4,
        isElimination: false,
        timestamp: 20000,
      };
      const state: GameSliceState = {
        ...initialGameSliceState,
        discardQueue: [discard1, discard2],
        currentDiscard: discard1,
      };
      const action = gameActions.dismissDiscard();
      const newState = gameReducer(state, action);

      expect(newState.currentDiscard).toEqual(discard2);
    });

    it("should set currentDiscard to null when queue empty", () => {
      const discard: CardDiscardEvent = {
        playerIndex: 0,
        playerName: "Alice",
        cardRole: "Duke",
        reason: "coup",
        turn: 2,
        isElimination: false,
        timestamp: 10000,
      };
      const state: GameSliceState = {
        ...initialGameSliceState,
        discardQueue: [discard],
        currentDiscard: discard,
      };
      const action = gameActions.dismissDiscard();
      const newState = gameReducer(state, action);

      expect(newState.discardQueue).toEqual([]);
      expect(newState.currentDiscard).toBeNull();
    });

    it("should clear eliminatingPlayer if currentDiscard was elimination", () => {
      const discard: CardDiscardEvent = {
        playerIndex: 1,
        playerName: "Bob",
        cardRole: "Captain",
        reason: "lost_last_influence",
        turn: 5,
        isElimination: true,
        timestamp: 10000,
      };
      const state: GameSliceState = {
        ...initialGameSliceState,
        discardQueue: [discard],
        currentDiscard: discard,
        eliminatingPlayer: 1,
      };
      const action = gameActions.dismissDiscard();
      const newState = gameReducer(state, action);

      expect(newState.eliminatingPlayer).toBeNull();
    });

    it("should preserve eliminatingPlayer if currentDiscard was not elimination", () => {
      const discard: CardDiscardEvent = {
        playerIndex: 0,
        playerName: "Alice",
        cardRole: "Duke",
        reason: "coup",
        turn: 2,
        isElimination: false,
        timestamp: 10000,
      };
      const state: GameSliceState = {
        ...initialGameSliceState,
        discardQueue: [discard],
        currentDiscard: discard,
        eliminatingPlayer: 2,
      };
      const action = gameActions.dismissDiscard();
      const newState = gameReducer(state, action);

      expect(newState.eliminatingPlayer).toBe(2);
    });

    it("should handle empty queue", () => {
      const action = gameActions.dismissDiscard();
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.discardQueue).toEqual([]);
      expect(newState.currentDiscard).toBeNull();
    });
  });

  describe("RESET action", () => {
    it("should reset to initial state", () => {
      const state: GameSliceState = {
        currentMatch: { matchId: "m1", lobbyId: "l1", playerNames: ["A"], participantIds: ["p1"] },
        identity: { playerId: "p1", playerIndex: 0, playerNames: ["A"] },
        players: [{ index: 0, name: "Alice", alive: true, coins: 5, cardCount: 2, avatar: "" }],
        roles: ["Duke", "Assassin"],
        hand: [{ id: "1", role: "Duke" }],
        activePlayerIndex: 0,
        pendingPrompt: { kind: "action", requestId: "r1", allowedActions: [] },
        promptClosedReason: "completed",
        targeting: { active: true, actionId: "a1", requestId: "r1", selectedTarget: null },
        turnTimer: { activePlayerIndex: 0, durationMs: 30000, running: true, paused: false, key: "k1" },
        pause: { status: "paused", pausedByPlayerId: "p1", pausedByIndex: 0, pausedByName: "A", deadlineMs: 1, durationMs: 2, pauseReason: "", eligibleVoters: [], kickVotes: [] },
        gameOver: { winnerIndex: 0, winnerName: "Alice" },
        discardQueue: [],
        currentDiscard: null,
        eliminatingPlayer: null,
      };
      const action = gameActions.reset();
      const newState = gameReducer(state, action);

      expect(newState).toEqual(initialGameSliceState);
    });

    it("should handle reset from already initial state", () => {
      const action = gameActions.reset();
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState).toEqual(initialGameSliceState);
    });
  });

  describe("Unknown actions", () => {
    it("should return current state for unknown action type", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        players: [{ index: 0, name: "Alice", alive: true, coins: 5, cardCount: 2, avatar: "" }],
        roles: ["Duke"],
      };
      const action = { type: "UNKNOWN_ACTION" } as unknown as GameAction;
      const newState = gameReducer(state, action);

      expect(newState).toEqual(state);
    });

    it("should handle random action type gracefully", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        activePlayerIndex: 2,
      };
      const action = { type: "RANDOM_TYPE" } as unknown as GameAction;
      const newState = gameReducer(state, action);

      expect(newState.activePlayerIndex).toBe(2);
    });
  });

  describe("State Transitions", () => {
    it("should handle full game lifecycle: lobby start → game config → game state → game over → reset", () => {
      let state = initialGameSliceState;

      // Lobby started
      state = gameReducer(state, gameActions.lobbyStarted({
        lobby_id: "lobby-1",
        match_id: "match-1",
        player_index: 0,
        player_count: 2,
        player_names: ["Alice", "Bob"],
      }, "player-1", null));
      expect(state.players).toHaveLength(2);
      expect(state.identity?.playerIndex).toBe(0);

      // Game config
      state = gameReducer(state, gameActions.gameConfig({ roles: ["Duke", "Assassin"] }));
      expect(state.roles).toEqual(["Duke", "Assassin"]);

      // Game state update
      state = gameReducer(state, gameActions.gameState({
        players: [
          { index: 0, name: "Alice", alive: true, coins: 2, card_count: 2 },
          { index: 1, name: "Bob", alive: true, coins: 2, card_count: 2 },
        ],
        active_player_index: 0,
      }, null));
      expect(state.players[0].coins).toBe(2);

      // Game over
      state = gameReducer(state, gameActions.gameOver(0, "Alice"));
      expect(state.gameOver?.winnerName).toBe("Alice");

      // Reset
      state = gameReducer(state, gameActions.reset());
      expect(state.currentMatch).toBeNull();
      expect(state.players).toEqual([]);
    });

    it("should handle turn progression through multiple states", () => {
      const identity: GameIdentity = { playerId: "p1", playerIndex: 0, playerNames: ["Alice", "Bob"] };
      let state: GameSliceState = {
        ...initialGameSliceState,
        identity,
        players: [
          { index: 0, name: "Alice", alive: true, coins: 2, cardCount: 2, avatar: "" },
          { index: 1, name: "Bob", alive: true, coins: 2, cardCount: 2, avatar: "" },
        ],
      };

      // Turn timer starts
      state = gameReducer(state, gameActions.turnTimer({
        active_player_index: 0,
        duration_ms: 30000,
        state: "start",
        turn_number: 1,
      }));
      expect(state.turnTimer?.running).toBe(true);

      // Action requested
      state = gameReducer(state, gameActions.requestAction({
        actor_index: 0,
        allowed_actions: ["income", "foreign_aid"],
      }, "req-1", identity));
      expect(state.pendingPrompt?.kind).toBe("action");

      // Clear prompt
      state = gameReducer(state, gameActions.clearPrompt());
      expect(state.pendingPrompt).toBeNull();

      // Next turn
      state = gameReducer(state, gameActions.turnTimer({
        active_player_index: 1,
        duration_ms: 30000,
        state: "start",
        turn_number: 2,
      }));
      expect(state.turnTimer?.activePlayerIndex).toBe(1);
    });

    it("should handle challenge flow: action → challenge → resolution", () => {
      const identity: GameIdentity = { playerId: "p1", playerIndex: 0, playerNames: ["Alice", "Bob", "Charlie"] };
      let state: GameSliceState = {
        ...initialGameSliceState,
        identity,
        players: [
          { index: 0, name: "Alice", alive: true, coins: 3, cardCount: 2, avatar: "" },
          { index: 1, name: "Bob", alive: true, coins: 2, cardCount: 2, avatar: "" },
          { index: 2, name: "Charlie", alive: true, coins: 2, cardCount: 2, avatar: "" },
        ],
      };

      // Action taken
      state = gameReducer(state, gameActions.requestAction({
        actor_index: 0,
        allowed_actions: ["duke", "captain"],
      }, "req-1", identity));

      // Challenge window opens
      state = gameReducer(state, gameActions.challengeWindow({
        action_id: "duke",
        actor_index: 0,
        claimed_role: "Duke",
        eligible: true,
        kind: "main",
        target_index: 1,
      }, "req-2"));
      expect(state.pendingPrompt?.kind).toBe("challenge");

      // Challenge resolved, card discarded
      state = gameReducer(state, gameActions.cardDiscarded({
        playerIndex: 0,
        playerName: "Alice",
        cardRole: "Duke",
        reason: "challenge_failed",
        turn: 3,
        isElimination: false,
        timestamp: Date.now(),
      }));
      expect(state.discardQueue).toHaveLength(1);

      // Dismiss discard
      state = gameReducer(state, gameActions.dismissDiscard());
      expect(state.discardQueue).toHaveLength(0);
    });

    it("should handle targeting flow: set → select → clear", () => {
      let state = initialGameSliceState;

      // Set targeting
      state = gameReducer(state, gameActions.setTargeting("coup", "req-1"));
      expect(state.targeting?.active).toBe(true);
      expect(state.targeting?.selectedTarget).toBeNull();

      // Select target
      state = gameReducer(state, gameActions.setTargetSelected(2));
      expect(state.targeting?.selectedTarget).toBe(2);

      // Clear targeting
      state = gameReducer(state, gameActions.clearTargeting());
      expect(state.targeting).toBeNull();
    });

    it("should handle pause lifecycle: active → paused → resumed → inactive", () => {
      let state: GameSliceState = {
        ...initialGameSliceState,
        pause: { status: "inactive" },
      };

      // Game paused
      state = gameReducer(state, gameActions.gamePaused({
        paused_by_player_id: "p1",
        paused_by_index: 0,
        paused_by_name: "Alice",
        deadline_ms: 60000,
        duration_ms: 300000,
        pause_reason: "AFK",
        eligible_voters: [0, 1, 2],
        kick_votes: [],
      }));
      expect(state.pause.status).toBe("paused");

      // Vote update
      state = gameReducer(state, gameActions.kickVoteUpdate({
        eligible_voters: [0, 1, 2],
        kick_votes: [1, 2],
      }));
      const pauseState = state.pause;
      expect(pauseState.status === "paused" ? pauseState.kickVotes : undefined).toEqual([1, 2]);

      // Game resumed
      state = gameReducer(state, gameActions.gameResumed({
        resumed_by_player_id: "p2",
        resumed_by_index: 1,
        resumed_by_name: "Bob",
        resume_reason: "Player returned",
      }));
      expect(state.pause.status).toBe("resumed");
    });
  });

  describe("Action Creators", () => {
    it("should create LOBBY_STARTED action", () => {
      const payload = {
        lobby_id: "lobby-1",
        player_index: 0,
        player_count: 2,
        player_names: ["Alice", "Bob"],
      };
      const action = gameActions.lobbyStarted(payload, "player-1", null);
      expect(action).toEqual({
        type: "LOBBY_STARTED",
        payload,
        currentPlayerId: "player-1",
        currentLobby: null,
      });
    });

    it("should create GAME_CONFIG action", () => {
      const payload = { roles: ["Duke", "Assassin"] };
      const action = gameActions.gameConfig(payload);
      expect(action).toEqual({
        type: "GAME_CONFIG",
        payload,
      });
    });

    it("should create GAME_STATE action", () => {
      const payload = {
        players: [{ index: 0, name: "Alice", alive: true, coins: 2, card_count: 2 }],
        active_player_index: 0,
      };
      const identity: GameIdentity | null = { playerId: "p1", playerIndex: 0, playerNames: ["Alice"] };
      const action = gameActions.gameState(payload, identity);
      expect(action).toEqual({
        type: "GAME_STATE",
        payload,
        currentIdentity: identity,
      });
    });

    it("should create REQUEST_ACTION action", () => {
      const payload = { actor_index: 0, allowed_actions: ["income"] };
      const identity: GameIdentity | null = { playerId: "p1", playerIndex: 0, playerNames: ["Alice"] };
      const action = gameActions.requestAction(payload, "req-1", identity);
      expect(action).toEqual({
        type: "REQUEST_ACTION",
        payload,
        requestId: "req-1",
        currentIdentity: identity,
      });
    });

    it("should create CHALLENGE_WINDOW action", () => {
      const payload = {
        action_id: "a1",
        actor_index: 0,
        claimed_role: "Duke",
        eligible: true,
        kind: "main",
      };
      const action = gameActions.challengeWindow(payload, "req-1");
      expect(action).toEqual({
        type: "CHALLENGE_WINDOW",
        payload,
        requestId: "req-1",
      });
    });

    it("should create COUNTER_WINDOW action", () => {
      const payload = {
        action_id: "a1",
        actor_index: 1,
        allowed_actions: ["pass"],
        eligible: true,
      };
      const action = gameActions.counterWindow(payload, "req-1");
      expect(action).toEqual({
        type: "COUNTER_WINDOW",
        payload,
        requestId: "req-1",
      });
    });

    it("should create REQUEST_STEP action", () => {
      const payload = {
        prompt: "Choose",
        step: { context: "exchange", count: 1, options: ["Duke", "Assassin"] },
      };
      const action = gameActions.requestStep(payload, "req-1");
      expect(action).toEqual({
        type: "REQUEST_STEP",
        payload,
        requestId: "req-1",
      });
    });

    it("should create HAND_STATE action", () => {
      const payload = { hand: ["Duke", "Assassin"], player_index: 0 };
      const identity: GameIdentity | null = { playerId: "p1", playerIndex: 0, playerNames: ["Alice"] };
      const action = gameActions.handState(payload, identity);
      expect(action).toEqual({
        type: "HAND_STATE",
        payload,
        currentIdentity: identity,
      });
    });

    it("should create PROMPT_CLOSED action", () => {
      const payload = { reason: "completed" };
      const currentPrompt: Prompt | null = { kind: "action", requestId: "req-1", allowedActions: [] };
      const action = gameActions.promptClosed(payload, "req-1", currentPrompt);
      expect(action).toEqual({
        type: "PROMPT_CLOSED",
        payload,
        requestId: "req-1",
        currentPrompt,
      });
    });

    it("should create TURN_TIMER action", () => {
      const payload = {
        active_player_index: 0,
        duration_ms: 30000,
        state: "start",
        turn_number: 1,
      };
      const action = gameActions.turnTimer(payload);
      expect(action).toEqual({
        type: "TURN_TIMER",
        payload,
      });
    });

    it("should create CLEAR_PROMPT action", () => {
      const action = gameActions.clearPrompt();
      expect(action).toEqual({ type: "CLEAR_PROMPT" });
    });

    it("should create SET_TARGETING action", () => {
      const action = gameActions.setTargeting("action-1", "req-1");
      expect(action).toEqual({
        type: "SET_TARGETING",
        actionId: "action-1",
        requestId: "req-1",
      });
    });

    it("should create SET_TARGET_SELECTED action", () => {
      const action = gameActions.setTargetSelected(2);
      expect(action).toEqual({
        type: "SET_TARGET_SELECTED",
        targetIndex: 2,
      });
    });

    it("should create CLEAR_TARGETING action", () => {
      const action = gameActions.clearTargeting();
      expect(action).toEqual({ type: "CLEAR_TARGETING" });
    });

    it("should create GAME_OVER action", () => {
      const action = gameActions.gameOver(1, "Bob");
      expect(action).toEqual({
        type: "GAME_OVER",
        winnerIndex: 1,
        winnerName: "Bob",
      });
    });

    it("should create CARD_DISCARDED action", () => {
      const payload: CardDiscardEvent = {
        playerIndex: 0,
        playerName: "Alice",
        cardRole: "Duke",
        reason: "coup",
        turn: 1,
        isElimination: false,
        timestamp: 12345,
      };
      const action = gameActions.cardDiscarded(payload);
      expect(action).toEqual({
        type: "CARD_DISCARDED",
        payload,
      });
    });

    it("should create DISMISS_DISCARD action", () => {
      const action = gameActions.dismissDiscard();
      expect(action).toEqual({ type: "DISMISS_DISCARD" });
    });

    it("should create PLAYER_ELIMINATED action", () => {
      const payload = { player_index: 1, reason: "coup", turn: 5 };
      const action = gameActions.playerEliminated(payload);
      expect(action).toEqual({
        type: "PLAYER_ELIMINATED",
        payload,
      });
    });

    it("should create GAME_PAUSED action", () => {
      const payload = {
        paused_by_player_id: "p1",
        paused_by_index: 0,
        paused_by_name: "Alice",
        deadline_ms: 60000,
        duration_ms: 300000,
        pause_reason: "AFK",
        eligible_voters: [0, 1],
        kick_votes: [],
      };
      const action = gameActions.gamePaused(payload);
      expect(action).toEqual({
        type: "GAME_PAUSED",
        payload,
      });
    });

    it("should create GAME_RESUMED action", () => {
      const payload = {
        resumed_by_player_id: "p1",
        resumed_by_index: 0,
        resumed_by_name: "Alice",
        resume_reason: "Returned",
      };
      const action = gameActions.gameResumed(payload);
      expect(action).toEqual({
        type: "GAME_RESUMED",
        payload,
      });
    });

    it("should create KICK_VOTE_UPDATE action", () => {
      const payload = { eligible_voters: [0, 1], kick_votes: [0] };
      const action = gameActions.kickVoteUpdate(payload);
      expect(action).toEqual({
        type: "KICK_VOTE_UPDATE",
        payload,
      });
    });

    it("should create PLAYER_KICKED action", () => {
      const payload = { player_index: 1, reason: "voted_to_kick" };
      const action = gameActions.playerKicked(payload);
      expect(action).toEqual({
        type: "PLAYER_KICKED",
        payload,
      });
    });

    it("should create RESET action", () => {
      const action = gameActions.reset();
      expect(action).toEqual({ type: "RESET" });
    });
  });

  describe("Edge Cases", () => {
    it("should handle rapid state changes", () => {
      let state = initialGameSliceState;

      // Multiple rapid changes
      state = gameReducer(state, gameActions.gameConfig({ roles: ["Duke"] }));
      state = gameReducer(state, gameActions.turnTimer({
        active_player_index: 0,
        duration_ms: 30000,
        state: "start",
        turn_number: 1,
      }));
      state = gameReducer(state, gameActions.clearPrompt());
      state = gameReducer(state, gameActions.setTargeting("action", "req"));
      state = gameReducer(state, gameActions.clearTargeting());

      expect(state.roles).toEqual(["Duke"]);
      expect(state.turnTimer?.running).toBe(true);
      expect(state.targeting).toBeNull();
    });

    it("should handle empty player names array", () => {
      const payload = {
        lobby_id: "lobby-1",
        player_index: 0,
        player_count: 0,
        player_names: [],
      };
      const action = gameActions.lobbyStarted(payload, "p1", null);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.players).toEqual([]);
    });

    it("should handle null values in game state payload", () => {
      const payload = {
        players: [
          { index: 0, name: "Alice", alive: true, coins: null as unknown as number, card_count: null as unknown as number },
        ],
        active_player_index: 0,
      };
      const action = gameActions.gameState(payload, null);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.players[0].coins).toBeNull();
      expect(newState.players[0].cardCount).toBeNull();
    });

    it("should handle setting target when targeting is null", () => {
      const action = gameActions.setTargetSelected(1);
      const newState = gameReducer(initialGameSliceState, action);

      // Should not crash, state unchanged
      expect(newState).toEqual(initialGameSliceState);
    });

    it("should handle multiple discards in sequence", () => {
      let state = initialGameSliceState;

      const discard1: CardDiscardEvent = {
        playerIndex: 0,
        playerName: "Alice",
        cardRole: "Duke",
        reason: "coup",
        turn: 1,
        isElimination: false,
        timestamp: 10000,
      };
      const discard2: CardDiscardEvent = {
        playerIndex: 1,
        playerName: "Bob",
        cardRole: "Assassin",
        reason: "challenge",
        turn: 2,
        isElimination: false,
        timestamp: 20000,
      };
      const discard3: CardDiscardEvent = {
        playerIndex: 0,
        playerName: "Alice",
        cardRole: "Captain",
        reason: "elimination",
        turn: 3,
        isElimination: true,
        timestamp: 30000,
      };

      state = gameReducer(state, gameActions.cardDiscarded(discard1));
      state = gameReducer(state, gameActions.cardDiscarded(discard2));
      state = gameReducer(state, gameActions.cardDiscarded(discard3));

      expect(state.discardQueue).toHaveLength(3);
      expect(state.currentDiscard).toEqual(discard1);
      expect(state.eliminatingPlayer).toBe(0);

      // Dismiss all
      state = gameReducer(state, gameActions.dismissDiscard());
      state = gameReducer(state, gameActions.dismissDiscard());
      state = gameReducer(state, gameActions.dismissDiscard());

      expect(state.discardQueue).toHaveLength(0);
      expect(state.currentDiscard).toBeNull();
      expect(state.eliminatingPlayer).toBeNull();
    });

    it("should maintain state reference integrity", () => {
      const state: GameSliceState = {
        ...initialGameSliceState,
        players: [{ index: 0, name: "Alice", alive: true, coins: 2, cardCount: 2, avatar: "" }],
      };
      // Use LOBBY_STARTED action which creates new players array
      const action = gameActions.lobbyStarted({
        lobby_id: "lobby-123",
        player_index: 0,
        player_count: 1,
        player_names: ["Bob"],
      }, "player-1", null);
      const newState = gameReducer(state, action);

      expect(newState).not.toBe(state);
      expect(newState.players).not.toBe(state.players);
    });

    it("should not mutate original state", () => {
      const originalState: GameSliceState = {
        currentMatch: null,
        identity: null,
        players: [],
        roles: [],
        hand: [],
        activePlayerIndex: null,
        pendingPrompt: null,
        promptClosedReason: null,
        targeting: null,
        turnTimer: null,
        pause: { status: "inactive" },
        gameOver: null,
        discardQueue: [],
        currentDiscard: null,
        eliminatingPlayer: null,
      };

      const stateCopy = { ...originalState, pause: { ...originalState.pause } };
      const action = gameActions.gameConfig({ roles: ["Duke"] });
      gameReducer(originalState, action);

      expect(originalState.roles).toEqual([]);
      expect(originalState).toEqual(stateCopy);
    });
  });
});
