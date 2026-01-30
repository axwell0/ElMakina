/**
 * Action helpers
 *
 * Helper functions for working with actions and their presentation.
 */

import { cardImageForRole } from "../cards";
import { roleForAction } from "./definitions";

/**
 * Default challenge image path
 */
export const CHALLENGE_IMAGE = "/cards/colonel.png";

/**
 * Get the card image associated with a main action
 * @param actionId - The action identifier
 * @returns Image path or null if not found
 */
export function mainActionImage(actionId: string): string | null {
  const role = roleForAction(actionId);
  return role ? cardImageForRole(role) : null;
}

/**
 * Get the card image associated with a counter action
 * @param actionId - The action identifier
 * @returns Image path or null if not found
 */
export function counterActionImage(actionId: string): string | null {
  const role = roleForAction(actionId);
  return role ? cardImageForRole(role) : null;
}
