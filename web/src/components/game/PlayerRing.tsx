import React from 'react';
import {cn} from '@/lib/utils';
import {motion} from 'framer-motion';
import {socket} from '@/network/socket';
import {Coins, Target} from 'lucide-react';
import {getPlayerPositions} from '@/lib/layout';
import type {RootAction} from '@/state/slices';
import type {GameIdentity, PlayerSnapshot, Prompt, TargetingState, TurnTimerState} from '@/state/types';


type PlayerRingProps = {
    players: PlayerSnapshot[];
    identity: GameIdentity | null;
    activePlayerIndex: number | null;
    targeting: TargetingState | null;
    pendingPrompt: Prompt | null;
    turnTimer: TurnTimerState | null;
    dispatch: React.Dispatch<RootAction>;
    revealHands?: boolean;
    handsByIndex?: Record<number, string[]>;
    onActionSent?: (payload: { actionId: string; sourceIndex: number; targetIndex: number }) => void;
    strikePulse?: { id: string; targetIndex: number } | null;
};

const PlayerRingBase: React.FC<PlayerRingProps> = ({
    players,
    identity,
    activePlayerIndex,
    targeting,
    pendingPrompt,
    turnTimer,
    dispatch,
    revealHands,
    handsByIndex,
    onActionSent,
    strikePulse,
}) => {
    const enemies = players.filter(player => player.index !== identity?.playerIndex);
    const activeTimer = turnTimer?.running ? turnTimer : null;

    const opponentPositions = React.useMemo(() => {
        return getPlayerPositions(enemies.length);
    }, [enemies.length]);

    return (
        <div className="absolute inset-0 pointer-events-auto">
            {enemies.map((player, i) => {
                const isActive = activePlayerIndex === player.index;
                const pos = opponentPositions[i];
                if (!pos) return null;

                const isTargetingMode = Boolean(
                    identity &&
                    targeting?.active &&
                    targeting.requestId &&
                    targeting.actionId &&
                    pendingPrompt?.kind === "action" &&
                    pendingPrompt.requestId === targeting.requestId &&
                    player.index !== identity.playerIndex
                );

                // Check if player meets action-specific requirements
                const meetsActionRequirements = (() => {
                    if (!isTargetingMode || !player.alive) return false;
                    const actionId = targeting?.actionId;
                    if (actionId === "steal") {
                        return (player.coins ?? 0) >= 2;
                    }
                    return true;
                })();

                const isTargetable = isTargetingMode && meetsActionRequirements;



                const coinCount = player.coins ?? 0;

                const isInvalidTarget = isTargetingMode && !meetsActionRequirements;
                const invalidReason = isInvalidTarget && targeting?.actionId === "steal"
                    ? "Need 2+ coins to steal"
                    : null;
                const roles = revealHands ? handsByIndex?.[player.index] ?? [] : [];
                const isStrikeTarget = strikePulse?.targetIndex === player.index;
                const cardCount = player.cardCount ?? 0;

                const PlayerInfo = (
                    <div className="relative flex flex-col items-center justify-center group">
                        {/* Name Tag - Shows on Hover */}
                        <div className="absolute -top-8 opacity-0 group-hover:opacity-100 transition-all duration-200 pointer-events-none whitespace-nowrap z-30">
                            <div className="px-3 py-1 rounded-md bg-popover text-popover-foreground text-xs font-bold shadow-md border border-border">
                                {player.name}
                            </div>
                            <div className="w-2 h-2 bg-popover rotate-45 absolute -bottom-1 left-1/2 -translate-x-1/2 border-r border-b border-border"></div>
                        </div>

                        {/* Avatar Circle */}
                        <div
                            className={cn(
                                "relative rounded-full flex items-center justify-center transition-all duration-300 bg-secondary shadow-lg z-20",
                                "w-[clamp(3.5rem,8vw,5rem)] h-[clamp(3.5rem,8vw,5rem)]",
                                isActive ? "border-2 border-accent ring-4 ring-accent/20 shadow-[0_0_20px_rgba(251,191,36,0.4)]" : "border-2 border-border",
                                !player.alive && "opacity-50 grayscale border-dashed",
                                isTargetable && "cursor-pointer hover:scale-105 hover:border-accent hover:shadow-[0_0_15px_rgba(251,191,36,0.4)]",
                                isInvalidTarget && "opacity-60 border-muted-foreground/30 cursor-not-allowed"
                            )}
                        >
                            {isTargetable && player.alive && (
                                <div className="absolute -top-2 -right-2 w-6 h-6 rounded-full bg-primary flex items-center justify-center shadow-lg border-2 border-accent animate-[spin_3s_linear_infinite] z-30">
                                    <Target className="h-3 w-3 text-primary-foreground" />
                                </div>
                            )}
                            {isInvalidTarget && (
                                <div className="absolute inset-0 flex items-center justify-center z-30 pointer-events-none">
                                    <div className="px-2 py-0.5 rounded bg-destructive/90 text-destructive-foreground text-[10px] font-semibold shadow-md">
                                        {invalidReason}
                                    </div>
                                </div>
                            )}

                            {activeTimer && activeTimer.activePlayerIndex === player.index && (
                                <div className="absolute -inset-1 rounded-full overflow-hidden z-10 pointer-events-none">
                                    <svg className="w-full h-full -rotate-90 transform" viewBox="0 0 100 100">
                                        <circle
                                            cx="50" cy="50" r="48"
                                            stroke="currentColor" strokeWidth="4" fill="transparent"
                                            className={cn("text-accent transition-all duration-100 ease-linear", activeTimer.paused && "animate-none")}
                                            style={{
                                                strokeDasharray: 301.6,
                                                animation: !activeTimer.paused ? `timer-countdown ${activeTimer.durationMs}ms linear forwards` : 'none',
                                                strokeDashoffset: activeTimer.paused ? "150" : "0"
                                            }}
                                        />
                                    </svg>
                                </div>
                            )}

                            {player.avatar ? (
                                <img src={player.avatar} alt={player.name} className="h-full w-full object-cover rounded-full p-0.5" />
                            ) : (
                                <span className="font-serif text-lg font-bold text-foreground/80">{player.name.charAt(0).toUpperCase()}</span>
                            )}
                        </div>

                        {/* Coins Badge - Always Visible */}
                        <div className={cn(
                            "absolute -bottom-2 z-30 flex items-center gap-1 px-2.5 py-0.5 rounded-full bg-background border border-border shadow-sm transition-opacity",
                            !player.alive && "opacity-50"
                        )}>
                            <span className="text-xs font-bold tabular-nums text-foreground">{coinCount}</span>
                            <Coins className="h-3 w-3 text-accent" />
                        </div>
                    </div>
                );

                // Cards positioned closer to center
                const PlayerCards = player.alive && (
                    <div className="flex items-center justify-center gap-1" style={{ width: 'max-content', filter: 'drop-shadow(0 4px 6px rgba(0,0,0,0.3))' }}>
                        {revealHands && roles.length > 0 ? (
                            roles.map((role, idx) => (
                                <div
                                    key={`${role}-${idx}`}
                                    className="rounded bg-card border border-accent/40 flex items-center justify-center shadow-md h-[clamp(2.5rem,5vh,4rem)] w-[clamp(1.75rem,3.5vh,2.5rem)]"
                                    style={{ transform: `rotate(${(idx - roles.length / 2) * 2}deg)` }}
                                >
                                    <span className="text-[8px] font-bold text-accent-foreground/80 -rotate-90 whitespace-nowrap">{role.substring(0, 4)}</span>
                                </div>
                            ))
                        ) : (
                            Array.from({ length: Math.min(cardCount, 3) }).map((_, i) => (
                                <div
                                    key={i}
                                    className="rounded bg-primary/90 border border-accent/20 shadow-sm relative overflow-hidden h-[clamp(2.5rem,5vh,4rem)] w-[clamp(1.75rem,3.5vh,2.5rem)]"
                                    style={{ transform: `rotate(${(i - 1) * 3}deg)` }}
                                >
                                    <div className="absolute inset-0 opacity-20 bg-[radial-gradient(circle_at_center,_var(--tw-gradient-stops))] from-accent to-transparent" />
                                </div>
                            ))
                        )}
                    </div>
                );

                return (
                    <React.Fragment key={`${player.name}-${strikePulse?.id ?? 'idle'}`}>
                        {/* Cards at Inner Ring - Rotated */}
                        {player.alive && (
                            <div
                                className="absolute -translate-x-1/2 -translate-y-1/2 z-10 pointer-events-none"
                                style={{
                                    top: `${pos.cardY}%`,
                                    left: `${pos.cardX}%`,
                                    transform: `translate(-50%, -50%) rotate(${pos.angleDeg}deg)`
                                }}
                            >
                                {PlayerCards}
                            </div>
                        )}

                        {/* Player Info at Outer Ring */}
                        <div
                            className="absolute flex flex-col items-center justify-center -translate-x-1/2 -translate-y-1/2 z-20"
                            style={{ top: `${pos.y}%`, left: `${pos.x}%`, transform: `translate(-50%, -50%)` }}
                            onClick={() => {
                                if (isTargetable && targeting?.requestId) {
                                    // Dispatch targeting logic
                                    if (targeting.actionId === "accuse") {
                                        dispatch({ type: "SET_TARGET_SELECTED", targetIndex: player.index });
                                    } else {
                                        const actionId = targeting.actionId;
                                        const sourceIndex = identity?.playerIndex;
                                        if (actionId == null || sourceIndex == null) return;
                                        socket.send(
                                            "action",
                                            { id: actionId, source_index: sourceIndex, target_index: player.index },
                                            targeting.requestId
                                        );
                                        onActionSent?.({ actionId, sourceIndex, targetIndex: player.index });
                                        dispatch({ type: "CLEAR_PROMPT" });
                                        dispatch({ type: "CLEAR_TARGETING" });
                                    }
                                }
                            }}
                        >
                            <motion.div
                                animate={isStrikeTarget ? { x: [0, -5, 5, 0], opacity: [1, 0.5, 1] } : {}}
                                transition={{ duration: 0.5 }}
                            >
                                {PlayerInfo}
                            </motion.div>
                        </div>
                    </React.Fragment>
                );
            })}
        </div>
    );
};

export const PlayerRing = React.memo(PlayerRingBase, (prev, next) => (
    prev.players === next.players &&
    prev.identity === next.identity &&
    prev.activePlayerIndex === next.activePlayerIndex &&
    prev.targeting === next.targeting &&
    prev.pendingPrompt === next.pendingPrompt &&
    prev.turnTimer === next.turnTimer &&
    prev.dispatch === next.dispatch &&
    prev.revealHands === next.revealHands &&
    prev.handsByIndex === next.handsByIndex &&
    prev.strikePulse?.id === next.strikePulse?.id &&
    prev.strikePulse?.targetIndex === next.strikePulse?.targetIndex
));
