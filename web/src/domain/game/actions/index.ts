/**
 * Action domain module
 *
 * Core domain logic for game actions and their relationships to card roles.
 * This module contains no React or WebSocket dependencies.
 */

export {
  ACTION_ROLE,
  ACTION_LABEL_OVERRIDES,
  roleForAction,
  actionLabel,
} from "./definitions";

export { CHALLENGE_IMAGE, mainActionImage, counterActionImage } from "./helpers";
