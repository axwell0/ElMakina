/**
 * Centralized storage key definitions
 * 
 * All localStorage keys used across the application are defined here
 * to prevent collisions and provide a single source of truth.
 * 
 * Keys are organized by domain/feature for clarity.
 */

const STORAGE_PREFIX = "elmakina" as const;

function createKey(...parts: string[]): string {
  return [STORAGE_PREFIX, ...parts].join(".");
}

export const STORAGE_KEYS = {
  // Connection/Identity
  reconnectToken: createKey("reconnectToken"),
  nickname: createKey("nickname"),
  playerId: createKey("playerId"),
  avatar: createKey("avatar"),
  connectionLog: createKey("connectionLog"),

  // UI Preferences
  theme: createKey("theme"),
  sfxMuted: createKey("sfxMuted"),

  // Game Data
  replayHistory: createKey("replayHistory"),
} as const;

export type StorageKey = typeof STORAGE_KEYS[keyof typeof STORAGE_KEYS];
