import React, { useEffect } from 'react';
import { Card } from './Card';
import { useGame } from '../store/gameContext';
import { AnimatePresence, motion, useReducedMotion } from 'framer-motion';

const REASON_LABELS: Record<string, string> = {
    'challenge_lost': 'Lost a challenge',
    'coup': 'Target of Coup',
    'assassinate': 'Assassination victim',
    'accuse': 'Accused correctly',
    'exchange': 'Exchanged away',
};

export const CardDiscardedModal: React.FC = () => {
    const { state, dispatch } = useGame();
    const discard = state.currentDiscard;
    const prefersReducedMotion = useReducedMotion();
    const autoDismissTimer = React.useRef<NodeJS.Timeout | null>(null);

    useEffect(() => {
        if (!discard) return;
        
        // Auto-dismiss after 3 seconds
        autoDismissTimer.current = setTimeout(() => {
            dispatch({ type: "DISMISS_DISCARD" });
        }, 3000);

        return () => {
            if (autoDismissTimer.current) {
                clearTimeout(autoDismissTimer.current);
            }
        };
    }, [discard, dispatch]);

    const handleClick = () => {
        if (autoDismissTimer.current) {
            clearTimeout(autoDismissTimer.current);
        }
        dispatch({ type: "DISMISS_DISCARD" });
    };

    if (!discard) return null;

    const queuePosition = state.discardQueue.findIndex(d => d === discard) + 1;
    const queueTotal = state.discardQueue.length;

    return (
        <AnimatePresence>
            <motion.div
                initial={{ opacity: 0 }} 
                animate={{ opacity: 1 }} 
                exit={{ opacity: 0 }}
                className="fixed inset-0 z-[2000] flex flex-col items-center justify-center bg-black/70 text-white"
                onClick={handleClick}
            >
                <div className="reveal-vignette" />
                
                <h2 className="mb-2 text-xs uppercase tracking-[0.45em] text-white/60">
                    {discard.isElimination ? 'Player Eliminated!' : 'Card Discarded'}
                </h2>
                
                <p className="mb-1 text-lg sm:text-xl font-semibold text-white">
                    {discard.playerName}
                </p>
                
                <p className="mb-6 text-sm text-white/70 italic">
                    {REASON_LABELS[discard.reason] || discard.reason}
                </p>

                <motion.div
                    key={`${discard.playerName}-${discard.cardRole}`}
                    className="reveal-flip"
                    initial={prefersReducedMotion ? { opacity: 0 } : { scale: 0.7, opacity: 0, y: 20 }}
                    animate={prefersReducedMotion ? { opacity: 1 } : { scale: 1, opacity: 1, y: 0 }}
                    transition={{ duration: prefersReducedMotion ? 0.2 : 0.35 }}
                >
                    <div className="reveal-flip-inner">
                        <div className="reveal-card-face reveal-front">
                            <div className="reveal-front-label">UNKNOWN</div>
                        </div>
                        <div className="reveal-card-face reveal-back">
                            <div className="reveal-shine" />
                            <div className="reveal-burst reveal-burst-anim" />
                            <div className="reveal-foil" />
                            <Card role={discard.cardRole} isFaceUp={true} />
                        </div>
                    </div>
                </motion.div>

                {queueTotal > 1 && (
                    <p className="mt-4 text-xs text-white/50">
                        Showing {queuePosition} of {queueTotal}
                    </p>
                )}

                <p className="mt-6 text-xs uppercase tracking-[0.35em] text-white/50">
                    Click anywhere to dismiss
                </p>
                
                {/* Progress bar for auto-dismiss */}
                <div className="mt-4 w-48 h-1 bg-white/20 rounded-full overflow-hidden">
                    <motion.div 
                        className="h-full bg-accent"
                        initial={{ width: "100%" }}
                        animate={{ width: "0%" }}
                        transition={{ duration: 3, ease: "linear" }}
                    />
                </div>
            </motion.div>
        </AnimatePresence>
    );
};
