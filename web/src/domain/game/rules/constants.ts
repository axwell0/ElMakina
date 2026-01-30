/**
 * Game rules constants
 *
 * Core game constants like costs, limits, and thresholds.
 * These are the fundamental rules that govern gameplay.
 */

/**
 * Starting coins for each player
 */
export const STARTING_COINS = 2;

/**
 * Coins required to perform a coup
 */
export const COUP_COST = 7;

/**
 * Coins required to perform an assassination
 */
export const ASSASSINATION_COST = 3;

/**
 * Coins gained from income action
 */
export const INCOME_AMOUNT = 1;

/**
 * Coins gained from foreign aid
 */
export const FOREIGN_AID_AMOUNT = 2;

/**
 * Coins gained from businesswoman action
 */
export const BUSINESSWOMAN_AMOUNT = 4;

/**
 * Coins stolen by thief
 */
export const STEAL_AMOUNT = 2;

/**
 * Maximum coins a player can hold (backend enforced)
 */
export const MAX_COINS = 12;

/**
 * Maximum number of players
 */
export const MAX_PLAYERS = 9;

/**
 * Minimum number of players
 */
export const MIN_PLAYERS = 2;

/**
 * Get the starting hand size based on player count
 * Returns 3 cards for 2-4 players, 2 cards for 5+ players
 * @param playerCount - The number of players in the game
 * @returns The starting hand size (2 or 3 cards)
 */
export function getStartingHandSize(playerCount: number): number {
  return playerCount <= 4 ? 3 : 2;
}
