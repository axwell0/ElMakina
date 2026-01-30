import React from 'react';
import {AnimatePresence, motion} from 'framer-motion';
import {useGame} from '../store/gameContext';
import {socket} from '../network/socket';
import {cn} from '@/lib/utils';
import {actionLabel, roleForAction} from '@/lib/actions';
import {cardImageForRole} from '@/lib/cards';
import type {Prompt} from '../store/types';
import {ArrowRight, Coins, Crown, Eye, HandCoins, Repeat, Shield, Skull, Sword, Swords, Target} from 'lucide-react';
import {ShineEffect} from '@/components/particles';

const phaseLabelForPrompt = (prompt: Extract<Prompt, { kind: "challenge" | "counter" }>) => {
    if (prompt.kind === "counter") {
        return "Counter Phase";
    }
    if (prompt.challengeKind === "counter") {
        return "Counter Challenge Phase";
    }
    return "Challenge Phase";
};

export const PhaseOverlay: React.FC = () => {
    const { state } = useGame();
    const prompt = state.pendingPrompt;
    const [hasResponded, setHasResponded] = React.useState(false);
    const [announcement, setAnnouncement] = React.useState<{ label: string; key: string } | null>(null);

    React.useEffect(() => {
        if (!prompt || (prompt.kind !== "challenge" && prompt.kind !== "counter")) {
            return;
        }
        setHasResponded(false);
        const label = phaseLabelForPrompt(prompt);
        const key = `${prompt.requestId}-${label}-${Date.now()}`;
        setAnnouncement({ label, key });
        const timeout = window.setTimeout(() => {
            setAnnouncement(null);
        }, 900);
        return () => window.clearTimeout(timeout);
    }, [prompt]);

    if (!prompt || (prompt.kind !== "challenge" && prompt.kind !== "counter")) {
        return null;
    }

    const playerIndex = state.identity?.playerIndex;
    const canRespond = typeof playerIndex === "number";
    const timeoutMs = prompt.timeoutMs;

    const nameForIndex = (index: number) =>
        state.identity?.playerNames?.[index]
        ?? state.players.find(player => player.index === index)?.name
        ?? `Player ${index + 1}`;

    const actorName = nameForIndex(prompt.actorIndex);
    const targetName = typeof prompt.targetIndex === "number" ? nameForIndex(prompt.targetIndex) : null;

    const avatarForIndex = (index: number) => {
        return state.players.find(player => player.index === index)?.avatar;
    };

    const actorAvatar = avatarForIndex(prompt.actorIndex);
    const targetAvatar = typeof prompt.targetIndex === "number" ? avatarForIndex(prompt.targetIndex) : null;
    const actionText = actionLabel(prompt.actionId);

    const roleName = prompt.kind === "challenge" ? prompt.claimedRole : roleForAction(prompt.actionId);
    const cardImage = roleName ? cardImageForRole(roleName) : null;

    const actionIcons: Record<string, React.ElementType> = {
        income: Coins,
        foreign_aid: HandCoins,
        coup: Sword,
        businesswoman: Coins,
        tax: Crown,
        investigate: Eye,
        accuse: Target,
        assassinate: Skull,
        steal: HandCoins,
        exchange: Repeat,
        block_steal: Shield,
        block_investigate: Shield,
        block_terrorist: Shield,
        block_foreign_aid: Shield,
        tax_business_woman: Crown,
        escape: Repeat,
    };
    const ActionIcon = actionIcons[prompt.actionId] ?? null;

    const counterActions = prompt.kind === "counter" ? prompt.allowedActions : [];
    const [primaryCounter, ...secondaryCounters] = counterActions;
    const primaryLabel = prompt.kind === "challenge"
        ? "Challenge"
        : primaryCounter
            ? actionLabel(primaryCounter)
            : "Counter";

    const isEligible = prompt.eligible !== false;

    const sendChallenge = (pass: boolean) => {
        if (!canRespond || hasResponded || !isEligible) return;
        socket.send("challenge", { challenger_index: playerIndex, pass }, prompt.requestId);
        setHasResponded(true);
        setTimeout(() => setHasResponded(false), 5000);
    };

    const sendCounter = (actionId: string) => {
        if (!canRespond || hasResponded || !isEligible) return;
        socket.send("counter", { id: actionId, source_index: playerIndex, main_action: prompt.actionId }, prompt.requestId);
        setHasResponded(true);
        setTimeout(() => setHasResponded(false), 5000);
    };

    const sendCounterPass = () => {
        if (!canRespond || hasResponded || !isEligible) return;
        socket.send("counter", { pass: true }, prompt.requestId);
        setHasResponded(true);
        setTimeout(() => setHasResponded(false), 5000);
    };

    return (
        <div className="pointer-events-none absolute inset-0 z-[9999] flex items-center justify-center px-2 sm:px-4 py-3 sm:py-8">
            <AnimatePresence>
                {announcement && (
                    <motion.div
                        key={announcement.key}
                        initial={{ opacity: 0, scale: 0.9, y: -20 }}
                        animate={{ opacity: 1, scale: 1, y: 0 }}
                        exit={{ opacity: 0, scale: 0.95, y: -10 }}
                        className="absolute top-[18%] rounded-full border border-border bg-card/80 px-4 sm:px-6 py-1.5 sm:py-2 text-xs uppercase tracking-[0.45em] text-muted-foreground shadow-lg backdrop-blur"
                    >
                        {announcement.label}
                    </motion.div>
                )}
            </AnimatePresence>

            <div className="relative pointer-events-auto">
                <motion.div
                    key={prompt.requestId}
                    initial={{ scale: 0.3, rotateX: 90, y: -100, opacity: 0, filter: "brightness(2) blur(8px)" }}
                    animate={{ scale: 1, rotateX: 0, y: 0, opacity: 1, filter: "brightness(1) blur(0px)" }}
                    exit={{ scale: 0.3, rotateY: 90, opacity: 0, filter: "brightness(0) blur(10px)" }}
                    transition={{ type: "spring", stiffness: 300, damping: 25, mass: 0.8 }}
                    className="relative w-full max-w-sm rounded-2xl border-2 border-accent bg-card/95 p-3 sm:p-4 shadow-[0_0_50px_rgba(0,0,0,0.5)] backdrop-blur-md flex flex-col -translate-y-8"
                    style={{ position: 'relative', top: '0', maxHeight: '90vh' }}
                >
                    <div className="absolute inset-0 bg-linear-to-b from-accent/5 to-transparent pointer-events-none top-0" />

                    <motion.div
                        className="absolute inset-0 overflow-hidden rounded-2xl pointer-events-none"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: [0, 0.4, 0] }}
                        transition={{ duration: 1.5, delay: 0.2 }}
                    >
                        {[...Array(8)].map((_, i) => (
                            <motion.div
                                key={i}
                                className="absolute top-1/2 left-1/2 w-1 h-full bg-linear-to-t from-transparent via-accent/50 to-transparent origin-bottom"
                                style={{ transform: `rotate(${i * 45}deg)` }}
                                initial={{ scaleY: 0, opacity: 0 }}
                                animate={{ scaleY: 2, opacity: [0, 1, 0] }}
                                transition={{ duration: 0.8, delay: 0.1 + i * 0.05, ease: "easeOut" }}
                            />
                        ))}
                    </motion.div>

                    <ShineEffect />

                    <div className="mb-2 flex items-center gap-3">
                        <motion.div
                            className="relative flex h-10 w-10 items-center justify-center rounded-full bg-gradient-to-br from-accent/30 to-primary/30"
                            initial={{ scale: 0, rotate: -180 }}
                            animate={{ scale: 1, rotate: 0 }}
                            transition={{ type: "spring", stiffness: 200, damping: 15, delay: 0.2 }}
                        >
                            <motion.div
                                className="absolute inset-0 rounded-full bg-accent/40 blur-lg"
                                animate={{ scale: [1, 1.3, 1], opacity: [0.5, 0.8, 0.5] }}
                                transition={{ duration: 1.5, repeat: Number.POSITIVE_INFINITY }}
                            />
                            <motion.div
                                className="absolute inset-0 rounded-full border border-accent/60"
                                animate={{ rotate: 360 }}
                                transition={{ duration: 3, repeat: Number.POSITIVE_INFINITY, ease: "linear" }}
                            />
                            {ActionIcon ? (
                                <ActionIcon className="h-5 w-5 text-accent" />
                            ) : prompt.kind === "challenge" ? (
                                <Swords className="h-5 w-5 text-accent" />
                            ) : (
                                <Shield className="h-5 w-5 text-accent" />
                            )}
                        </motion.div>
                        <div className="flex-1">
                            <motion.h3
                                className="font-serif text-lg sm:text-xl text-foreground drop-shadow-md leading-tight"
                                initial={{ x: -20, opacity: 0 }}
                                animate={{ x: 0, opacity: 1 }}
                                transition={{ delay: 0.3 }}
                            >
                                {actionText}
                            </motion.h3>
                            <motion.p
                                className="text-xs text-muted-foreground"
                                initial={{ x: -20, opacity: 0 }}
                                animate={{ x: 0, opacity: 1 }}
                                transition={{ delay: 0.4 }}
                            >
                                by <span className="text-accent font-medium">{actorName}</span>
                            </motion.p>
                        </div>
                    </div>

                    <div className="flex flex-col items-center justify-center gap-1.5 mb-2 p-1.5 rounded-xl bg-secondary/20 border border-border/50">
                        <div className="flex items-center gap-2 sm:gap-4 relative z-10">
                            {/* Actor */}
                            <div className="flex flex-col items-center gap-0.5">
                                <div className="h-8 w-8 sm:h-10 sm:w-10 rounded-full border-2 border-accent bg-background overflow-hidden shadow-sm">
                                    {actorAvatar ? (
                                        <img src={actorAvatar} alt={actorName} className="h-full w-full object-cover" />
                                    ) : (
                                        <div className="h-full w-full flex items-center justify-center font-serif font-bold text-xs text-foreground bg-accent/10">
                                            {actorName.charAt(0)}
                                        </div>
                                    )}
                                </div>
                                <span className="text-[10px] font-bold text-accent uppercase tracking-wider max-w-[60px] truncate text-center">
                                    {actorName}
                                </span>
                            </div>

                            {/* Action Arrow */}
                            <div className="flex flex-col items-center gap-0.5 px-0.5">
                                <ArrowRight className="h-3 w-3 text-accent" />
                            </div>

                            {/* Target (if applicable) */}
                            {targetName ? (
                                <div className="flex flex-col items-center gap-0.5">
                                    <div className="h-8 w-8 sm:h-10 sm:w-10 rounded-full border-2 border-border bg-background overflow-hidden shadow-sm">
                                        {targetAvatar ? (
                                            <img src={targetAvatar} alt={targetName} className="h-full w-full object-cover" />
                                        ) : (
                                            <div className="h-full w-full flex items-center justify-center font-serif font-bold text-xs text-muted-foreground bg-secondary">
                                                {targetName.charAt(0)}
                                            </div>
                                        )}
                                    </div>
                                    <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider max-w-[60px] truncate text-center">
                                        {targetName}
                                    </span>
                                </div>
                            ) : (
                                <div className="flex flex-col items-center gap-0.5 opacity-50">
                                    <div className="h-8 w-8 sm:h-10 sm:w-10 rounded-full border-2 border-dashed border-border flex items-center justify-center bg-secondary/50">
                                        <span className="text-[10px] text-muted-foreground uppercase">All</span>
                                    </div>
                                    <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider">
                                        Everyone
                                    </span>
                                </div>
                            )}
                        </div>

                        {roleName && prompt.kind === "challenge" && (
                            <div className="text-xs text-muted-foreground font-serif italic border-t border-border/40 pt-1 w-full text-center">
                                claiming <span className="font-bold text-accent">{roleName}</span>
                            </div>
                        )}
                    </div>

                    {cardImage && (
                        <div className="relative mx-auto mt-0.5 w-fit max-w-full shrink-0 px-4">
                            <img
                                src={cardImage}
                                alt={roleName ?? "role"}
                                className="h-24 sm:h-28 md:h-32 w-auto max-w-full rounded-lg border border-border object-cover shadow-lg"
                            />
                        </div>
                    )}

                    {timeoutMs ? (
                        <div className="modal-timer modal-timer-large mt-2" style={{ animationDuration: `${timeoutMs}ms` }} />
                    ) : null}

                    <div className="mt-2 flex flex-col gap-1.5 shrink-0">
                        <div className="flex flex-col gap-1.5">
                            <button
                                disabled={!canRespond || hasResponded || !isEligible}
                                onClick={() => {
                                    if (prompt.kind === "challenge") {
                                        sendChallenge(false);
                                    } else if (primaryCounter) {
                                        sendCounter(primaryCounter);
                                    }
                                }}
                                className={cn(
                                    "w-full h-10 rounded-lg border-2 border-primary bg-primary/10 px-4 text-xs font-bold uppercase tracking-[0.3em] text-primary transition-all duration-300",
                                    "hover:bg-primary hover:text-primary-foreground shadow-sm active:scale-95 disabled:cursor-not-allowed disabled:opacity-50"
                                )}
                            >
                                {primaryLabel}
                            </button>

                            <button
                                className={cn(
                                    "w-full h-9 rounded-lg border border-border bg-transparent px-4 py-1 text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground transition-all duration-300",
                                    "hover:bg-secondary hover:text-foreground active:scale-95 disabled:cursor-not-allowed disabled:opacity-50"
                                )}
                                disabled={!canRespond || hasResponded || !isEligible}
                                onClick={prompt.kind === "challenge" ? () => sendChallenge(true) : sendCounterPass}
                            >
                                Pass / Allow
                            </button>
                        </div>

                        {!hasResponded && isEligible && prompt.kind === "counter" && secondaryCounters.length > 0 && (
                            <div className="flex flex-col gap-1">
                                <div className="text-[10px] text-center uppercase tracking-widest text-muted-foreground">Others</div>
                                <div className="flex flex-wrap items-center justify-center gap-1.5">
                                    {secondaryCounters.map(action => (
                                        <button
                                            key={action}
                                            className="h-7 rounded-full border border-border bg-card/80 px-3 text-[10px] font-bold uppercase tracking-wider text-muted-foreground hover:text-foreground hover:border-accent transition-colors"
                                            disabled={!canRespond || !isEligible}
                                            onClick={() => sendCounter(action)}
                                        >
                                            {actionLabel(action)}
                                        </button>
                                    ))}
                                </div>
                            </div>
                        )}
                    </div>

                    <div className="mt-2 text-[10px] font-bold uppercase tracking-[0.2em] text-center text-muted-foreground/60">
                        {!hasResponded && !isEligible && "Waiting..."}
                        {hasResponded && "Responded"}
                    </div>
                </motion.div>
            </div>
        </div>
    );
};
