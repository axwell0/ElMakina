import React from 'react';
import {AnimatePresence} from 'framer-motion';
import {Card} from '@/components/Card';
import type {HandCard} from '@/store/types';

type HandTrayProps = {
    hand: HandCard[];
    isActive: boolean;
    onHoverStart: (role: string) => void;
    onHoverEnd: (role: string) => void;
    className?: string;
};

const HandTrayBase: React.FC<HandTrayProps> = ({ hand, isActive, onHoverStart, onHoverEnd, className }) => {
    return (
        <div
            className={[
                "flex items-end gap-2 [perspective:1000px] transition-opacity",
                isActive ? "opacity-100" : "opacity-60 grayscale",
                className ?? "",
            ].join(' ')}
        >
            <AnimatePresence mode="popLayout">
                {hand.map((card) => (
                    <Card
                        key={card.id}
                        role={card.role}
                        isFaceUp={true}
                        onHoverStart={() => onHoverStart(card.role)}
                        onHoverEnd={() => onHoverEnd(card.role)}
                        className="h-16 w-12 sm:h-24 sm:w-16 md:h-28 md:w-20 rounded-lg border border-accent/40 shadow-lg"
                    />
                ))}
            </AnimatePresence>
        </div>
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
