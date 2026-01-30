import React from 'react';
import {motion} from 'framer-motion';
import {cn} from '@/lib/utils';

import type {GameIdentity, PlayerSnapshot, TurnTimerState} from '@/state/types';

type SelfHudProps = {
    identity: GameIdentity;
    player: PlayerSnapshot | undefined;
    activePlayerIndex: number | null;
    timer: TurnTimerState | null;
    className?: string;
};

const SelfHudBase: React.FC<SelfHudProps> = ({
    identity,
    player,
    activePlayerIndex,
    timer,
    className,
}) => {
    const isActive = activePlayerIndex === identity.playerIndex;
    const activeTimer =
        timer && timer.running && timer.activePlayerIndex === identity.playerIndex ? timer : null;
    const playerName = identity.playerNames[identity.playerIndex] || "Player";

    return (
        <div className={cn("group relative", className)}>
            {/* Hover Name Badge */}
            <div className="absolute -top-[clamp(2rem,3vw,2.5rem)] left-1/2 -translate-x-1/2 px-3 py-1 bg-popover text-popover-foreground text-xs font-bold rounded-md opacity-0 group-hover:opacity-100 transition-opacity duration-200 whitespace-nowrap z-50 pointer-events-none shadow-lg border border-border">
                {playerName}
                <div className="absolute -bottom-1 left-1/2 -translate-x-1/2 w-2 h-2 bg-popover rotate-45 border-r border-b border-border"></div>
            </div>

            <motion.div
                className={cn(
                    "relative flex h-12 w-12 items-center justify-center overflow-hidden rounded-full border-2 shadow-lg transition-transform group-hover:scale-105",
                    isActive ? "border-accent ring-2 ring-accent/20" : "border-border"
                )}
                initial={{ opacity: 0, scale: 0.5 }}
                animate={{ opacity: 1, scale: 1 }}
            >
                {activeTimer && (
                    <div className="absolute inset-0 pointer-events-none">
                        <svg className="w-full h-full -rotate-90 transform" viewBox="0 0 100 100">
                            <circle
                                cx="50" cy="50" r="46"
                                stroke="currentColor" strokeWidth="4" fill="transparent"
                                className={cn("text-accent transition-all duration-100 ease-linear", activeTimer.paused && "animate-none")}
                                style={{
                                    strokeDasharray: 289,
                                    animation: !activeTimer.paused ? `timer-countdown ${activeTimer.durationMs}ms linear forwards` : 'none',
                                    strokeDashoffset: activeTimer.paused ? "145" : "0"
                                }}
                            />
                        </svg>
                    </div>
                )}
                {player?.avatar ? (
                    <img
                        src={player.avatar}
                        alt={playerName}
                        className="h-full w-full object-cover p-0.5 rounded-full"
                    />
                ) : (
                    <span className="text-lg font-semibold text-foreground/80">{playerName.charAt(0)}</span>
                )}
            </motion.div>
        </div>
    );
};

export const SelfHud = React.memo(SelfHudBase, (prev, next) => (
    prev.identity === next.identity &&
    prev.player === next.player &&
    prev.activePlayerIndex === next.activePlayerIndex &&
    prev.timer === next.timer &&
    prev.className === next.className
));
