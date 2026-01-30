/**
 * Card image mappings
 *
 * Maps card roles to their image asset paths.
 */

import type { CardRole } from "./types";
import { isCardRole } from "./types";

/**
 * Mapping of card roles to image paths
 */
export const ROLE_IMAGE: Record<CardRole, string> = {
  Businesswoman: "/cards/business.png",
  TaxCollector: "/cards/tax.png",
  Policewoman: "/cards/police.png",
  Colonel: "/cards/colonel.png",
  Terrorist: "/cards/terrorist.png",
  Thief: "/cards/thief.png",
  Politician: "/cards/politician.png",
};

/**
 * Get the image path for a card role
 * @param role - The card role
 * @returns Image path or null if role is invalid
 */
export function cardImageForRole(role: string): string | null {
  if (isCardRole(role)) {
    return ROLE_IMAGE[role];
  }
  return null;
}

/**
 * Get the image path for a card role (throws on invalid role)
 * @param role - The card role
 * @returns Image path
 * @throws Error if role is invalid
 */
export function cardImageForRoleOrThrow(role: CardRole): string {
  const image = ROLE_IMAGE[role];
  if (!image) {
    throw new Error(`Invalid card role: ${role}`);
  }
  return image;
}
