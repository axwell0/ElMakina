/**
 * UI slice tests
 *
 * Tests for UI state management including preferences, logs, chat,
 * transient UI state (investigation, elimination, turn timer), and replay history.
 */

import { describe, it, expect } from "vitest";
import {
  uiReducer,
  uiActions,
  initialUIState,
  type UISliceState,
  type UIAction,
} from "@/state/slices/ui";
import type {
  LogEntry,
  ChatMessage,
  InvestigateResult,
  EliminationToast,
  TurnTimerState,
  ReplayEntry,
} from "@/state/types";

describe("UI Slice", () => {
  describe("Initial State", () => {
    it("should have correct initial values", () => {
      expect(initialUIState.sfxMuted).toBe(false);
      expect(initialUIState.theme).toBe("light");
      expect(initialUIState.logs).toEqual([]);
      expect(initialUIState.chat).toEqual([]);
      expect(initialUIState.investigateResult).toBeNull();
      expect(initialUIState.eliminationToast).toBeNull();
      expect(initialUIState.turnTimer).toBeNull();
      expect(initialUIState.replayHistory).toEqual([]);
      expect(initialUIState.lastUpdateTs).toBe(0);
      expect(initialUIState.mockScenario).toBeUndefined();
    });
  });

  describe("SET_SFX_MUTED action", () => {
    it("should set sfxMuted to true", () => {
      const action = uiActions.setSfxMuted(true);
      const newState = uiReducer(initialUIState, action);

      expect(newState.sfxMuted).toBe(true);
    });

    it("should set sfxMuted to false", () => {
      const state: UISliceState = { ...initialUIState, sfxMuted: true };
      const action = uiActions.setSfxMuted(false);
      const newState = uiReducer(state, action);

      expect(newState.sfxMuted).toBe(false);
    });

    it("should preserve other state when muting", () => {
      const state: UISliceState = {
        ...initialUIState,
        theme: "dark",
        logs: [{ turn: 1, scope: "public", message: "test" }],
      };
      const action = uiActions.setSfxMuted(true);
      const newState = uiReducer(state, action);

      expect(newState.sfxMuted).toBe(true);
      expect(newState.theme).toBe("dark");
      expect(newState.logs).toHaveLength(1);
    });
  });

  describe("SET_THEME action", () => {
    it("should set theme to dark", () => {
      const action = uiActions.setTheme("dark");
      const newState = uiReducer(initialUIState, action);

      expect(newState.theme).toBe("dark");
    });

    it("should set theme to light", () => {
      const state: UISliceState = { ...initialUIState, theme: "dark" };
      const action = uiActions.setTheme("light");
      const newState = uiReducer(state, action);

      expect(newState.theme).toBe("light");
    });

    it("should preserve other state when changing theme", () => {
      const state: UISliceState = {
        ...initialUIState,
        sfxMuted: true,
        chat: [{ id: "1", senderIndex: 0, senderName: "Player1", text: "Hello", timestamp: 1000 }],
      };
      const action = uiActions.setTheme("dark");
      const newState = uiReducer(state, action);

      expect(newState.theme).toBe("dark");
      expect(newState.sfxMuted).toBe(true);
      expect(newState.chat).toHaveLength(1);
    });
  });

  describe("GAME_LOG action", () => {
    it("should add log entry to empty logs", () => {
      const entry: LogEntry = { turn: 1, scope: "public", message: "Game started" };
      const action = uiActions.gameLog(entry);
      const newState = uiReducer(initialUIState, action);

      expect(newState.logs).toHaveLength(1);
      expect(newState.logs[0]).toEqual(entry);
    });

    it("should append log entry to existing logs", () => {
      const state: UISliceState = {
        ...initialUIState,
        logs: [{ turn: 1, scope: "public", message: "First log" }],
      };
      const entry: LogEntry = { turn: 2, scope: "private", message: "Second log" };
      const action = uiActions.gameLog(entry);
      const newState = uiReducer(state, action);

      expect(newState.logs).toHaveLength(2);
      expect(newState.logs[0].message).toBe("First log");
      expect(newState.logs[1].message).toBe("Second log");
    });

    it("should update lastUpdateTs when adding log", () => {
      const beforeAction = Date.now();
      const entry: LogEntry = { turn: 1, scope: "public", message: "Test" };
      const action = uiActions.gameLog(entry);
      const newState = uiReducer(initialUIState, action);
      const afterAction = Date.now();

      expect(newState.lastUpdateTs).toBeGreaterThanOrEqual(beforeAction);
      expect(newState.lastUpdateTs).toBeLessThanOrEqual(afterAction);
    });

    it("should handle multiple log entries", () => {
      let state = initialUIState;

      state = uiReducer(state, uiActions.gameLog({ turn: 1, scope: "public", message: "Log 1" }));
      state = uiReducer(state, uiActions.gameLog({ turn: 1, scope: "public", message: "Log 2" }));
      state = uiReducer(state, uiActions.gameLog({ turn: 2, scope: "private", message: "Log 3" }));

      expect(state.logs).toHaveLength(3);
      expect(state.logs[2].message).toBe("Log 3");
    });
  });

  describe("CHAT_MESSAGE action", () => {
    it("should add chat message to empty chat", () => {
      const message: ChatMessage = {
        id: "msg-1",
        senderIndex: 0,
        senderName: "Player1",
        text: "Hello!",
        timestamp: 1000,
      };
      const action = uiActions.chatMessage(message);
      const newState = uiReducer(initialUIState, action);

      expect(newState.chat).toHaveLength(1);
      expect(newState.chat[0]).toEqual(message);
    });

    it("should append chat message to existing chat", () => {
      const state: UISliceState = {
        ...initialUIState,
        chat: [{ id: "msg-1", senderIndex: 0, senderName: "Player1", text: "Hi", timestamp: 1000 }],
      };
      const message: ChatMessage = {
        id: "msg-2",
        senderIndex: 1,
        senderName: "Player2",
        text: "Hello!",
        timestamp: 2000,
      };
      const action = uiActions.chatMessage(message);
      const newState = uiReducer(state, action);

      expect(newState.chat).toHaveLength(2);
      expect(newState.chat[1].text).toBe("Hello!");
    });

    it("should keep only last 100 messages", () => {
      let state = initialUIState;

      // Add 105 messages
      for (let i = 0; i < 105; i++) {
        state = uiReducer(
          state,
          uiActions.chatMessage({
            id: `msg-${i}`,
            senderIndex: 0,
            senderName: "Player",
            text: `Message ${i}`,
            timestamp: i * 1000,
          })
        );
      }

      expect(state.chat).toHaveLength(100);
      expect(state.chat[0].text).toBe("Message 5"); // First 5 removed
      expect(state.chat[99].text).toBe("Message 104");
    });

    it("should update lastUpdateTs when adding chat message", () => {
      const beforeAction = Date.now();
      const message: ChatMessage = {
        id: "msg-1",
        senderIndex: 0,
        senderName: "Player1",
        text: "Test",
        timestamp: 1000,
      };
      const action = uiActions.chatMessage(message);
      const newState = uiReducer(initialUIState, action);
      const afterAction = Date.now();

      expect(newState.lastUpdateTs).toBeGreaterThanOrEqual(beforeAction);
      expect(newState.lastUpdateTs).toBeLessThanOrEqual(afterAction);
    });
  });

  describe("INVESTIGATE_RESULT action", () => {
    it("should set investigate result", () => {
      const result: InvestigateResult = { targetName: "Player1", role: "Duke" };
      const action = uiActions.investigateResult(result);
      const newState = uiReducer(initialUIState, action);

      expect(newState.investigateResult).toEqual(result);
    });

    it("should overwrite existing investigate result", () => {
      const state: UISliceState = {
        ...initialUIState,
        investigateResult: { targetName: "OldPlayer", role: "Captain" },
      };
      const result: InvestigateResult = { targetName: "NewPlayer", role: "Assassin" };
      const action = uiActions.investigateResult(result);
      const newState = uiReducer(state, action);

      expect(newState.investigateResult).toEqual(result);
    });

    it("should preserve other state when setting investigate result", () => {
      const state: UISliceState = {
        ...initialUIState,
        theme: "dark",
        logs: [{ turn: 1, scope: "public", message: "test" }],
      };
      const result: InvestigateResult = { targetName: "Player1", role: "Contessa" };
      const action = uiActions.investigateResult(result);
      const newState = uiReducer(state, action);

      expect(newState.investigateResult).toEqual(result);
      expect(newState.theme).toBe("dark");
      expect(newState.logs).toHaveLength(1);
    });
  });

  describe("CLEAR_INVESTIGATE action", () => {
    it("should clear investigate result", () => {
      const state: UISliceState = {
        ...initialUIState,
        investigateResult: { targetName: "Player1", role: "Duke" },
      };
      const action = uiActions.clearInvestigate();
      const newState = uiReducer(state, action);

      expect(newState.investigateResult).toBeNull();
    });

    it("should handle clearing when no investigate result exists", () => {
      const action = uiActions.clearInvestigate();
      const newState = uiReducer(initialUIState, action);

      expect(newState.investigateResult).toBeNull();
    });

    it("should preserve other state when clearing investigate", () => {
      const state: UISliceState = {
        ...initialUIState,
        investigateResult: { targetName: "Player1", role: "Duke" },
        theme: "dark",
        chat: [{ id: "1", senderIndex: 0, senderName: "Player1", text: "Hi", timestamp: 1000 }],
      };
      const action = uiActions.clearInvestigate();
      const newState = uiReducer(state, action);

      expect(newState.investigateResult).toBeNull();
      expect(newState.theme).toBe("dark");
      expect(newState.chat).toHaveLength(1);
    });
  });

  describe("ELIMINATION action", () => {
    it("should set elimination toast", () => {
      const toast: EliminationToast = {
        playerIndex: 1,
        playerName: "Player2",
        reason: "Lost influence",
        turn: 5,
        id: "elim-1",
      };
      const action = uiActions.elimination(toast);
      const newState = uiReducer(initialUIState, action);

      expect(newState.eliminationToast).toEqual(toast);
    });

    it("should overwrite existing elimination toast", () => {
      const state: UISliceState = {
        ...initialUIState,
        eliminationToast: {
          playerIndex: 0,
          playerName: "OldPlayer",
          reason: "Old reason",
          turn: 1,
          id: "elim-old",
        },
      };
      const toast: EliminationToast = {
        playerIndex: 2,
        playerName: "NewPlayer",
        reason: "Coup",
        turn: 10,
        id: "elim-new",
      };
      const action = uiActions.elimination(toast);
      const newState = uiReducer(state, action);

      expect(newState.eliminationToast).toEqual(toast);
    });

    it("should preserve other state when setting elimination toast", () => {
      const state: UISliceState = {
        ...initialUIState,
        theme: "dark",
        sfxMuted: true,
      };
      const toast: EliminationToast = {
        playerIndex: 0,
        playerName: "Player1",
        reason: "Eliminated",
        turn: 3,
        id: "elim-1",
      };
      const action = uiActions.elimination(toast);
      const newState = uiReducer(state, action);

      expect(newState.eliminationToast).toEqual(toast);
      expect(newState.theme).toBe("dark");
      expect(newState.sfxMuted).toBe(true);
    });
  });

  describe("CLEAR_ELIMINATION_TOAST action", () => {
    it("should clear elimination toast", () => {
      const state: UISliceState = {
        ...initialUIState,
        eliminationToast: {
          playerIndex: 1,
          playerName: "Player2",
          reason: "Lost influence",
          turn: 5,
          id: "elim-1",
        },
      };
      const action = uiActions.clearEliminationToast();
      const newState = uiReducer(state, action);

      expect(newState.eliminationToast).toBeNull();
    });

    it("should handle clearing when no elimination toast exists", () => {
      const action = uiActions.clearEliminationToast();
      const newState = uiReducer(initialUIState, action);

      expect(newState.eliminationToast).toBeNull();
    });

    it("should preserve other state when clearing elimination toast", () => {
      const state: UISliceState = {
        ...initialUIState,
        eliminationToast: {
          playerIndex: 0,
          playerName: "Player1",
          reason: "Eliminated",
          turn: 3,
          id: "elim-1",
        },
        logs: [{ turn: 1, scope: "public", message: "test" }],
      };
      const action = uiActions.clearEliminationToast();
      const newState = uiReducer(state, action);

      expect(newState.eliminationToast).toBeNull();
      expect(newState.logs).toHaveLength(1);
    });
  });

  describe("TURN_TIMER action", () => {
    it("should set turn timer", () => {
      const timer: TurnTimerState = {
        activePlayerIndex: 0,
        durationMs: 30000,
        running: true,
        paused: false,
        key: "timer-1",
      };
      const action = uiActions.turnTimer(timer);
      const newState = uiReducer(initialUIState, action);

      expect(newState.turnTimer).toEqual(timer);
    });

    it("should update existing turn timer", () => {
      const state: UISliceState = {
        ...initialUIState,
        turnTimer: {
          activePlayerIndex: 0,
          durationMs: 30000,
          running: true,
          paused: false,
          key: "timer-old",
        },
      };
      const timer: TurnTimerState = {
        activePlayerIndex: 1,
        durationMs: 45000,
        running: false,
        paused: true,
        key: "timer-new",
      };
      const action = uiActions.turnTimer(timer);
      const newState = uiReducer(state, action);

      expect(newState.turnTimer).toEqual(timer);
    });

    it("should set turn timer to null", () => {
      const state: UISliceState = {
        ...initialUIState,
        turnTimer: {
          activePlayerIndex: 0,
          durationMs: 30000,
          running: true,
          paused: false,
          key: "timer-1",
        },
      };
      const action = uiActions.turnTimer(null);
      const newState = uiReducer(state, action);

      expect(newState.turnTimer).toBeNull();
    });

    it("should preserve other state when setting turn timer", () => {
      const state: UISliceState = {
        ...initialUIState,
        theme: "dark",
        investigateResult: { targetName: "Player1", role: "Duke" },
      };
      const timer: TurnTimerState = {
        activePlayerIndex: 2,
        durationMs: 60000,
        running: true,
        paused: false,
        key: "timer-1",
      };
      const action = uiActions.turnTimer(timer);
      const newState = uiReducer(state, action);

      expect(newState.turnTimer).toEqual(timer);
      expect(newState.theme).toBe("dark");
      expect(newState.investigateResult).not.toBeNull();
    });
  });

  describe("ADD_REPLAY action", () => {
    it("should add replay entry to empty history", () => {
      const entry: ReplayEntry = {
        matchId: "match-1",
        lobbyId: "lobby-1",
        playerId: "player-1",
        playerNames: ["Player1", "Player2"],
        participantIds: ["p1", "p2"],
        endedAt: 1000,
      };
      const action = uiActions.addReplay(entry);
      const newState = uiReducer(initialUIState, action);

      expect(newState.replayHistory).toHaveLength(1);
      expect(newState.replayHistory[0]).toEqual(entry);
    });

    it("should prepend new replay entry", () => {
      const state: UISliceState = {
        ...initialUIState,
        replayHistory: [
          {
            matchId: "match-1",
            lobbyId: "lobby-1",
            playerId: "player-1",
            playerNames: ["Player1"],
            participantIds: ["p1"],
            endedAt: 1000,
          },
        ],
      };
      const entry: ReplayEntry = {
        matchId: "match-2",
        lobbyId: "lobby-2",
        playerId: "player-2",
        playerNames: ["Player2"],
        participantIds: ["p2"],
        endedAt: 2000,
      };
      const action = uiActions.addReplay(entry);
      const newState = uiReducer(state, action);

      expect(newState.replayHistory).toHaveLength(2);
      expect(newState.replayHistory[0].matchId).toBe("match-2");
      expect(newState.replayHistory[1].matchId).toBe("match-1");
    });

    it("should deduplicate by matchId", () => {
      const state: UISliceState = {
        ...initialUIState,
        replayHistory: [
          {
            matchId: "match-1",
            lobbyId: "lobby-1",
            playerId: "player-1",
            playerNames: ["OldPlayer"],
            participantIds: ["p1"],
            endedAt: 1000,
          },
        ],
      };
      const entry: ReplayEntry = {
        matchId: "match-1",
        lobbyId: "lobby-1",
        playerId: "player-1",
        playerNames: ["NewPlayer"],
        participantIds: ["p1"],
        endedAt: 2000,
        winnerName: "Winner",
      };
      const action = uiActions.addReplay(entry);
      const newState = uiReducer(state, action);

      expect(newState.replayHistory).toHaveLength(1);
      expect(newState.replayHistory[0].playerNames[0]).toBe("NewPlayer");
      expect(newState.replayHistory[0].winnerName).toBe("Winner");
    });

    it("should keep only last 30 replay entries", () => {
      let state = initialUIState;

      // Add 35 entries
      for (let i = 0; i < 35; i++) {
        state = uiReducer(
          state,
          uiActions.addReplay({
            matchId: `match-${i}`,
            lobbyId: `lobby-${i}`,
            playerId: `player-${i}`,
            playerNames: [`Player${i}`],
            participantIds: [`p${i}`],
            endedAt: i * 1000,
          })
        );
      }

      expect(state.replayHistory).toHaveLength(30);
      expect(state.replayHistory[0].matchId).toBe("match-34");
      expect(state.replayHistory[29].matchId).toBe("match-5");
    });

    it("should move duplicate entry to front when re-added", () => {
      let state: UISliceState = {
        ...initialUIState,
        replayHistory: [
          { matchId: "match-1", lobbyId: "l1", playerId: "p1", playerNames: ["A"], participantIds: ["a"], endedAt: 1000 },
          { matchId: "match-2", lobbyId: "l2", playerId: "p2", playerNames: ["B"], participantIds: ["b"], endedAt: 2000 },
          { matchId: "match-3", lobbyId: "l3", playerId: "p3", playerNames: ["C"], participantIds: ["c"], endedAt: 3000 },
        ],
      };

      // Re-add match-2 (should move to front)
      const action = uiActions.addReplay({
        matchId: "match-2",
        lobbyId: "l2",
        playerId: "p2",
        playerNames: ["B"],
        participantIds: ["b"],
        endedAt: 4000,
      });
      state = uiReducer(state, action);

      expect(state.replayHistory).toHaveLength(3);
      expect(state.replayHistory[0].matchId).toBe("match-2");
      expect(state.replayHistory[1].matchId).toBe("match-1");
      expect(state.replayHistory[2].matchId).toBe("match-3");
    });
  });

  describe("UPDATE_TIMESTAMP action", () => {
    it("should update lastUpdateTs", () => {
      const beforeAction = Date.now();
      const action = uiActions.updateTimestamp();
      const newState = uiReducer(initialUIState, action);
      const afterAction = Date.now();

      expect(newState.lastUpdateTs).toBeGreaterThanOrEqual(beforeAction);
      expect(newState.lastUpdateTs).toBeLessThanOrEqual(afterAction);
    });

    it("should preserve all other state when updating timestamp", () => {
      const state: UISliceState = {
        ...initialUIState,
        theme: "dark",
        sfxMuted: true,
        logs: [{ turn: 1, scope: "public", message: "test" }],
        chat: [{ id: "1", senderIndex: 0, senderName: "P1", text: "Hi", timestamp: 1000 }],
        investigateResult: { targetName: "Player1", role: "Duke" },
        eliminationToast: { playerIndex: 0, playerName: "P1", reason: "Out", turn: 1, id: "e1" },
        turnTimer: { activePlayerIndex: 0, durationMs: 30000, running: true, paused: false, key: "t1" },
        replayHistory: [{ matchId: "m1", lobbyId: "l1", playerId: "p1", playerNames: ["A"], participantIds: ["a"], endedAt: 1000 }],
      };

      const action = uiActions.updateTimestamp();
      const newState = uiReducer(state, action);

      expect(newState.theme).toBe("dark");
      expect(newState.sfxMuted).toBe(true);
      expect(newState.logs).toHaveLength(1);
      expect(newState.chat).toHaveLength(1);
      expect(newState.investigateResult).not.toBeNull();
      expect(newState.eliminationToast).not.toBeNull();
      expect(newState.turnTimer).not.toBeNull();
      expect(newState.replayHistory).toHaveLength(1);
    });
  });

  describe("RESET action", () => {
    it("should reset to initial state when preservePreferences is false", () => {
      const state: UISliceState = {
        sfxMuted: true,
        theme: "dark",
        logs: [{ turn: 1, scope: "public", message: "test" }],
        chat: [{ id: "1", senderIndex: 0, senderName: "P1", text: "Hi", timestamp: 1000 }],
        investigateResult: { targetName: "Player1", role: "Duke" },
        eliminationToast: { playerIndex: 0, playerName: "P1", reason: "Out", turn: 1, id: "e1" },
        turnTimer: { activePlayerIndex: 0, durationMs: 30000, running: true, paused: false, key: "t1" },
        replayHistory: [{ matchId: "m1", lobbyId: "l1", playerId: "p1", playerNames: ["A"], participantIds: ["a"], endedAt: 1000 }],
        lastUpdateTs: 9999,
        mockScenario: "test-scenario",
      };

      const action = uiActions.reset(false);
      const newState = uiReducer(state, action);

      expect(newState).toEqual(initialUIState);
    });

    it("should preserve preferences when preservePreferences is true", () => {
      const state: UISliceState = {
        ...initialUIState,
        sfxMuted: true,
        theme: "dark",
        logs: [{ turn: 1, scope: "public", message: "test" }],
        chat: [{ id: "1", senderIndex: 0, senderName: "P1", text: "Hi", timestamp: 1000 }],
      };

      const action = uiActions.reset(true);
      const newState = uiReducer(state, action);

      expect(newState.sfxMuted).toBe(true);
      expect(newState.theme).toBe("dark");
      expect(newState.logs).toEqual([]);
      expect(newState.chat).toEqual([]);
      expect(newState.investigateResult).toBeNull();
      expect(newState.eliminationToast).toBeNull();
      expect(newState.turnTimer).toBeNull();
      expect(newState.replayHistory).toEqual([]);
    });

    it("should reset from initial state with preservePreferences false", () => {
      const action = uiActions.reset(false);
      const newState = uiReducer(initialUIState, action);

      expect(newState).toEqual(initialUIState);
    });

    it("should reset from initial state with preservePreferences true", () => {
      const action = uiActions.reset(true);
      const newState = uiReducer(initialUIState, action);

      expect(newState).toEqual(initialUIState);
    });
  });

  describe("Unknown actions", () => {
    it("should return current state for unknown action type", () => {
      const state: UISliceState = {
        ...initialUIState,
        theme: "dark",
        sfxMuted: true,
      };
      const action = { type: "UNKNOWN_ACTION" } as unknown as UIAction;
      const newState = uiReducer(state, action);

      expect(newState).toEqual(state);
    });

    it("should handle random action gracefully", () => {
      const state: UISliceState = {
        ...initialUIState,
        logs: [{ turn: 1, scope: "public", message: "test" }],
      };
      const action = { type: "RANDOM_TYPE" } as unknown as UIAction;
      const newState = uiReducer(state, action);

      expect(newState).toEqual(state);
    });
  });

  describe("State Transitions", () => {
    it("should handle full game lifecycle with logs and chat", () => {
      let state = initialUIState;

      // Game starts
      state = uiReducer(state, uiActions.setTheme("dark"));
      state = uiReducer(state, uiActions.gameLog({ turn: 1, scope: "public", message: "Game started" }));
      state = uiReducer(state, uiActions.chatMessage({ id: "c1", senderIndex: 0, senderName: "P1", text: "GL HF", timestamp: 1000 }));

      expect(state.theme).toBe("dark");
      expect(state.logs).toHaveLength(1);
      expect(state.chat).toHaveLength(1);

      // Investigation happens
      state = uiReducer(state, uiActions.investigateResult({ targetName: "P2", role: "Assassin" }));
      expect(state.investigateResult).not.toBeNull();

      // Clear investigation
      state = uiReducer(state, uiActions.clearInvestigate());
      expect(state.investigateResult).toBeNull();

      // Player eliminated
      state = uiReducer(state, uiActions.elimination({ playerIndex: 1, playerName: "P2", reason: "Coup", turn: 5, id: "e1" }));
      expect(state.eliminationToast).not.toBeNull();

      // Clear elimination toast
      state = uiReducer(state, uiActions.clearEliminationToast());
      expect(state.eliminationToast).toBeNull();

      // Reset with preserved preferences
      state = uiReducer(state, uiActions.reset(true));
      expect(state.theme).toBe("dark");
      expect(state.sfxMuted).toBe(false);
      expect(state.logs).toHaveLength(0);
      expect(state.chat).toHaveLength(0);
    });

    it("should handle investigation cycle", () => {
      let state = initialUIState;

      // First investigation
      state = uiReducer(state, uiActions.investigateResult({ targetName: "Player1", role: "Duke" }));
      expect(state.investigateResult?.targetName).toBe("Player1");

      // Clear
      state = uiReducer(state, uiActions.clearInvestigate());
      expect(state.investigateResult).toBeNull();

      // Second investigation
      state = uiReducer(state, uiActions.investigateResult({ targetName: "Player2", role: "Captain" }));
      expect(state.investigateResult?.targetName).toBe("Player2");

      // Clear again
      state = uiReducer(state, uiActions.clearInvestigate());
      expect(state.investigateResult).toBeNull();
    });

    it("should handle multiple eliminations", () => {
      let state = initialUIState;

      state = uiReducer(state, uiActions.elimination({ playerIndex: 1, playerName: "P2", reason: "Coup", turn: 3, id: "e1" }));
      expect(state.eliminationToast?.playerName).toBe("P2");

      state = uiReducer(state, uiActions.clearEliminationToast());
      expect(state.eliminationToast).toBeNull();

      state = uiReducer(state, uiActions.elimination({ playerIndex: 2, playerName: "P3", reason: "Challenge", turn: 5, id: "e2" }));
      expect(state.eliminationToast?.playerName).toBe("P3");
    });

    it("should handle turn timer transitions", () => {
      let state = initialUIState;

      // Timer starts
      state = uiReducer(state, uiActions.turnTimer({ activePlayerIndex: 0, durationMs: 30000, running: true, paused: false, key: "t1" }));
      expect(state.turnTimer?.running).toBe(true);

      // Timer paused
      state = uiReducer(state, uiActions.turnTimer({ activePlayerIndex: 0, durationMs: 30000, running: false, paused: true, key: "t1" }));
      expect(state.turnTimer?.paused).toBe(true);

      // Timer cleared
      state = uiReducer(state, uiActions.turnTimer(null));
      expect(state.turnTimer).toBeNull();

      // New turn timer
      state = uiReducer(state, uiActions.turnTimer({ activePlayerIndex: 1, durationMs: 45000, running: true, paused: false, key: "t2" }));
      expect(state.turnTimer?.activePlayerIndex).toBe(1);
    });
  });

  describe("Action Creators", () => {
    it("should create SET_SFX_MUTED action", () => {
      const action = uiActions.setSfxMuted(true);
      expect(action).toEqual({ type: "SET_SFX_MUTED", muted: true });
    });

    it("should create SET_THEME action", () => {
      const action = uiActions.setTheme("dark");
      expect(action).toEqual({ type: "SET_THEME", theme: "dark" });
    });

    it("should create GAME_LOG action", () => {
      const entry: LogEntry = { turn: 1, scope: "public", message: "Test" };
      const action = uiActions.gameLog(entry);
      expect(action).toEqual({ type: "GAME_LOG", entry });
    });

    it("should create CHAT_MESSAGE action", () => {
      const message: ChatMessage = { id: "1", senderIndex: 0, senderName: "P1", text: "Hi", timestamp: 1000 };
      const action = uiActions.chatMessage(message);
      expect(action).toEqual({ type: "CHAT_MESSAGE", payload: message });
    });

    it("should create INVESTIGATE_RESULT action", () => {
      const result: InvestigateResult = { targetName: "P1", role: "Duke" };
      const action = uiActions.investigateResult(result);
      expect(action).toEqual({ type: "INVESTIGATE_RESULT", payload: result });
    });

    it("should create CLEAR_INVESTIGATE action", () => {
      const action = uiActions.clearInvestigate();
      expect(action).toEqual({ type: "CLEAR_INVESTIGATE" });
    });

    it("should create ELIMINATION action", () => {
      const toast: EliminationToast = { playerIndex: 0, playerName: "P1", reason: "Out", turn: 1, id: "e1" };
      const action = uiActions.elimination(toast);
      expect(action).toEqual({ type: "ELIMINATION", toast });
    });

    it("should create CLEAR_ELIMINATION_TOAST action", () => {
      const action = uiActions.clearEliminationToast();
      expect(action).toEqual({ type: "CLEAR_ELIMINATION_TOAST" });
    });

    it("should create TURN_TIMER action", () => {
      const timer: TurnTimerState = { activePlayerIndex: 0, durationMs: 30000, running: true, paused: false, key: "t1" };
      const action = uiActions.turnTimer(timer);
      expect(action).toEqual({ type: "TURN_TIMER", timer });
    });

    it("should create TURN_TIMER action with null", () => {
      const action = uiActions.turnTimer(null);
      expect(action).toEqual({ type: "TURN_TIMER", timer: null });
    });

    it("should create ADD_REPLAY action", () => {
      const entry: ReplayEntry = { matchId: "m1", lobbyId: "l1", playerId: "p1", playerNames: ["A"], participantIds: ["a"], endedAt: 1000 };
      const action = uiActions.addReplay(entry);
      expect(action).toEqual({ type: "ADD_REPLAY", entry });
    });

    it("should create UPDATE_TIMESTAMP action", () => {
      const action = uiActions.updateTimestamp();
      expect(action).toEqual({ type: "UPDATE_TIMESTAMP" });
    });

    it("should create RESET action with preservePreferences true", () => {
      const action = uiActions.reset(true);
      expect(action).toEqual({ type: "RESET", preservePreferences: true });
    });

    it("should create RESET action with preservePreferences false", () => {
      const action = uiActions.reset(false);
      expect(action).toEqual({ type: "RESET", preservePreferences: false });
    });
  });

  describe("Edge Cases", () => {
    it("should handle empty log entry arrays", () => {
      const action = uiActions.gameLog({ turn: 1, scope: "public", message: "" });
      const newState = uiReducer(initialUIState, action);

      expect(newState.logs).toHaveLength(1);
      expect(newState.logs[0].message).toBe("");
    });

    it("should handle empty chat message", () => {
      const message: ChatMessage = { id: "1", senderIndex: 0, senderName: "", text: "", timestamp: 0 };
      const action = uiActions.chatMessage(message);
      const newState = uiReducer(initialUIState, action);

      expect(newState.chat).toHaveLength(1);
      expect(newState.chat[0].text).toBe("");
    });

    it("should handle multiple sequential theme changes", () => {
      let state = initialUIState;

      state = uiReducer(state, uiActions.setTheme("dark"));
      expect(state.theme).toBe("dark");

      state = uiReducer(state, uiActions.setTheme("light"));
      expect(state.theme).toBe("light");

      state = uiReducer(state, uiActions.setTheme("dark"));
      expect(state.theme).toBe("dark");
    });

    it("should handle rapid timestamp updates", () => {
      let state = initialUIState;
      const timestamps: number[] = [];

      for (let i = 0; i < 5; i++) {
        state = uiReducer(state, uiActions.updateTimestamp());
        timestamps.push(state.lastUpdateTs);
      }

      // Each timestamp should be greater than or equal to the previous
      for (let i = 1; i < timestamps.length; i++) {
        expect(timestamps[i]).toBeGreaterThanOrEqual(timestamps[i - 1]);
      }
    });

    it("should handle rapid sfxMuted toggles", () => {
      let state = initialUIState;

      for (let i = 0; i < 10; i++) {
        state = uiReducer(state, uiActions.setSfxMuted(i % 2 === 0));
      }

      expect(state.sfxMuted).toBe(false); // 9th index (even) -> false
    });

    it("should handle replay deduplication with many entries", () => {
      let state = initialUIState;

      // Add 5 entries
      for (let i = 0; i < 5; i++) {
        state = uiReducer(
          state,
          uiActions.addReplay({
            matchId: `match-${i}`,
            lobbyId: `lobby-${i}`,
            playerId: `player-${i}`,
            playerNames: [`Player${i}`],
            participantIds: [`p${i}`],
            endedAt: i * 1000,
          })
        );
      }

      // Duplicate match-2 (should move to front)
      state = uiReducer(
        state,
        uiActions.addReplay({
          matchId: "match-2",
          lobbyId: "lobby-2",
          playerId: "player-2",
          playerNames: ["UpdatedPlayer"],
          participantIds: ["p2"],
          endedAt: 9999,
        })
      );

      expect(state.replayHistory).toHaveLength(5);
      expect(state.replayHistory[0].matchId).toBe("match-2");
      expect(state.replayHistory[0].playerNames[0]).toBe("UpdatedPlayer");
    });

    it("should maintain state reference integrity", () => {
      const state: UISliceState = {
        ...initialUIState,
        theme: "dark",
      };

      const action = uiActions.setTheme("light");
      const newState = uiReducer(state, action);

      // New state should be a different object
      expect(newState).not.toBe(state);
    });

    it("should not mutate original state", () => {
      const originalState: UISliceState = { ...initialUIState };
      const stateCopy = { ...originalState };

      const action = uiActions.setTheme("dark");
      uiReducer(originalState, action);

      // Original state should remain unchanged
      expect(originalState).toEqual(stateCopy);
    });

    it("should not mutate arrays in original state", () => {
      const originalLogs: LogEntry[] = [{ turn: 1, scope: "public", message: "test" }];
      const originalState: UISliceState = {
        ...initialUIState,
        logs: originalLogs,
      };

      const action = uiActions.gameLog({ turn: 2, scope: "private", message: "new" });
      uiReducer(originalState, action);

      // Original logs array should remain unchanged
      expect(originalLogs).toHaveLength(1);
      expect(originalLogs[0].message).toBe("test");
    });

    it("should handle empty chat array correctly", () => {
      const message: ChatMessage = { id: "1", senderIndex: 0, senderName: "P1", text: "First", timestamp: 1000 };
      const action = uiActions.chatMessage(message);
      const newState = uiReducer(initialUIState, action);

      expect(newState.chat).toHaveLength(1);
      expect(newState.chat[0].text).toBe("First");
    });

    it("should handle empty replay history correctly", () => {
      const entry: ReplayEntry = { matchId: "m1", lobbyId: "l1", playerId: "p1", playerNames: ["A"], participantIds: ["a"], endedAt: 1000 };
      const action = uiActions.addReplay(entry);
      const newState = uiReducer(initialUIState, action);

      expect(newState.replayHistory).toHaveLength(1);
      expect(newState.replayHistory[0].matchId).toBe("m1");
    });

    it("should handle clearing investigate when already null", () => {
      const action = uiActions.clearInvestigate();
      const newState = uiReducer(initialUIState, action);

      expect(newState.investigateResult).toBeNull();
    });

    it("should handle clearing elimination when already null", () => {
      const action = uiActions.clearEliminationToast();
      const newState = uiReducer(initialUIState, action);

      expect(newState.eliminationToast).toBeNull();
    });

    it("should handle setting turn timer to null when already null", () => {
      const action = uiActions.turnTimer(null);
      const newState = uiReducer(initialUIState, action);

      expect(newState.turnTimer).toBeNull();
    });
  });
});
