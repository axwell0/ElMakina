import React from 'react';
import {motion} from 'framer-motion';

interface CoinParticleProps {
    id: string;
    x: number;
    y: number;
    onComplete: (id: string) => void;
}

export const CoinParticle: React.FC<CoinParticleProps> = ({ id, x, y, onComplete }) => {
    return (
        <motion.div
            initial={{ opacity: 1, x, y, scale: 0.5 }}
            animate={{
                y: y - 100,
                opacity: 0,
                scale: 1.5,
                rotate: 360
            }}
            transition={{ duration: 1, ease: "easeOut" }}
            onAnimationComplete={() => onComplete(id)}
            style={{
                position: 'fixed',
                top: 0,
                left: 0,
                fontSize: 32,
                zIndex: 200,
                pointerEvents: 'none',
                textShadow: '0 0 15px rgba(255, 215, 0, 0.8)'
            }}
        >
            💰
        </motion.div>
    );
};
