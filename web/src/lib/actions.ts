// Re-export from domain module for backwards compatibility
export type { CardRole } from "@/domain/game";
export {
    roleForAction,
    actionLabel,
    CHALLENGE_IMAGE,
    mainActionImage,
    counterActionImage,
} from "@/domain/game";
