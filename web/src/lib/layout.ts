/**
 * Shared layout constants and utilities for the game board.
 * Coordinates are in percentage (%) units relative to the board container.
 */

export const BOARD_LAYOUT = {
    X_CENTER: 50,
    Y_CENTER: 50,
    RADIUS_X: 44,  // Horizontal radius (with margins)
    RADIUS_Y: 42,  // Vertical radius (with margins)
    CARD_OFFSET: 12, // Distance from player to cards (towards center)
};

export interface LayoutPosition {
    x: number;
    y: number;
    cardX: number;
    cardY: number;
    angleDeg: number;
}

/**
 * Calculates evenly distributed positions around an ellipse.
 * @param count Number of players to distribute (excluding self).
 * @returns Array of positions.
 */
export function getPlayerPositions(count: number): LayoutPosition[] {
    if (count <= 0) return [];

    const positions: LayoutPosition[] = [];

    for (let i = 0; i < count; i++) {
        // Distribute evenly in a circle, starting from top (-90 degrees)
        const angle = (i / count) * 2 * Math.PI - Math.PI / 2;

        // Position on ellipse
        const x = BOARD_LAYOUT.X_CENTER + Math.cos(angle) * BOARD_LAYOUT.RADIUS_X;
        const y = BOARD_LAYOUT.Y_CENTER + Math.sin(angle) * BOARD_LAYOUT.RADIUS_Y;

        // Calculate vector from player to center
        const dx = BOARD_LAYOUT.X_CENTER - x;
        const dy = BOARD_LAYOUT.Y_CENTER - y;
        const dist = Math.sqrt(dx * dx + dy * dy) || 1;

        // Unit vector towards center
        const ux = dx / dist;
        const uy = dy / dist;

        // Card position: move towards center by CARD_OFFSET
        const cardX = x + ux * BOARD_LAYOUT.CARD_OFFSET;
        const cardY = y + uy * BOARD_LAYOUT.CARD_OFFSET;

        // Rotation angle for cards (pointing towards center)
        const angleDeg = (Math.atan2(dx, -dy) * 180) / Math.PI;

        positions.push({
            x: Number(x.toFixed(4)),
            y: Number(y.toFixed(4)),
            cardX: Number(cardX.toFixed(4)),
            cardY: Number(cardY.toFixed(4)),
            angleDeg: Number(angleDeg.toFixed(4))
        });
    }

    return positions;
}
