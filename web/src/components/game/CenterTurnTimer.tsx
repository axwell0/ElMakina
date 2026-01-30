import React from 'react';
import {motion, useMotionValue} from 'framer-motion';
import type {PlayerSnapshot, TurnTimerState} from '@/store/types';
import {Timer} from 'lucide-react';

const nowMs = () => (typeof performance !== "undefined" ? performance.now() : Date.now());

const useTurnTimerProgress = (timer: TurnTimerState | null) => {
    const progress = useMotionValue(0);
    const [secondsLeft, setSecondsLeft] = React.useState<number | null>(null);
    const timerRef = React.useRef<TurnTimerState | null>(timer);
    const rafRef = React.useRef<number | null>(null);
    const startRef = React.useRef(0);
    const pausedAtRef = React.useRef<number | null>(null);
    const pausedTotalRef = React.useRef(0);
    const lastSecondRef = React.useRef<number | null>(null);

    React.useEffect(() => {
        timerRef.current = timer;
    }, [timer]);

    React.useEffect(() => {
        if (!timer?.running) {
            progress.set(0);
            setSecondsLeft(null);
            startRef.current = 0;
            pausedAtRef.current = null;
            pausedTotalRef.current = 0;
            lastSecondRef.current = null;
            if (rafRef.current != null) {
                cancelAnimationFrame(rafRef.current);
                rafRef.current = null;
            }
            return;
        }

        startRef.current = nowMs();
        pausedAtRef.current = null;
        pausedTotalRef.current = 0;
        progress.set(0);
        const initial = Math.max(0, Math.ceil(timer.durationMs / 1000));
        lastSecondRef.current = initial;
        setSecondsLeft(initial);
    }, [timer?.key, timer?.running, timer?.durationMs, progress]);

    React.useEffect(() => {
        if (!timer?.running) return;
        const tick = () => {
            const activeTimer = timerRef.current;
            if (!activeTimer?.running || activeTimer.paused) {
                rafRef.current = null;
                return;
            }
            const elapsed = nowMs() - startRef.current - pausedTotalRef.current;
            const durationMs = activeTimer.durationMs > 0 ? activeTimer.durationMs : 1;
            const nextProgress = Math.max(0, Math.min(1, elapsed / durationMs));
            progress.set(nextProgress);
            const remaining = Math.max(0, Math.ceil(((1 - nextProgress) * durationMs) / 1000));
            if (remaining !== lastSecondRef.current) {
                lastSecondRef.current = remaining;
                setSecondsLeft(remaining);
            }
            if (nextProgress < 1) {
                rafRef.current = requestAnimationFrame(tick);
            } else {
                rafRef.current = null;
            }
        };

        const now = nowMs();
        if (timer.paused) {
            if (pausedAtRef.current == null) {
                pausedAtRef.current = now;
            }
            if (rafRef.current != null) {
                cancelAnimationFrame(rafRef.current);
                rafRef.current = null;
            }
            return;
        }

        if (pausedAtRef.current != null) {
            pausedTotalRef.current += now - pausedAtRef.current;
            pausedAtRef.current = null;
        }

        rafRef.current = requestAnimationFrame(tick);
        return () => {
            if (rafRef.current != null) {
                cancelAnimationFrame(rafRef.current);
                rafRef.current = null;
            }
        };
    }, [timer?.paused, timer?.running, timer?.durationMs, timer?.key, progress]);

    return { progress, secondsLeft };
};

type CenterTurnTimerProps = {
    timer: TurnTimerState | null;
    actor: PlayerSnapshot | null;
};

const CenterTurnTimerBase: React.FC<CenterTurnTimerProps> = ({ timer }) => {
    const { secondsLeft } = useTurnTimerProgress(timer);

    if (!timer) return null;

    return (
        <div className="pointer-events-none absolute left-1/2 top-1/2 z-[1500] -translate-x-1/2 translate-y-20">
            <motion.div
                className="rounded-lg border border-border bg-card/90 px-4 py-2 shadow-lg backdrop-blur-sm"
                animate={{ scale: (secondsLeft ?? 99) <= 10 ? [1, 1.05, 1] : 1 }}
                transition={{ duration: 0.5, repeat: (secondsLeft ?? 99) <= 10 ? Number.POSITIVE_INFINITY : 0 }}
            >
                <div className="flex items-center gap-2">
                    <Timer className="h-5 w-5 text-accent" aria-hidden="true" />
                    <span className={`text-xl sm:text-2xl font-mono ${(secondsLeft ?? 99) <= 10 ? "text-primary" : "text-foreground"}`}>
                        {timer.paused
                            ? "Paused"
                            : `${secondsLeft ?? Math.max(0, Math.ceil(timer.durationMs / 1000))}s`}
                    </span>
                </div>
            </motion.div>
        </div>
    );
};

export const CenterTurnTimer = React.memo(CenterTurnTimerBase, (prev, next) => (
    prev.timer === next.timer && prev.actor === next.actor
));
