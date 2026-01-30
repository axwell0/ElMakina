/**
 * React hooks for storage synchronization
 * 
 * Provides hooks to keep React state synchronized with persistent storage.
 * These replace inline useEffect calls in components.
 */

import { useState, useCallback } from "react";
import type { StorageAdapter } from "../persistence/types";
import type { TypedStorage } from "../persistence/types";

/**
 * Hook to sync a state value with storage
 * 
 * @example
 * ```typescript
 * const [theme, setTheme] = useStorageSync(
 *   storageAdapter,
 *   STORAGE_KEYS.theme,
 *   "light"
 * );
 * ```
 */
export function useStorageSync(
  adapter: StorageAdapter,
  key: string,
  defaultValue: string
): [string, (value: string) => void] {
  const [value, setValue] = useState<string>(() => {
    if (!adapter.isAvailable()) return defaultValue;
    return adapter.getItem(key) ?? defaultValue;
  });

  const setStoredValue = useCallback(
    (newValue: string) => {
      setValue(newValue);
      adapter.setItem(key, newValue);
    },
    [adapter, key]
  );

  return [value, setStoredValue];
}

/**
 * Hook to sync boolean state with storage
 * 
 * @example
 * ```typescript
 * const [sfxMuted, setSfxMuted] = useBooleanStorageSync(
 *   storageAdapter,
 *   STORAGE_KEYS.sfxMuted,
 *   false
 * );
 * ```
 */
export function useBooleanStorageSync(
  adapter: StorageAdapter,
  key: string,
  defaultValue: boolean
): [boolean, (value: boolean) => void] {
  const [value, setValue] = useState<boolean>(() => {
    if (!adapter.isAvailable()) return defaultValue;
    const stored = adapter.getItem(key);
    if (stored === null) return defaultValue;
    return stored === "true";
  });

  const setStoredValue = useCallback(
    (newValue: boolean) => {
      setValue(newValue);
      adapter.setItem(key, String(newValue));
    },
    [adapter, key]
  );

  return [value, setStoredValue];
}

/**
 * Hook to sync typed JSON state with storage
 * 
 * @example
 * ```typescript
 * const [replays, setReplays] = useJsonStorageSync<ReplayEntry[]>(
 *   storageAdapter,
 *   STORAGE_KEYS.replayHistory,
 *   []
 * );
 * ```
 */
export function useJsonStorageSync<T>(
  adapter: StorageAdapter,
  key: string,
  defaultValue: T
): [T, (value: T) => void] {
  const [value, setValue] = useState<T>(() => {
    if (!adapter.isAvailable()) return defaultValue;
    const stored = adapter.getItem(key);
    if (stored === null) return defaultValue;
    
    try {
      return JSON.parse(stored) as T;
    } catch {
      return defaultValue;
    }
  });

  const setStoredValue = useCallback(
    (newValue: T) => {
      setValue(newValue);
      try {
        const serialized = JSON.stringify(newValue);
        adapter.setItem(key, serialized);
      } catch (error) {
        console.warn(`[useJsonStorageSync] Failed to serialize value for "${key}":`, error);
      }
    },
    [adapter, key]
  );

  return [value, setStoredValue];
}

/**
 * Hook to sync state with typed storage wrapper
 * More type-safe alternative to useJsonStorageSync
 * 
 * @example
 * ```typescript
 * const replayStorage = createTypedStorage(
 *   adapter,
 *   STORAGE_KEYS.replayHistory,
 *   jsonSerializer<ReplayEntry[]>()
 * );
 * const [replays, setReplays] = useTypedStorageSync(replayStorage, []);
 * ```
 */
export function useTypedStorageSync<T>(
  typedStorage: TypedStorage<T>,
  defaultValue: T
): [T, (value: T) => void] {
  const [value, setValue] = useState<T>(() => {
    return typedStorage.get() ?? defaultValue;
  });

  const setStoredValue = useCallback(
    (newValue: T) => {
      setValue(newValue);
      typedStorage.set(newValue);
    },
    [typedStorage]
  );

  return [value, setStoredValue];
}
