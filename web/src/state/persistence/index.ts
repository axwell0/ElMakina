/**
 * Storage persistence layer
 * 
 * Provides a clean abstraction over browser storage with:
 * - Type-safe operations
 * - SSR safety
 * - React integration hooks
 * - Easy testability (inject mock adapters)
 */

export type {
  StorageAdapter,
  StorageKey,
  StorageValue,
  TypedStorage,
  StorageSerializer,
} from "./types";

export { STORAGE_KEYS } from "./keys";

export {
  LocalStorageAdapter,
  localStorageAdapter,
} from "./local-storage";

export {
  createTypedStorage,
  jsonSerializer,
  booleanSerializer,
} from "./typed-storage";
