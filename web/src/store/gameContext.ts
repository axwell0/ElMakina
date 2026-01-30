import React, {createContext, useContext} from 'react';
import {GameState, initialGameState} from './types';
import type {Action} from './gameReducer';

export interface GameContextType {
    state: GameState;
    dispatch: React.Dispatch<Action>;
}

export const GameContext = createContext<GameContextType>({
    state: initialGameState,
    dispatch: () => undefined,
});

export const useGame = () => useContext(GameContext);
