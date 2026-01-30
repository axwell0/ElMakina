import React, {useEffect, useReducer} from 'react';
import {GameContext} from '@/store/gameContext';
import {gameReducer} from '@/store/gameReducer';
import {getMockState, type MockScenario} from './mockState';
import {socket} from '@/network/socket';

export const MockGameProvider: React.FC<{ children: React.ReactNode; scenario?: MockScenario }> = ({
    children,
    scenario = 'game',
}) => {
    const [state, dispatch] = useReducer(gameReducer, getMockState(scenario));

    useEffect(() => {
        socket.setMockMode(true);
        return () => {
            socket.setMockMode(false);
        };
    }, []);

    useEffect(() => {
        if (typeof window === "undefined") return;
        document.documentElement.classList.add("theme-tabletop");
        document.documentElement.classList.toggle("dark", state.theme === "dark");
    }, [state.theme]);

    return (
        <GameContext.Provider value={{ state, dispatch }}>
            {children}
        </GameContext.Provider>
    );
};
