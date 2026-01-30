/**
 * Typed storage wrapper
 * 
 * Provides type-safe storage operations with automatic
 * serialization/deserialization.
 */

import type { StorageAdapter, StorageSerializer, TypedStorage } from "./types";

type StorageKey = string;

/**
 * Create a typed storage wrapper for a specific data type
 * 
 * @example
 * ```typescript
 * const userStorage = createTypedStorage(
 *   storageAdapter,
 *   "user",
 *   {
 *     parse: (v) => JSON.parse(v) as User,
 *     stringify: (v) => JSON.stringify(v)
 *   }
 * );
 * 
 * userStorage.set({ id: 1, name: "Alice" });
 * const user = userStorage.get();
 * ```
 */
export function createTypedStorage<T>(
  adapter: StorageAdapter,
  key: StorageKey,
  serializer: StorageSerializer<T>
): TypedStorage<T> {
  return {
    get(): T | null {
      const raw = adapter.getItem(key);
      if (raw === null) return null;
      
      try {
        return serializer.parse(raw);
      } catch (error) {
        console.warn(`[TypedStorage] Failed to parse value for "${key}":`, error);
        return null;
      }
    },

    set(value: T | null): void {
      if (value === null) {
        adapter.removeItem(key);
        return;
      }
      
      try {
        const serialized = serializer.stringify(value);
        adapter.setItem(key, serialized);
      } catch (error) {
        console.warn(`[TypedStorage] Failed to serialize value for "${key}":`, error);
      }
    }
  };
}

/**
 * JSON serializer for common use cases
 */
export const jsonSerializer = <T>(): StorageSerializer<T> => ({
  parse: (value: string): T | null => {
    try {
      return JSON.parse(value) as T;
    } catch {
      return null;
    }
  },
  stringify: (value: T): string => JSON.stringify(value)
});

/**
 * Boolean serializer
 */
export const booleanSerializer = (): StorageSerializer<boolean> => ({
  parse: (value: string): boolean | null => {
    if (value === "true") return true;
    if (value === "false") return false;
    return null;
  },
  stringify: (value: boolean): string => String(value)
});
