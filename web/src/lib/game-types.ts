import type React from "react";
import {Anchor, Coins, Crown, HandCoins, Repeat, Skull, Sword,} from "lucide-react";

export type Role = 'Duke' | 'Assassin' | 'Captain' | 'Ambassador' | 'Contessa';

export interface Player {
  id: string;
  name: string;
  avatar: string;
  coins: number;
  cards: Role[];
  revealedCards: Role[];
  isActive: boolean;
  isEliminated: boolean;
}

export interface GameRoom {
  id: string;
  name: string;
  players: Player[];
  maxPlayers: number;
  status: 'waiting' | 'playing' | 'finished';
  host: string;
}

export type ActionType = 
  | 'income'
  | 'foreign-aid'
  | 'coup'
  | 'tax'
  | 'assassinate'
  | 'steal'
  | 'exchange';

export interface GameAction {
  type: ActionType;
  label: string;
  description: string;
  cost: number;
  role?: Role;
  targetRequired: boolean;
  blockableBy?: Role[];
  challengeable: boolean;
}

export const GAME_ACTIONS: GameAction[] = [
  {
    type: 'income',
    label: 'Income',
    description: 'Take 1 coin from the treasury',
    cost: 0,
    targetRequired: false,
    challengeable: false,
  },
  {
    type: 'foreign-aid',
    label: 'Foreign Aid',
    description: 'Take 2 coins from the treasury',
    cost: 0,
    targetRequired: false,
    blockableBy: ['Duke'],
    challengeable: false,
  },
  {
    type: 'coup',
    label: 'Coup',
    description: 'Pay 7 coins to eliminate a player\'s influence',
    cost: 7,
    targetRequired: true,
    challengeable: false,
  },
  {
    type: 'tax',
    label: 'Tax',
    description: 'Take 3 coins from the treasury',
    cost: 0,
    role: 'Duke',
    targetRequired: false,
    challengeable: true,
  },
  {
    type: 'assassinate',
    label: 'Assassinate',
    description: 'Pay 3 coins to assassinate another player',
    cost: 3,
    role: 'Assassin',
    targetRequired: true,
    blockableBy: ['Contessa'],
    challengeable: true,
  },
  {
    type: 'steal',
    label: 'Steal',
    description: 'Steal 2 coins from another player',
    cost: 0,
    role: 'Captain',
    targetRequired: true,
    blockableBy: ['Captain', 'Ambassador'],
    challengeable: true,
  },
  {
    type: 'exchange',
    label: 'Exchange',
    description: 'Exchange cards with the Court deck',
    cost: 0,
    role: 'Ambassador',
    targetRequired: false,
    challengeable: true,
  },
];

export const ROLE_COLORS: Record<Role, string> = {
  Duke: 'bg-primary',
  Assassin: 'bg-muted',
  Captain: 'bg-accent',
  Ambassador: 'bg-secondary',
  Contessa: 'bg-primary/80',
};

export const ACTION_ICONS: Record<ActionType, React.ElementType> = {
  income: Coins,
  "foreign-aid": HandCoins,
  coup: Sword,
  tax: Crown,
  assassinate: Skull,
  steal: Anchor,
  exchange: Repeat,
};
