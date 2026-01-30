"use client";

import {createContext, type ReactNode, useContext, useState} from 'react';
import type {ActionType, GameRoom, Player, Role} from './game-types';

type Screen = 'lobby' | 'game' | 'gameover';
type Theme = 'tabletop' | 'modern';

export interface PendingAction {
  type: ActionType;
  actor: Player;
  target?: Player;
  claimedRole?: Role;
}

export interface PendingCounter {
  blocker: Player;
  claimedRole: Role;
  originalAction: PendingAction;
}

interface GameContextType {
  screen: Screen;
  setScreen: (screen: Screen) => void;
  theme: Theme;
  setTheme: (theme: Theme) => void;
  currentPlayer: Player | null;
  setCurrentPlayer: (player: Player | null) => void;
  players: Player[];
  setPlayers: (players: Player[]) => void;
  rooms: GameRoom[];
  setRooms: (rooms: GameRoom[]) => void;
  activePlayerId: string | null;
  setActivePlayerId: (id: string | null) => void;
  winner: Player | null;
  setWinner: (player: Player | null) => void;
  turnLog: string[];
  addLogEntry: (entry: string) => void;
  // New game state
  pendingAction: PendingAction | null;
  setPendingAction: (action: PendingAction | null) => void;
  pendingCounter: PendingCounter | null;
  setPendingCounter: (counter: PendingCounter | null) => void;
}

const GameContext = createContext<GameContextType | null>(null);

const MOCK_ROOMS: GameRoom[] = [
  {
    id: '1',
    name: 'Royal Court',
    players: [
      { id: '1', name: 'Duke Maximilian', avatar: '/placeholder.svg?height=40&width=40', coins: 3, cards: ['Duke', 'Assassin'], revealedCards: [], isActive: true, isEliminated: false },
      { id: '2', name: 'Lady Isabella', avatar: '/placeholder.svg?height=40&width=40', coins: 2, cards: ['Captain', 'Contessa'], revealedCards: [], isActive: false, isEliminated: false },
    ],
    maxPlayers: 6,
    status: 'waiting',
    host: 'Duke Maximilian',
  },
  {
    id: '2',
    name: 'Merchant Guild',
    players: [
      { id: '3', name: 'Baron Von Richter', avatar: '/placeholder.svg?height=40&width=40', coins: 4, cards: ['Ambassador', 'Duke'], revealedCards: [], isActive: false, isEliminated: false },
      { id: '4', name: 'Countess Elara', avatar: '/placeholder.svg?height=40&width=40', coins: 1, cards: ['Assassin', 'Captain'], revealedCards: [], isActive: false, isEliminated: false },
      { id: '5', name: 'Sir Edmund', avatar: '/placeholder.svg?height=40&width=40', coins: 5, cards: ['Contessa', 'Ambassador'], revealedCards: [], isActive: false, isEliminated: false },
    ],
    maxPlayers: 4,
    status: 'playing',
    host: 'Baron Von Richter',
  },
  {
    id: '3',
    name: 'Shadow Council',
    players: [
      { id: '6', name: 'Lord Blackwood', avatar: '/placeholder.svg?height=40&width=40', coins: 2, cards: ['Duke', 'Captain'], revealedCards: [], isActive: false, isEliminated: false },
    ],
    maxPlayers: 6,
    status: 'waiting',
    host: 'Lord Blackwood',
  },
];

export function GameProvider({ children }: { children: ReactNode }) {
  const [screen, setScreen] = useState<Screen>('lobby');
  const [theme, setTheme] = useState<Theme>('tabletop');
  const [currentPlayer, setCurrentPlayer] = useState<Player | null>(null);
  const [players, setPlayers] = useState<Player[]>([]);
  const [rooms, setRooms] = useState<GameRoom[]>(MOCK_ROOMS);
  const [activePlayerId, setActivePlayerId] = useState<string | null>(null);
  const [winner, setWinner] = useState<Player | null>(null);
  const [turnLog, setTurnLog] = useState<string[]>([
    'Game started',
    'Duke Maximilian takes 1 coin (Income)',
    'Lady Isabella claims Duke and takes 3 coins (Tax)',
    'Baron Von Richter challenges Lady Isabella...',
    'Challenge failed! Baron loses influence.',
  ]);
  const [pendingAction, setPendingAction] = useState<PendingAction | null>(null);
  const [pendingCounter, setPendingCounter] = useState<PendingCounter | null>(null);

  const addLogEntry = (entry: string) => {
    setTurnLog(prev => [...prev, entry]);
  };

  return (
    <GameContext.Provider value={{
      screen,
      setScreen,
      theme,
      setTheme,
      currentPlayer,
      setCurrentPlayer,
      players,
      setPlayers,
      rooms,
      setRooms,
      activePlayerId,
      setActivePlayerId,
      winner,
      setWinner,
      turnLog,
      addLogEntry,
      pendingAction,
      setPendingAction,
      pendingCounter,
      setPendingCounter,
    }}>
      {children}
    </GameContext.Provider>
  );
}

export function useGame() {
  const context = useContext(GameContext);
  if (!context) throw new Error('useGame must be used within GameProvider');
  return context;
}
