import React, { useEffect } from 'react';
import { Card } from './Card';
import { useGame } from '../store/gameContext';
import { AnimatePresence, motion, useReducedMotion } from 'framer-motion';
import { playSfx } from '@/lib/audio';

export const RevealModal: React.FC = () => {
    const { state, dispatch } = useGame();
    const reveal = state.investigateResult;
    const sfxRef = React.useRef<HTMLAudioElement | null>(null);
    const prefersReducedMotion = useReducedMotion();

    useEffect(() => {
        if (typeof Audio === "undefined") return;
        const audio = new Audio("/sfx/what.mp3");
        audio.volume = 0.6;
        sfxRef.current = audio;
        return () => {
            sfxRef.current = null;
        };
    }, []);

    useEffect(() => {
        if (!reveal) return;
        const timer = setTimeout(() => {
            playSfx(sfxRef.current, state.sfxMuted);
        }, 450);
        return () => clearTimeout(timer);
    }, [reveal, state.sfxMuted]);

    if (!reveal) return null;

    return (
        <AnimatePresence>
            <motion.div
                initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
                className="fixed inset-0 z-[2000] flex flex-col items-center justify-center bg-black/80 text-white"
                onClick={() => dispatch({ type: "CLEAR_INVESTIGATE" })}
            >
                <div className="reveal-vignette" />
                <h2 className="mb-2 text-xs uppercase tracking-[0.45em] text-white/60">Investigation Result</h2>
                <p className="mb-6 text-base sm:text-lg font-semibold text-white">
                    You uncovered {reveal.targetName}'s role
                </p>
                <motion.div
                    key={`${reveal.targetName}-${reveal.role}`}
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
                            <Card role={reveal.role} isFaceUp={true} />
                        </div>
                    </div>
                </motion.div>
                <p className="mt-6 text-xs uppercase tracking-[0.35em] text-white/50">Click anywhere to close</p>
            </motion.div>
        </AnimatePresence>
    );
};
