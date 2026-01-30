/**
 * Action definitions
 *
 * Core domain types and mappings for game actions.
 */

import type { CardRole } from "../cards";

/**
 * Maps action IDs to their associated card roles
 */
export const ACTION_ROLE: Record<string, CardRole> = {
  businesswoman: "Businesswoman",
  tax: "TaxCollector",
  tax_business_woman: "TaxCollector",
  investigate: "Policewoman",
  block_investigate: "Policewoman",
  accuse: "Colonel",
  assassinate: "Terrorist",
  block_terrorist: "Colonel",
  steal: "Thief",
  block_steal: "Thief",
  exchange: "Politician",
  block_foreign_aid: "TaxCollector",
};

/**
 * Human-readable labels for actions
 */
export const ACTION_LABEL_OVERRIDES: Record<string, string> = {
  businesswoman: "take 4 coins",
  tax: "collect tax",
  tax_business_woman: "tax businesswoman",
  investigate: "investigate",
  accuse: "accuse",
  assassinate: "assassinate",
  steal: "steal 2 coins",
  exchange: "exchange",
  income: "income",
  foreign_aid: "foreign aid",
  coup: "coup",
  block_foreign_aid: "block foreign aid",
  block_investigate: "block investigate",
  block_terrorist: "block assassinate",
  block_steal: "block steal",
};

/**
 * Get the card role associated with an action
 * @param actionId - The action identifier
 * @returns The associated CardRole or null if no role required
 */
export function roleForAction(actionId: string): CardRole | null {
  return ACTION_ROLE[actionId] ?? null;
}

/**
 * Get the human-readable label for an action
 * @param actionId - The action identifier
 * @returns The action label (defaults to formatted actionId)
 */
export function actionLabel(actionId: string): string {
  return ACTION_LABEL_OVERRIDES[actionId] ?? actionId.replace(/_/g, " ");
}
