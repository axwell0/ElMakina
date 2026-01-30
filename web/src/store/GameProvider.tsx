import React, {useEffect, useReducer} from 'react';
import {socket} from '../network/socket';
import {GameContext} from './gameContext';
import { localStorageAdapter, STORAGE_KEYS } from '@/state/persistence';
import { rootReducer, initialSlicedState, toGameState, type SlicedGameState } from '@/state/slices';

function loadPersistedState(init: SlicedGameState): SlicedGameState {
    if (!localStorageAdapter.isAvailable()) return init;

    const storedMute = localStorageAdapter.getItem(STORAGE_KEYS.sfxMuted);
    const storedTheme = localStorageAdapter.getItem(STORAGE_KEYS.theme) as "light" | "dark" | null;
    const storedReplays = localStorageAdapter.getItem(STORAGE_KEYS.replayHistory);

    const next: SlicedGameState = { ...init };
    
    if (storedMute !== null) {
        next.ui = { ...next.ui, sfxMuted: storedMute === "true" };
    }
    if (storedTheme !== null) {
        next.ui = { ...next.ui, theme: storedTheme };
    } else {
        // Default to dark for the signature tabletop feel
        next.ui = { ...next.ui, theme: "dark" };
    }
    if (storedReplays) {
        try {
            const parsed = JSON.parse(storedReplays);
            if (Array.isArray(parsed)) {
                next.ui = { ...next.ui, replayHistory: parsed };
            }
        } catch {
            next.ui = { ...next.ui, replayHistory: [] };
        }
    }
    return next;
}

export const GameProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const [slicedState, dispatch] = useReducer(rootReducer, initialSlicedState, loadPersistedState);
    
    // Convert sliced state to flat GameState for backwards compatibility
    const state = toGameState(slicedState);
    const stateRef = React.useRef(state);
    
    useEffect(() => {
        stateRef.current = state;
    }, [state]);

    useEffect(() => {
        socket.setConnectionHandlers(
            () => dispatch({ type: "CONNECT" }),
            () => dispatch({ type: "DISCONNECT" })
        );

        const unsubscribe = socket.onMessage((envelope) => {
            dispatch({ type: "MESSAGE", envelope });
        });

        socket.connect();

        return () => {
            unsubscribe();
            socket.disconnect();
        };
    }, []);

    useEffect(() => {
        if (typeof window === "undefined") return;
        const handleOnline = () => {
            if (!socket.isOpen()) {
                socket.reconnectNow();
            }
        };
        const handleVisibility = () => {
            if (document.visibilityState === "visible" && !socket.isOpen()) {
                socket.reconnectNow();
            }
        };
        window.addEventListener("online", handleOnline);
        document.addEventListener("visibilitychange", handleVisibility);
        return () => {
            window.removeEventListener("online", handleOnline);
            document.removeEventListener("visibilitychange", handleVisibility);
        };
    }, []);

    // Persist state changes to storage
    useEffect(() => {
        localStorageAdapter.setItem(STORAGE_KEYS.sfxMuted, String(state.sfxMuted));
    }, [state.sfxMuted]);

    useEffect(() => {
        localStorageAdapter.setItem(STORAGE_KEYS.theme, state.theme);
    }, [state.theme]);

    useEffect(() => {
        try {
            const serialized = JSON.stringify(state.replayHistory);
            localStorageAdapter.setItem(STORAGE_KEYS.replayHistory, serialized);
        } catch (error) {
            console.warn("Failed to persist replay history:", error);
        }
    }, [state.replayHistory]);

    useEffect(() => {
        if (typeof window === "undefined") return;
        if (process.env.NODE_ENV === "production") return;
        window.__ELMAKINA_DEV__ = {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            dispatch: dispatch as any,
            getState: () => stateRef.current,
            sendEnvelope: (envelope) => dispatch({ type: "MESSAGE", envelope }),
            pauseGame: (payload) =>
                dispatch({
                    type: "MESSAGE",
                    envelope: {
                        type: "game_paused",
                        payload: {
                            paused_by_player_id: payload.paused_by_player_id ?? "",
                            paused_by_index: payload.paused_by_index ?? 0,
                            paused_by_name: payload.paused_by_name ?? "Unknown",
                            deadline_ms: payload.deadline_ms ?? Date.now() + 60000,
                            duration_ms: payload.duration_ms ?? 60000,
                            pause_reason: payload.pause_reason ?? "disconnect",
                            eligible_voters: payload.eligible_voters ?? [],
                            kick_votes: payload.kick_votes ?? [],
                        },
                    },
                }),
            updateKickVotes: (payload) =>
                dispatch({
                    type: "MESSAGE",
                    envelope: {
                        type: "kick_vote_update",
                        payload: {
                            eligible_voters: payload.eligible_voters ?? [],
                            kick_votes: payload.kick_votes ?? [],
                        },
                    },
                }),
            resumeGame: () =>
                dispatch({
                    type: "MESSAGE",
                    envelope: {
                        type: "game_resumed",
                        payload: {
                            resumed_by_player_id: "dev",
                            resumed_by_index: 0,
                            resumed_by_name: "Developer",
                            resume_reason: "reconnected",
                        },
                    },
                }),
            connectionLog: () => socket.getConnectionLog(),
        };
    }, [dispatch]);

    return (
        <GameContext.Provider value={{ state, dispatch }}>
            {children}
        </GameContext.Provider>
    );
};
