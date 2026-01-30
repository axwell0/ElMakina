/**
 * LocalStorage adapter implementation
 * 
 * Wraps browser localStorage with the StorageAdapter interface,
 * providing SSR safety and graceful error handling.
 */

import type { StorageAdapter, StorageKey, StorageValue } from "./types";

export class LocalStorageAdapter implements StorageAdapter {
  private isClient: boolean;

  constructor() {
    this.isClient = typeof window !== "undefined";
  }

  isAvailable(): boolean {
    if (!this.isClient) return false;
    
    try {
      const test = "__storage_test__";
      localStorage.setItem(test, test);
      localStorage.removeItem(test);
      return true;
    } catch {
      return false;
    }
  }

  getItem(key: StorageKey): StorageValue {
    if (!this.isClient) return null;
    
    try {
      return localStorage.getItem(key);
    } catch (error) {
      console.warn(`[Storage] Failed to get item "${key}":`, error);
      return null;
    }
  }

  setItem(key: StorageKey, value: StorageValue): void {
    if (!this.isClient) return;
    
    try {
      if (value === null) {
        localStorage.removeItem(key);
      } else {
        localStorage.setItem(key, value);
      }
    } catch (error) {
      console.warn(`[Storage] Failed to set item "${key}":`, error);
    }
  }

  removeItem(key: StorageKey): void {
    if (!this.isClient) return;
    
    try {
      localStorage.removeItem(key);
    } catch (error) {
      console.warn(`[Storage] Failed to remove item "${key}":`, error);
    }
  }
}

/**
 * Singleton instance for convenience
 * Use createLocalStorage() for testing or custom instances
 */
export const localStorageAdapter = new LocalStorageAdapter();
