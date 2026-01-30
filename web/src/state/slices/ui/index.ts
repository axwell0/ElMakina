/**
 * UI state slice
 *
 * Manages UI preferences, logs, chat, and replay history.
 */

import type {
  ChatMessage,
  EliminationToast,
  InvestigateResult,
  LogEntry,
  ReplayEntry,
  TurnTimerState,
} from "@/state/types";

export interface UISliceState {
  // Preferences
  sfxMuted: boolean;
  theme: "light" | "dark";

  // Logs and chat
  logs: LogEntry[];
  chat: ChatMessage[];

  // Transient UI state
  investigateResult: InvestigateResult | null;
  eliminationToast: EliminationToast | null;
  turnTimer: TurnTimerState | null;

  // Replay
  replayHistory: ReplayEntry[];

  // Meta
  lastUpdateTs: number;
  mockScenario?: string;
}

export const initialUIState: UISliceState = {
  sfxMuted: false,
  theme: "light",
  logs: [],
  chat: [],
  investigateResult: null,
  eliminationToast: null,
  turnTimer: null,
  replayHistory: [],
  lastUpdateTs: 0,
  mockScenario: undefined,
};

export type UIAction =
  | { type: "SET_SFX_MUTED"; muted: boolean }
  | { type: "SET_THEME"; theme: "light" | "dark" }
  | { type: "GAME_LOG"; entry: LogEntry }
  | { type: "CHAT_MESSAGE"; payload: { id: string; senderIndex: number; senderName: string; text: string; timestamp: number } }
  | { type: "INVESTIGATE_RESULT"; payload: { targetName: string; role: string } }
  | { type: "CLEAR_INVESTIGATE" }
  | { type: "ELIMINATION"; toast: EliminationToast }
  | { type: "CLEAR_ELIMINATION_TOAST" }
  | { type: "TURN_TIMER"; timer: TurnTimerState | null }
  | { type: "ADD_REPLAY"; entry: ReplayEntry }
  | { type: "UPDATE_TIMESTAMP" }
  | { type: "RESET"; preservePreferences: boolean };

export function uiReducer(
  state: UISliceState,
  action: UIAction
): UISliceState {
  switch (action.type) {
    case "SET_SFX_MUTED":
      return { ...state, sfxMuted: action.muted };

    case "SET_THEME":
      return { ...state, theme: action.theme };

    case "GAME_LOG":
      return {
        ...state,
        logs: [...state.logs, action.entry],
        lastUpdateTs: Date.now(),
      };

    case "CHAT_MESSAGE":
      return {
        ...state,
        chat: [...state.chat, action.payload].slice(-100), // Keep last 100 messages
        lastUpdateTs: Date.now(),
      };

    case "INVESTIGATE_RESULT":
      return { ...state, investigateResult: action.payload };

    case "CLEAR_INVESTIGATE":
      return { ...state, investigateResult: null };

    case "ELIMINATION":
      return { ...state, eliminationToast: action.toast };

    case "CLEAR_ELIMINATION_TOAST":
      return { ...state, eliminationToast: null };

    case "TURN_TIMER":
      return { ...state, turnTimer: action.timer };

    case "ADD_REPLAY": {
      // Add new entry and deduplicate by matchId
      const deduped = state.replayHistory.filter(
        (item) => item.matchId !== action.entry.matchId
      );
      return {
        ...state,
        replayHistory: [action.entry, ...deduped].slice(0, 30), // Keep last 30
      };
    }

    case "UPDATE_TIMESTAMP":
      return { ...state, lastUpdateTs: Date.now() };

    case "RESET": {
      if (action.preservePreferences) {
        return {
          ...initialUIState,
          sfxMuted: state.sfxMuted,
          theme: state.theme,
        };
      }
      return initialUIState;
    }

    default:
      return state;
  }
}

// Action creators
export const uiActions = {
  setSfxMuted: (muted: boolean): UIAction => ({
    type: "SET_SFX_MUTED",
    muted,
  }),
  setTheme: (theme: "light" | "dark"): UIAction => ({
    type: "SET_THEME",
    theme,
  }),
  gameLog: (entry: LogEntry): UIAction => ({
    type: "GAME_LOG",
    entry,
  }),
  chatMessage: (message: { id: string; senderIndex: number; senderName: string; text: string; timestamp: number }): UIAction => ({
    type: "CHAT_MESSAGE",
    payload: message,
  }),
  investigateResult: (result: { targetName: string; role: string }): UIAction => ({
    type: "INVESTIGATE_RESULT",
    payload: result,
  }),
  clearInvestigate: (): UIAction => ({ type: "CLEAR_INVESTIGATE" }),
  elimination: (toast: EliminationToast): UIAction => ({
    type: "ELIMINATION",
    toast,
  }),
  clearEliminationToast: (): UIAction => ({ type: "CLEAR_ELIMINATION_TOAST" }),
  turnTimer: (timer: TurnTimerState | null): UIAction => ({
    type: "TURN_TIMER",
    timer,
  }),
  addReplay: (entry: ReplayEntry): UIAction => ({
    type: "ADD_REPLAY",
    entry,
  }),
  updateTimestamp: (): UIAction => ({ type: "UPDATE_TIMESTAMP" }),
  reset: (preservePreferences: boolean): UIAction => ({
    type: "RESET",
    preservePreferences,
  }),
} as const;
