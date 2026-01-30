/**
 * Card domain module
 *
 * Core domain logic for card roles and their associated assets.
 * This module contains no React or WebSocket dependencies.
 */

export type { CardRole } from "./types";
export { ALL_CARD_ROLES, isCardRole } from "./types";
export { ROLE_IMAGE, cardImageForRole, cardImageForRoleOrThrow } from "./images";
