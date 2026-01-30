/**
 * Storage abstraction layer types
 * 
 * Provides type-safe interface for persistence operations,
 * allowing easy swapping between localStorage, sessionStorage,
 * or other storage backends.
 */

export type StorageKey = string;

export type StorageValue = string | null;

export interface StorageAdapter {
  /**
   * Retrieve value from storage
   * @param key - Storage key
   * @returns Stored value or null if not found
   */
  getItem(key: StorageKey): StorageValue;

  /**
   * Store value in storage
   * @param key - Storage key
   * @param value - Value to store (null to remove)
   */
  setItem(key: StorageKey, value: StorageValue): void;

  /**
   * Remove value from storage
   * @param key - Storage key
   */
  removeItem(key: StorageKey): void;

  /**
   * Check if storage is available (not SSR, not disabled)
   */
  isAvailable(): boolean;
}

export interface TypedStorage<T> {
  /**
   * Retrieve and parse typed value
   * @returns Parsed value or null if not found/invalid
   */
  get(): T | null;

  /**
   * Serialize and store typed value
   * @param value - Value to store (null to remove)
   */
  set(value: T | null): void;
}

export type StorageSerializer<T> = {
  parse: (value: string) => T | null;
  stringify: (value: T) => string;
};
