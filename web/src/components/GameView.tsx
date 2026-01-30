import React from 'react';
import {useGame} from '../store/gameContext';
import {ActionPanel} from './ActionPanel';
import {InteractionModal} from './InteractionModal';
import {RevealModal} from './RevealModal';
import {CoinManager} from './CoinManager';
import {GameOverModal} from './GameOverModal';
import {PhaseOverlay} from './PhaseOverlay';
import {GamePausedOverlay} from './GamePausedOverlay';
import {cardImageForRole} from '@/lib/cards';
import {PlayerRing} from './game/PlayerRing';
import {HandTray} from './game/HandTray';
import {HoverRoleCard, type RoleDetailsMap} from './game/HoverRoleCard';
import {SelfHud} from './game/SelfHud';
import {getPlayerPositions} from '../lib/layout';
import {ChatBox} from './game/ChatBox';
import {ArrowLeft, Coins, Flame, Moon, Sun, Volume2, VolumeX} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {socket} from '@/network/socket';
import {cn} from '@/lib/utils';
import {AnimatePresence, motion} from 'framer-motion';

export const GameView: React.FC = () => {
    const { state, dispatch } = useGame();
    const [hoverRole] = React.useState<string | null>(null);
    const [showReconnecting, setShowReconnecting] = React.useState(false);
    const preloadedRef = React.useRef(false);
    const boardRef = React.useRef<HTMLDivElement | null>(null);
    const [actionCue, setActionCue] = React.useState<{ id: string; actionId: string; sourceIndex: number; targetIndex: number } | null>(null);

    React.useEffect(() => {
        if (!actionCue) return;
        const timer = window.setTimeout(() => setActionCue(null), 1400);
        return () => window.clearTimeout(timer);
    }, [actionCue]);

    React.useEffect(() => {
    if (state.mockScenario !== 'assassinate') return;
        if (!state.players.length) return;
        const sourceIndex = state.players.find(p => p.alive)?.index ?? 0;
        const targetIndex =
            state.players.find(p => p.alive && p.index !== sourceIndex)?.index ?? sourceIndex;
        if (sourceIndex === targetIndex) return;
        const timer = window.setInterval(() => {
            setActionCue({
                id: `mock-assassinate-${Date.now()}`,
                actionId: 'assassinate',
                sourceIndex,
                targetIndex,
            });
        }, 3000);
        return () => window.clearInterval(timer);
    }, [state.mockScenario, state.players]);

    React.useEffect(() => {
        if (state.isConnected) {
            setShowReconnecting(false);
            return;
        }
        // Small delay before showing the big overlay
        const timer = setTimeout(() => {
            setShowReconnecting(true);
        }, 5000);
        return () => clearTimeout(timer);
    }, [state.isConnected]);

    const playersByIndex = React.useMemo(
        () => new Map(state.players.map(player => [player.index, player])),
        [state.players]
    );
    const lastTurn = state.logs.length > 0 ? state.logs[state.logs.length - 1].turn : null;
    const roundLabel = lastTurn && lastTurn > 0 ? lastTurn : 1;
    const aliveCount = state.players.filter(player => player.alive).length;

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
    const centerTimer = state.turnTimer?.running ? state.turnTimer : null;
    const centerActor =
        centerTimer && typeof centerTimer.activePlayerIndex === 'number'
            ? playersByIndex.get(centerTimer.activePlayerIndex) ?? null
            : null;

    React.useEffect(() => {
        if (!state.currentLobby || state.currentLobby.status !== 'in_game') return;
        if (preloadedRef.current) return;
        preloadedRef.current = true;
        const assets = [
            '/cards/business.png',
            '/cards/tax.png',
            '/cards/police.png',
            '/cards/colonel.png',
            '/cards/terrorist.png',
            '/cards/thief.png',
            '/cards/politician.png',
        ];
        const avatarAssets = state.players.map(player => player.avatar).filter(Boolean) as string[];
        [...assets, ...avatarAssets].forEach((src) => {
            const img = new Image();
            img.src = src;
        });
    }, [state.currentLobby, state.players]);

    const handleLeaveGame = () => {
        socket.disconnect();
        socket.connect();
    };

    const enemyPositions = React.useMemo(() => {
        const enemies = state.players.filter(player => player.index !== state.identity?.playerIndex);
        if (enemies.length === 0) return new Map<number, { x: number; y: number }>();

        const playerPositions = getPlayerPositions(enemies.length);
        const map = new Map<number, { x: number; y: number }>();

        enemies.forEach((enemy, idx) => {
            const pos = playerPositions[idx];
            if (pos) map.set(enemy.index, { x: pos.x, y: pos.y });
        });

        return map;
    }, [state.players, state.identity?.playerIndex]);

    const resolvePosition = React.useCallback((playerIndex?: number) => {
        const board = boardRef.current;
        if (!board) return null;
        const rect = board.getBoundingClientRect();
        if (typeof playerIndex !== 'number') {
            return { x: rect.width / 2, y: rect.height / 2 };
        }
        if (state.identity && playerIndex === state.identity.playerIndex) {
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
    }, [enemyPositions, state.identity]);

    const strike = actionCue && actionCue.actionId === 'assassinate'
        ? {
            from: resolvePosition(actionCue.sourceIndex),
            to: resolvePosition(actionCue.targetIndex),
        }
        : null;
    const strikePulse = actionCue && actionCue.actionId === 'assassinate'
        ? { id: actionCue.id, targetIndex: actionCue.targetIndex }
        : null;

    return (
        <div className="h-svh w-full overflow-hidden bg-background text-foreground flex flex-col">
            <div className="fixed inset-0 pointer-events-none overflow-hidden">
                <div className="absolute top-1/4 left-1/4 h-80 w-80 rounded-full bg-accent/10 blur-3xl" />
                <div className="absolute bottom-1/4 right-1/4 h-96 w-96 rounded-full bg-primary/10 blur-3xl" />
            </div>

            <header className="relative z-[1200] border-b border-border bg-card/80 px-4 py-2 backdrop-blur-sm shrink-0">
                <div className="flex w-full items-center justify-between">
                    <Button
                        variant="ghost"
                        className="text-muted-foreground hover:text-foreground h-11 px-6"
                        onClick={handleLeaveGame}
                    >
                        <ArrowLeft className="mr-2 h-4 w-4" aria-hidden="true" />
                        Leave Game
                    </Button>
                    <div className="flex items-center gap-2">
                        <Flame className="h-4 w-4 text-accent" aria-hidden="true" />
                        <h1 className="font-sans font-bold text-base sm:text-lg text-foreground tracking-tight">ElMakina</h1>
                    </div>
                    <div className="flex items-center gap-2">
                        <Button
                            onClick={() => dispatch({ type: "SET_SFX_MUTED", muted: !state.sfxMuted })}
                            variant="ghost"
                            size="icon"
                            className="h-9 w-9"
                            aria-label={state.sfxMuted ? "Unmute SFX" : "Mute SFX"}
                        >
                            {state.sfxMuted ? <VolumeX className="h-4 w-4" /> : <Volume2 className="h-4 w-4" />}
                        </Button>
                        <Button
                            onClick={() => dispatch({ type: "SET_THEME", theme: state.theme === 'dark' ? 'light' : 'dark' })}
                            variant="ghost"
                            size="icon"
                            className="h-9 w-9"
                            aria-label="Toggle Theme"
                        >
                            {state.theme === 'dark' ? <Moon className="h-4 w-4" /> : <Sun className="h-4 w-4" />}
                        </Button>
                    </div>
                </div>
            </header>

            <main className="relative flex-1 flex flex-col min-h-0 overflow-hidden">
                <div className="flex h-full w-full flex-row gap-4 px-4 pb-4 pt-4">
                    {/* Left Sidebar - Player HUD */}
                    <div className="flex-[0_0_100%] md:flex-[0_0_35%] lg:flex-[0_0_30%] flex flex-col justify-between z-[1500]">
                        <div className="flex flex-col gap-4">
                            <div className="relative w-full rounded-2xl border-2 border-accent bg-card/90 p-3 shadow-2xl backdrop-blur-sm">
                                {state.identity && (
                                    <div className="absolute -top-[clamp(1rem,2vw,1.25rem)] -left-[clamp(0.5rem,1.5vw,0.75rem)] z-[1510]">
                                        <SelfHud
                                            player={state.players.find((p) => p.index === state.identity?.playerIndex)}
                                            identity={state.identity}
                                            activePlayerIndex={state.activePlayerIndex}
                                            timer={state.turnTimer}
                                        />
                                    </div>
                                )}

                                {/* Action Panel - Always visible, greyed out when not player's turn */}
                                <div className={cn(
                                    "w-full mb-2 transition-opacity duration-300",
                                    state.activePlayerIndex !== state.identity?.playerIndex && "opacity-50 pointer-events-none"
                                )}>
                                    <ActionPanel />
                                </div>

                                {/* Centered Coin Display */}
                                {state.identity && (
                                    <div className="flex items-center justify-center gap-2 mb-2 py-2 border-y border-border/20 bg-accent/5 rounded-full px-6">
                                        <span className="text-lg font-black tabular-nums text-accent">
                                            {state.players.find(p => p.index === state.identity?.playerIndex)?.coins ?? 0}
                                        </span>
                                        <Coins className="h-5 w-5 text-accent" />
                                    </div>
                                )}

                                {/* Hand Cards */}
                                <div className="flex w-full items-center justify-center">
                                    <HandTray
                                        hand={state.hand}
                                        isActive={state.activePlayerIndex === state.identity?.playerIndex}
                                        onHoverStart={() => {
                                            if (state.activePlayerIndex === state.identity?.playerIndex) {
                                                // Optional: highlight relevant actions
                                            }
                                        }}
                                        onHoverEnd={() => { }}
                                    />
                                </div>
                            </div>
                        </div>

                        {/* Chat Box at the bottom of sidebar */}
                        <div className="mt-4">
                            <ChatBox />
                        </div>
                    </div>

                    {/* Main Board Area */}
                    <div className="flex-[0_0_70%] flex flex-col items-center justify-center min-h-0 relative">
                        <div
                            ref={boardRef}
                            className="relative w-full h-full rounded-[2rem] border-4 border-border bg-secondary/30 shadow-2xl shrink min-h-0 overflow-visible"
                            style={{
                                backgroundImage:
                                    "radial-gradient(ellipse at center, var(--secondary) 0%, var(--card) 70%)",
                            }}
                            role="region"
                            aria-label="Game table with players"
                        >
                            <AnimatePresence>
                                {strike && strike.from && strike.to ? (
                                    <motion.div
                                        key={`strike-live-${actionCue?.id}`}
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
                                                left: (strike.to?.x ?? 0) - 36,
                                                top: (strike.to?.y ?? 0) - 36,
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
                                                left: (strike.to?.x ?? 0) - 50,
                                                top: (strike.to?.y ?? 0) - 50,
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
                                                left: (strike.to?.x ?? 0) - 26,
                                                top: (strike.to?.y ?? 0) - 26,
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
                                                left: (strike.to?.x ?? 0) - 70,
                                                top: (strike.to?.y ?? 0) - 70,
                                                width: 140,
                                                height: 140,
                                            }}
                                        />

                                        {[0, 1, 2, 3, 4].map((idx) => (
                                            <motion.div
                                                key={`smoke-live-${actionCue?.id}-${idx}`}
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
                                                    left: (strike.to?.x ?? 0) - 24,
                                                    top: (strike.to?.y ?? 0) - 24,
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
                                            style={{ width: 70, height: 70, left: (strike.to?.x ?? 0) - 35, top: (strike.to?.y ?? 0) - 35 }}
                                        />
                                    </motion.div>
                                ) : null}
                            </AnimatePresence>
                            <div className="absolute left-1/2 top-1/2 z-10 flex -translate-x-1/2 -translate-y-1/2 flex-col items-center gap-2">
                                <div className="relative flex flex-col items-center justify-center overflow-hidden rounded-full border-2 border-accent bg-card/95 shadow-xl backdrop-blur-sm transition-all duration-500 w-[clamp(3rem,10vw,6rem)] h-[clamp(3rem,10vw,6rem)]">
                                    {centerActor ? (
                                        <div className="relative h-full w-full">
                                            {/* Circular Timer SVG */}
                                            {centerTimer && (
                                                <svg className="absolute inset-0 -rotate-90 transform" viewBox="0 0 100 100">
                                                    <circle
                                                        cx="50"
                                                        cy="50"
                                                        r="48"
                                                        stroke="currentColor"
                                                        strokeWidth="4"
                                                        fill="transparent"
                                                        className="text-accent/20"
                                                    />
                                                    <circle
                                                        cx="50"
                                                        cy="50"
                                                        r="48"
                                                        stroke="currentColor"
                                                        strokeWidth="4"
                                                        fill="transparent"
                                                        strokeDasharray="301.6"
                                                        className={cn(
                                                            "text-accent transition-all duration-100 ease-linear",
                                                            centerTimer.paused && "animate-none"
                                                        )}
                                                        style={{
                                                            animation: !centerTimer.paused
                                                                ? `timer-countdown ${centerTimer.durationMs}ms linear forwards`
                                                                : 'none',
                                                            strokeDashoffset: centerTimer.paused ? "150" : "0" // Just a visual placeholder if paused
                                                        }}
                                                    />
                                                </svg>
                                            )}

                                            {centerActor.avatar ? (
                                                <img
                                                    src={centerActor.avatar}
                                                    alt={centerActor.name}
                                                    className="h-full w-full object-cover rounded-full p-1"
                                                />
                                            ) : (
                                                <div className="flex h-full w-full items-center justify-center font-serif text-2xl sm:text-3xl text-foreground">
                                                    {centerActor.name.charAt(0)}
                                                </div>
                                            )}
                                        </div>
                                    ) : (
                                        <>
                                            <Flame className="mb-1 h-5 w-5 text-accent" aria-hidden="true" />
                                            <span className="text-[10px] text-muted-foreground">{aliveCount} P</span>
                                            <span className="font-serif text-sm sm:text-base md:text-lg text-foreground leading-tight">Rnd {roundLabel}</span>
                                        </>
                                    )}
                                </div>
                            </div>

                            <PlayerRing
                                players={state.players}
                                identity={state.identity}
                                activePlayerIndex={state.activePlayerIndex}
                                targeting={state.targeting}
                                pendingPrompt={state.pendingPrompt}
                                turnTimer={state.turnTimer}
                                strikePulse={strikePulse}
                                dispatch={dispatch}
                                onActionSent={(payload) => {
                                    if (payload.actionId !== 'assassinate') return;
                                    setActionCue({
                                        id: `${payload.actionId}-${Date.now()}`,
                                        actionId: payload.actionId,
                                        sourceIndex: payload.sourceIndex,
                                        targetIndex: payload.targetIndex,
                                    });
                                }}
                            />
                        </div>
                    </div>
                    <HoverRoleCard
                        role={hoverRole}
                        image={hoverImage}
                        details={roleDetails}
                        className="fixed left-3 sm:left-6 top-1/2 -translate-y-1/2 pointer-events-none"
                    />
                </div>
            </main>

            <PhaseOverlay />
            <GamePausedOverlay />
            {showReconnecting && state.pause.status !== "active" && (
                <div className="absolute inset-0 z-[1900] flex items-center justify-center px-3 sm:px-4">
                    <div className="absolute inset-0 bg-background/70 backdrop-blur-sm" />
                    <div className="relative rounded-2xl border border-border bg-card/90 px-4 sm:px-6 py-3 sm:py-4 shadow-xl">
                        <div className="text-sm uppercase tracking-[0.3em] text-muted-foreground text-center">
                            Reconnecting…
                        </div>
                    </div>
                </div>
            )}
            <InteractionModal />
            <RevealModal />
            <CoinManager />
            <GameOverModal />
        </div>
    );
};
