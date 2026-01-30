import React from 'react';
import {AnimatePresence, motion} from 'framer-motion';
import {ArrowLeft, Flame} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {cardImageForRole} from '@/lib/cards';
import type {GameIdentity, HandCard, PlayerSnapshot} from '@/state/types';
import {PlayerRing} from '@/components/game/PlayerRing';
import {HandTray} from '@/components/game/HandTray';
import {HoverRoleCard, type RoleDetailsMap} from '@/components/game/HoverRoleCard';
import {SelfHud} from '@/components/game/SelfHud';

type ReplayGameViewProps = {
    players: PlayerSnapshot[];
    identity: GameIdentity | null;
    activePlayerIndex: number | null;
    hand: HandCard[];
    handsByIndex: Record<number, string[]>;
    turnNumber: number;
    onBack: () => void;
    actionCue?: { id: string; text: string; role?: string; actionId?: string; sourceIndex?: number; targetIndex?: number } | null;
    viewerLabel?: string;
};

export const ReplayGameView: React.FC<ReplayGameViewProps> = ({
    players,
    identity,
    activePlayerIndex,
    hand,
    handsByIndex,
    turnNumber,
    onBack,
    actionCue,
    viewerLabel,
}) => {
    const [hoverRole, setHoverRole] = React.useState<string | null>(null);
    const boardRef = React.useRef<HTMLDivElement | null>(null);

    const enemyPositions = React.useMemo(() => {
        const enemies = players.filter(player => player.index !== identity?.playerIndex);
        if (enemies.length === 0) return new Map<number, { x: number; y: number }>();
        const positions: { x: number; y: number }[] = [];
        const centerX = 50;
        const centerY = 50;
        const radiusX = 38;
        const radiusY = 38;
        for (let i = 0; i < enemies.length; i++) {
            const angle = (i / enemies.length) * 2 * Math.PI - Math.PI / 2;
            const x = centerX + Math.cos(angle) * radiusX;
            const y = centerY + Math.sin(angle) * radiusY;
            positions.push({ x, y });
        }
        const map = new Map<number, { x: number; y: number }>();
        enemies.forEach((enemy, idx) => {
            const pos = positions[idx];
            if (pos) map.set(enemy.index, pos);
        });
        return map;
    }, [players, identity?.playerIndex]);

    const resolvePosition = React.useCallback((playerIndex?: number) => {
        const board = boardRef.current;
        if (!board) return null;
        const rect = board.getBoundingClientRect();
        if (typeof playerIndex !== 'number') {
            return { x: rect.width / 2, y: rect.height / 2 };
        }
        if (identity && playerIndex === identity.playerIndex) {
            return { x: rect.width * 0.5, y: rect.height * 0.82 };
        }
        const percent = enemyPositions.get(playerIndex);
        if (!percent) {
            return { x: rect.width / 2, y: rect.height / 2 };
        }
        return {
            x: (percent.x / 100) * rect.width,
            y: (percent.y / 100) * rect.height,
        };
    }, [enemyPositions, identity]);

    const strike = actionCue && actionCue.actionId === 'assassinate' && typeof actionCue.targetIndex === 'number'
        ? (() => {
            const from = resolvePosition(actionCue.sourceIndex);
            const to = resolvePosition(actionCue.targetIndex);
            if (!from || !to) return null;
            return { from, to };
        })()
        : null;
    const strikePulse = actionCue && actionCue.actionId === 'assassinate' && typeof actionCue.targetIndex === 'number'
        ? { id: actionCue.id, targetIndex: actionCue.targetIndex }
        : null;

    const playersByIndex = React.useMemo(
        () => new Map(players.map(player => [player.index, player])),
        [players]
    );
    const aliveCount = players.filter(player => player.alive).length;

    const roleDetails: RoleDetailsMap = React.useMemo(() => ({
        Businesswoman: { main: 'Take 4 coins.', counter: 'none.' },
        TaxCollector: { main: 'Tax: players with 7+ coins pay 1 coin.', counter: 'block foreign aid; tax Businesswoman (take 1 from active player).' },
        Policewoman: { main: 'Investigate: reveal a random card from a target.', counter: 'block investigate.' },
        Colonel: { main: 'Accuse a role: correct -> target discards, wrong -> target gains 4 coins.', counter: 'block assassinate.' },
        Terrorist: { main: 'Assassinate: target discards 1 card.', counter: 'none.' },
        Thief: { main: 'Steal: take 2 coins from a target with 2+ coins.', counter: 'block steal.' },
        Politician: { main: 'Exchange: draw cards equal to your hand and return the same number.', counter: 'none.' }
    }), []);

    const hoverImage = hoverRole ? cardImageForRole(hoverRole) : null;

    return (
        <div className="relative flex min-h-[460px] w-full flex-col overflow-hidden rounded-[2rem] border-4 border-border bg-secondary/30 shadow-2xl sm:min-h-[520px] md:min-h-[600px]">
            <div className="absolute inset-0 pointer-events-none bg-sky-500/10 mix-blend-screen" />
            <div className="absolute right-4 top-4 rounded-full border border-sky-400/50 bg-sky-500/15 px-3 py-1 text-[11px] uppercase tracking-[0.3em] text-sky-300">
                Replay Mode
            </div>

            <header className="relative z-10 border-b border-border/60 bg-card/70 px-4 py-2 backdrop-blur-sm">
                <div className="flex items-center justify-between">
                    <Button variant="ghost" size="sm" className="text-muted-foreground hover:text-foreground h-8" onClick={onBack}>
                        <ArrowLeft className="mr-2 h-4 w-4" aria-hidden="true" />
                        Back to Lobby
                    </Button>
                    <div className="flex items-center gap-2">
                        <Flame className="h-4 w-4 text-accent" aria-hidden="true" />
                        <h1 className="font-sans font-bold text-base sm:text-lg text-foreground tracking-tight">ElMakina</h1>
                    </div>
                    <div className="flex flex-col items-end gap-1 text-xs text-muted-foreground">
                        <span>Turn {turnNumber}</span>
                        {viewerLabel ? (
                            <span className="rounded-full border border-sky-400/50 bg-sky-500/15 px-2 py-0.5 text-[10px] uppercase tracking-[0.2em] text-sky-100">
                                You: {viewerLabel}
                            </span>
                        ) : null}
                    </div>
                </div>
            </header>

            <div className="relative flex-1 p-2 sm:p-3 md:p-4">
                <div
                    ref={boardRef}
                    className="relative mx-auto w-full max-w-2xl aspect-[16/10] sm:aspect-square md:aspect-[16/11] rounded-[2rem] border-4 border-border bg-secondary/30 shadow-2xl"
                    style={{
                        backgroundImage:
                            "radial-gradient(ellipse at center, var(--secondary) 0%, var(--card) 70%)",
                    }}
                    role="region"
                    aria-label="Replay table with players"
                >
                    <AnimatePresence>
                        {strike?.from && strike?.to ? (
                            <motion.div
                                key={`strike-${actionCue?.id}`}
                                initial={{ opacity: 0 }}
                                animate={{ opacity: 1 }}
                                exit={{ opacity: 0 }}
                                className="absolute inset-0 z-30 pointer-events-none"
                            >
                                <motion.div
                                    initial={{ x: strike.from.x, y: strike.from.y, scale: 0.7, opacity: 1, rotate: -30 }}
                                    animate={{
                                        x: [strike.from.x, (strike.from.x + strike.to.x) / 2, strike.to.x],
                                        y: [strike.from.y, Math.min(strike.from.y, strike.to.y) - 160, strike.to.y],
                                        rotate: [-45, 0, 270, 540],
                                        scale: [0.65, 0.9, 1.05],
                                    }}
                                    transition={{ duration: 1.6, ease: 'easeInOut' }}
                                    className="absolute left-0 top-0"
                                >
                                    <svg width="36" height="36" viewBox="0 0 36 36" className="drop-shadow-[0_0_14px_rgba(251,191,36,0.6)]">
                                        <circle cx="16" cy="20" r="10" fill="#1f2937" />
                                        <circle cx="12" cy="16" r="2.5" fill="#fbbf24" />
                                        <path d="M22 10 C26 8, 30 10, 28 16" stroke="#fbbf24" strokeWidth="2" fill="none" />
                                        <circle cx="28" cy="16" r="2.5" fill="#f97316" />
                                    </svg>
                                </motion.div>

                                <motion.div
                                    initial={{ opacity: 0, scale: 0.6 }}
                                    animate={{ opacity: [0, 1, 0], scale: [0.6, 2, 3] }}
                                    transition={{ duration: 0.7, delay: 1.2, ease: 'easeOut' }}
                                    className="absolute rounded-full bg-orange-400/60 blur-xl"
                                    style={{
                                        left: strike.to.x - 36,
                                        top: strike.to.y - 36,
                                        width: 72,
                                        height: 72,
                                    }}
                                />

                                <motion.div
                                    initial={{ opacity: 0, scale: 0.2 }}
                                    animate={{ opacity: [0, 0.9, 0], scale: [0.2, 2.2, 3.6] }}
                                    transition={{ duration: 0.7, delay: 1.15, ease: 'easeOut' }}
                                    className="absolute rounded-full bg-amber-300/50 blur-2xl"
                                    style={{
                                        left: strike.to.x - 50,
                                        top: strike.to.y - 50,
                                        width: 100,
                                        height: 100,
                                    }}
                                />

                                <motion.div
                                    initial={{ opacity: 0 }}
                                    animate={{ opacity: [0, 1, 0] }}
                                    transition={{ duration: 0.35, delay: 1.2 }}
                                    className="absolute rounded-full bg-white/80 blur-sm"
                                    style={{
                                        left: strike.to.x - 26,
                                        top: strike.to.y - 26,
                                        width: 52,
                                        height: 52,
                                    }}
                                />

                                <motion.div
                                    initial={{ opacity: 0, scale: 0.2 }}
                                    animate={{ opacity: [0, 0.7, 0], scale: [0.2, 2.8, 4.2] }}
                                    transition={{ duration: 0.8, delay: 1.25, ease: 'easeOut' }}
                                    className="absolute rounded-full border-2 border-amber-200/60"
                                    style={{
                                        left: strike.to.x - 70,
                                        top: strike.to.y - 70,
                                        width: 140,
                                        height: 140,
                                    }}
                                />

                                {[0, 1, 2, 3, 4].map((idx) => (
                                    <motion.div
                                        key={`smoke-${actionCue?.id}-${idx}`}
                                        initial={{ opacity: 0, scale: 0.6, x: 0, y: 0 }}
                                        animate={{
                                            opacity: [0, 0.7, 0],
                                            scale: [0.6, 1.8, 2.8],
                                            x: (idx - 2) * 22,
                                            y: -28 - idx * 8,
                                        }}
                                        transition={{ duration: 0.9, delay: 1.25 + idx * 0.06, ease: 'easeOut' }}
                                        className="absolute rounded-full bg-slate-200/50 blur-2xl"
                                        style={{
                                            left: strike.to.x - 24,
                                            top: strike.to.y - 24,
                                            width: 48,
                                            height: 48,
                                        }}
                                    />
                                ))}

                                <motion.div
                                    initial={{ x: strike.to.x - 28, y: strike.to.y - 28, opacity: 0 }}
                                    animate={{
                                        opacity: [0, 1, 0],
                                        x: [strike.to.x - 28, strike.to.x - 24, strike.to.x - 30, strike.to.x - 26, strike.to.x - 28],
                                        y: [strike.to.y - 28, strike.to.y - 26, strike.to.y - 30, strike.to.y - 27, strike.to.y - 28],
                                    }}
                                    transition={{ duration: 0.6, delay: 1.2, ease: 'easeOut' }}
                                    className="absolute rounded-full border-2 border-amber-400/90 shadow-[0_0_24px_rgba(251,191,36,0.7)]"
                                    style={{ width: 70, height: 70 }}
                                />
                            </motion.div>
                        ) : null}
                    </AnimatePresence>
                    <AnimatePresence>
                        {actionCue ? (
                            <motion.div
                                key={actionCue.id}
                                initial={{ opacity: 0, scale: 0.9, y: 10 }}
                                animate={{ opacity: 1, scale: 1, y: 0 }}
                                exit={{ opacity: 0, scale: 0.95, y: -6 }}
                                transition={{ duration: 0.35, ease: 'easeOut' }}
                                className="absolute left-1/2 top-1/2 z-20 -translate-x-1/2 -translate-y-1/2"
                            >
                                <div className="flex flex-col items-center gap-2">
                                    {actionCue.role ? (
                                        <div className="h-40 w-28 sm:h-44 sm:w-32 overflow-hidden rounded-xl border border-sky-400/60 bg-sky-500/10 shadow-xl">
                                            {cardImageForRole(actionCue.role) ? (
                                                <img
                                                    src={cardImageForRole(actionCue.role) || ''}
                                                    alt={actionCue.role}
                                                    className="h-full w-full object-cover"
                                                />
                                            ) : (
                                                <div className="flex h-full w-full items-center justify-center text-xs font-semibold text-sky-100">
                                                    {actionCue.role}
                                                </div>
                                            )}
                                        </div>
                                    ) : null}
                                    <div className="rounded-full border border-sky-400/60 bg-sky-500/20 px-5 py-1.5 text-center text-xs sm:text-sm font-semibold text-sky-100 shadow-xl backdrop-blur-sm">
                                        {actionCue.text}
                                    </div>
                                </div>
                            </motion.div>
                        ) : null}
                    </AnimatePresence>

                    <div className="absolute left-1/2 top-1/2 z-10 flex -translate-x-1/2 -translate-y-1/2 flex-col items-center gap-2">
                        <div className="relative flex h-16 w-16 sm:h-20 sm:w-20 md:h-24 md:w-24 flex-col items-center justify-center overflow-hidden rounded-full border-2 border-accent bg-card/95 shadow-xl backdrop-blur-sm transition-all duration-500">
                            <Flame className="mb-1 h-5 w-5 text-accent" aria-hidden="true" />
                            <span className="text-[10px] text-muted-foreground">{aliveCount} P</span>
                            <span className="font-serif text-sm sm:text-base md:text-lg text-foreground leading-tight">Rnd {turnNumber}</span>
                        </div>
                    </div>

                    <PlayerRing
                        players={players}
                        identity={identity}
                        activePlayerIndex={activePlayerIndex}
                        targeting={null}
                        pendingPrompt={null}
                        turnTimer={null}
                        strikePulse={strikePulse}
                        dispatch={() => undefined}
                        revealHands
                        handsByIndex={handsByIndex}
                    />
                </div>

                <HoverRoleCard
                    role={hoverRole}
                    image={hoverImage}
                    details={roleDetails}
                    className="fixed left-6 top-1/2 -translate-y-1/2"
                />
            </div>

            <div className="relative z-10 w-full max-w-5xl self-center shrink-0 p-4 pt-0">
                <div className="relative rounded-2xl border-2 border-accent/70 bg-card/90 p-4 shadow-2xl backdrop-blur-sm">
                    {identity && (
                        <div className="absolute -top-7 -left-3 z-30">
                            <SelfHud
                                identity={identity}
                                player={playersByIndex.get(identity.playerIndex)}
                                activePlayerIndex={activePlayerIndex}
                                timer={null}
                                className="shadow-2xl scale-90 sm:scale-100 origin-bottom-left border-accent border-2 bg-card rounded-xl px-4 py-2"
                            />
                        </div>
                    )}
                    <div className="mt-6 flex w-full flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                        <div className="flex flex-1 items-center justify-center min-w-0">
                            <HandTray
                                hand={hand}
                                isActive={activePlayerIndex === identity?.playerIndex}
                                onHoverStart={setHoverRole}
                                onHoverEnd={(role) => setHoverRole(prev => (prev === role ? null : prev))}
                            />
                        </div>
                        <div className="hidden lg:flex lg:flex-none">
                            <div className="rounded-xl border border-border/60 bg-background/60 px-4 py-3 text-xs text-muted-foreground">
                                Actions disabled in replay mode.
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};
