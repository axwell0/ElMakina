/**
 * Card role types and definitions
 *
 * Core domain types for card roles in the game.
 */

/**
 * Available card roles in the game
 * Each role has unique abilities and interactions
 */
export type CardRole =
  | "Businesswoman"
  | "TaxCollector"
  | "Policewoman"
  | "Colonel"
  | "Terrorist"
  | "Thief"
  | "Politician";

/**
 * All card roles as a readonly array
 */
export const ALL_CARD_ROLES: readonly CardRole[] = [
  "Businesswoman",
  "TaxCollector",
  "Policewoman",
  "Colonel",
  "Terrorist",
  "Thief",
  "Politician",
] as const;

/**
 * Type guard to check if a string is a valid CardRole
 */
export function isCardRole(role: string): role is CardRole {
  return ALL_CARD_ROLES.includes(role as CardRole);
}
