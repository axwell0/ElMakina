import React, { useCallback, useEffect, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Card } from '@/components/Card';
import type { HandCard } from '@/state/types';
import { cn } from '@/lib/utils';

type HandTrayProps = {
    hand: HandCard[];
    isActive: boolean;
    onHoverStart: (role: string) => void;
    onHoverEnd: (role: string) => void;
    className?: string;
};

/**
 * Calculate the fan angle and offset for each card
 * Cards are fanned out in an arc, with the center card being straight
 */
const calculateCardFan = (index: number, totalCards: number) => {
    if (totalCards <= 1) return { rotate: 0, xOffset: 0, yOffset: 0 };

    // Fan spread angle: max 60 degrees total (30 each side)
    const maxSpread = 60;
    const step = totalCards > 1 ? maxSpread / (totalCards - 1) : 0;
    const rotate = -maxSpread / 2 + step * index;

    // X offset to create overlapping fan effect
    // Cards overlap more as count increases
    const overlapAmount = Math.min(60, 100 / totalCards); // percentage of card width to overlap
    const xOffset = (index - (totalCards - 1) / 2) * (100 - overlapAmount);

    // Slight Y offset to create arc effect (outer cards slightly lower)
    const yOffset = Math.abs(rotate) * 0.3;

    return { rotate, xOffset, yOffset };
};

const HandTrayBase: React.FC<HandTrayProps> = ({ hand, isActive, onHoverStart, onHoverEnd, className }) => {
    const [focusedIndex, setFocusedIndex] = useState<number | null>(null);
    const [isVisible, setIsVisible] = useState(false);

    // Smooth slide-up animation on mount
    useEffect(() => {
        const timer = setTimeout(() => setIsVisible(true), 100);
        return () => clearTimeout(timer);
    }, []);

    // Keyboard navigation
    const handleKeyDown = useCallback((e: KeyboardEvent) => {
        if (hand.length === 0) return;

        switch (e.key) {
            case 'ArrowLeft':
                e.preventDefault();
                setFocusedIndex(prev => {
                    const newIndex = prev === null ? hand.length - 1 : Math.max(0, prev - 1);
                    onHoverStart(hand[newIndex].role);
                    return newIndex;
                });
                break;
            case 'ArrowRight':
                e.preventDefault();
                setFocusedIndex(prev => {
                    const newIndex = prev === null ? 0 : Math.min(hand.length - 1, prev + 1);
                    onHoverStart(hand[newIndex].role);
                    return newIndex;
                });
                break;
            case 'Escape':
                setFocusedIndex(null);
                onHoverEnd('');
                break;
            case 'Home':
                e.preventDefault();
                if (hand.length > 0) {
                    setFocusedIndex(0);
                    onHoverStart(hand[0].role);
                }
                break;
            case 'End':
                e.preventDefault();
                if (hand.length > 0) {
                    setFocusedIndex(hand.length - 1);
                    onHoverStart(hand[hand.length - 1].role);
                }
                break;
        }
    }, [hand, onHoverStart, onHoverEnd]);

    useEffect(() => {
        window.addEventListener('keydown', handleKeyDown);
        return () => window.removeEventListener('keydown', handleKeyDown);
    }, [handleKeyDown]);

    // Clear focus when hand changes
    useEffect(() => {
        setFocusedIndex(null);
    }, [hand.length]);

    if (hand.length === 0) return null;

    return (
        <motion.div
            initial={{ y: 100, opacity: 0 }}
            animate={{
                y: isVisible ? 0 : 100,
                opacity: isVisible ? 1 : 0
            }}
            transition={{
                type: "spring",
                stiffness: 100,
                damping: 20,
                staggerChildren: 0.05
            }}
            className={cn(
                "fixed bottom-0 left-1/2 z-[1600] w-full max-w-4xl -translate-x-1/2",
                "flex items-end justify-center",
                "px-4 pb-2 sm:pb-4",
                isActive ? "opacity-100" : "opacity-70 grayscale",
                className
            )}
            role="region"
            aria-label="Your hand cards"
            aria-live="polite"
        >
            {/* Container for the fanned cards */}
            <div
                className="relative flex items-end justify-center"
                style={{
                    height: 'clamp(6rem, 20vh, 10rem)',
                    width: `clamp(${hand.length * 3}rem, ${hand.length * 12}vw, ${hand.length * 6}rem)`
                }}
            >
                <AnimatePresence mode="popLayout">
                    {hand.map((card, index) => {
                        const { rotate, xOffset, yOffset } = calculateCardFan(index, hand.length);
                        const isFocused = focusedIndex === index;

                        return (
                            <motion.div
                                key={card.id}
                                layout
                                initial={{ scale: 0.8, opacity: 0, y: 50 }}
                                animate={{
                                    scale: 1,
                                    opacity: 1,
                                    y: 0,
                                    rotate: isFocused ? 0 : rotate,
                                    x: xOffset,
                                    zIndex: isFocused ? 100 : index
                                }}
                                exit={{ scale: 0.8, opacity: 0, y: 50 }}
                                transition={{
                                    type: "spring",
                                    stiffness: 200,
                                    damping: 25
                                }}
                                className={cn(
                                    "absolute bottom-0",
                                    "w-[clamp(3rem,10vw,5rem)]",
                                    "h-[clamp(4.5rem,15vw,7.5rem)]",
                                    "origin-bottom",
                                    "transition-transform duration-200",
                                    isFocused && "z-50 scale-110"
                                )}
                                style={{
                                    transform: `translateX(${xOffset}%) rotate(${rotate}deg) translateY(${yOffset}px)`,
                                    transformOrigin: 'bottom center'
                                }}
                                onMouseEnter={() => {
                                    setFocusedIndex(index);
                                    onHoverStart(card.role);
                                }}
                                onMouseLeave={() => {
                                    setFocusedIndex(null);
                                    onHoverEnd(card.role);
                                }}
                                onClick={() => {
                                    setFocusedIndex(index);
                                    onHoverStart(card.role);
                                }}
                                role="button"
                                tabIndex={0}
                                aria-label={`Card ${index + 1} of ${hand.length}: ${card.role}`}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter' || e.key === ' ') {
                                        e.preventDefault();
                                        setFocusedIndex(index);
                                        onHoverStart(card.role);
                                    }
                                }}
                            >
                                <Card
                                    role={card.role}
                                    isFaceUp={true}
                                    onHoverStart={() => {}}
                                    onHoverEnd={() => {}}
                                    className="h-full w-full rounded-xl border-2 border-accent/60 shadow-2xl"
                                />
                            </motion.div>
                        );
                    })}
                </AnimatePresence>

                {/* Keyboard hint */}
                {hand.length > 0 && (
                    <div className="absolute -top-8 left-1/2 -translate-x-1/2 whitespace-nowrap">
                        <span className="text-[10px] uppercase tracking-wider text-muted-foreground/60 bg-background/80 px-2 py-1 rounded-full">
                            Use ← → arrow keys to navigate cards
                        </span>
                    </div>
                )}
            </div>
        </motion.div>
    );
};

export const HandTray = React.memo(HandTrayBase, (prev, next) => {
    if (prev.isActive !== next.isActive) return false;
    if (prev.hand.length !== next.hand.length) return false;
    for (let i = 0; i < prev.hand.length; i++) {
        if (prev.hand[i]?.id !== next.hand[i]?.id || prev.hand[i]?.role !== next.hand[i]?.role) return false;
    }
    return prev.onHoverStart === next.onHoverStart && prev.onHoverEnd === next.onHoverEnd;
});
