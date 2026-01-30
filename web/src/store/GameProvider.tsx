import React, {useEffect, useReducer} from 'react';
import {initialGameState} from './types';
import {gameReducer} from './gameReducer';
import {socket} from '../network/socket';
import {GameContext} from './gameContext';

export const GameProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const stateRef = React.useRef(initialGameState);
    const [state, dispatch] = useReducer(gameReducer, initialGameState, (init) => {
        if (typeof window === "undefined") return init;

        const storedMute = localStorage.getItem("elmakina.sfxMuted");
        const storedTheme = localStorage.getItem("elmakina.theme") as "light" | "dark" | null;
        const storedReplays = localStorage.getItem("elmakina.replayHistory");

        const next = { ...init };
        if (storedMute !== null) {
            next.sfxMuted = storedMute === "true";
        }
        if (storedTheme !== null) {
            next.theme = storedTheme;
        } else {
            // Default to dark for the signature tabletop feel
            next.theme = "dark";
        }
        if (storedReplays) {
            try {
                const parsed = JSON.parse(storedReplays);
                if (Array.isArray(parsed)) {
                    next.replayHistory = parsed;
                }
            } catch {
                next.replayHistory = [];
            }
        }
        return next;
    });

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

    useEffect(() => {
        if (typeof window !== "undefined") {
            localStorage.setItem("elmakina.sfxMuted", String(state.sfxMuted));
        }
    }, [state.sfxMuted]);

    useEffect(() => {
        if (typeof window !== "undefined") {
            localStorage.setItem("elmakina.theme", state.theme);
        }
    }, [state.theme]);

    useEffect(() => {
        if (typeof window !== "undefined") {
            localStorage.setItem("elmakina.replayHistory", JSON.stringify(state.replayHistory));
        }
    }, [state.replayHistory]);

    useEffect(() => {
        if (typeof window === "undefined") return;
        if (process.env.NODE_ENV === "production") return;
        window.__ELMAKINA_DEV__ = {
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
