/**
 * State management hooks
 * 
 * React hooks for working with the centralized state and storage.
 */

export {
  useStorageSync,
  useBooleanStorageSync,
  useJsonStorageSync,
  useTypedStorageSync,
} from "./useStorageSync";

export {
  useGameSlice,
  usePause,
  useIdentity,
  useIsConnected,
} from "./useGameSlice";
