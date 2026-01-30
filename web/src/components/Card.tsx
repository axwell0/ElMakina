import React from 'react';
import {motion, useReducedMotion} from 'framer-motion';
import {cn} from '@/lib/utils';
import {cardImageForRole} from '@/lib/cards';

interface CardProps {
    role: string;
    isFaceUp: boolean;
    onClick?: () => void;
    onHoverStart?: () => void;
    onHoverEnd?: () => void;
    layoutId?: string;
    className?: string;
}

export const Card: React.FC<CardProps> = ({ role, isFaceUp, onClick, onHoverStart, onHoverEnd, layoutId, className }) => {
    const image = cardImageForRole(role);
    const prefersReducedMotion = useReducedMotion();
    const hoverEffect = isFaceUp && !prefersReducedMotion
        ? {
            scale: 1.3,
            zIndex: 100,
            y: -40,
            rotateX: 10,
            transition: { duration: 0.1, ease: "easeOut" } // Even faster tween-based transition for perceived speed
        }
        : {};
    return (
        <motion.div
            layoutId={layoutId} // For smooth transitions between decks/hands
            className={cn(
                "flex aspect-[2/3] w-full max-w-[120px] flex-col items-center justify-center overflow-hidden rounded-xl border shadow-sm transition-[transform,box-shadow,border-color] duration-200",
                isFaceUp ? "bg-card text-card-foreground" : "bg-muted/60 text-muted-foreground",
                className
            )}
            onClick={onClick}
            onHoverStart={onHoverStart}
            onHoverEnd={onHoverEnd}
            whileHover={hoverEffect}
            initial={{ scale: 0.8, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            style={{ transformStyle: 'preserve-3d', willChange: 'transform' }}
        >
            {isFaceUp ? (
                image ? (
                    <div className="relative h-full w-full">
                        <img
                            src={image}
                            alt={role}
                            className="h-full w-full rounded-xl border border-primary/40 object-cover shadow-md"
                        />
                    </div>
                ) : (
                    <div className="text-sm uppercase tracking-[0.3em] text-muted-foreground">Unknown</div>
                )
            ) : (
                <div className="relative h-full w-full">
                    <div className="absolute inset-0 bg-gradient-to-br from-primary/15 via-muted to-primary/5" />
                    <div className="absolute inset-3 rounded-lg border border-primary/30 bg-card/40" />
                </div>
            )}
        </motion.div>
    );
};
