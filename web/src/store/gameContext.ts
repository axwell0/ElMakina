import React, {createContext, useContext} from 'react';
import {GameState, initialGameState} from './types';
import type {RootAction} from '@/state/slices';

export interface GameContextType {
    state: GameState;
    dispatch: React.Dispatch<RootAction>;
}

export const GameContext = createContext<GameContextType>({
    state: initialGameState,
    dispatch: () => undefined,
});

export const useGame = () => useContext(GameContext);
