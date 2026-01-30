import React, { useEffect, useState } from 'react';
import { useGame } from '../store/gameContext';
import { CoinParticle } from './CoinParticle';
import { playSfx } from '@/lib/audio';

export const CoinManager: React.FC = () => {
    const { state } = useGame();
    const [particles, setParticles] = useState<{ id: string; x: number; y: number }[]>([]);
    const sfxRef = React.useRef<HTMLAudioElement | null>(null);
    const lastCoinsRef = React.useRef<Map<number, number | null>>(new Map());

    useEffect(() => {
        if (typeof Audio === "undefined") {
            return;
        }
        const audio = new Audio("/sfx/coin.mp3");
        audio.volume = 0.4;
        sfxRef.current = audio;
        return () => {
            sfxRef.current = null;
        };
    }, []);

    useEffect(() => {
        const enemies = state.players.filter(p => p.index !== state.identity?.playerIndex);

        state.players.forEach((player) => {
            const lastCoins = lastCoinsRef.current.get(player.index);
            if (player.coins === null) {
                lastCoinsRef.current.set(player.index, null);
                return;
            }
            if (lastCoins === undefined) {
                lastCoinsRef.current.set(player.index, player.coins);
                return;
            }
            if (player.coins > (lastCoins ?? 0)) {
                // Find position
                let x = 0, y = 0;

                if (player.index === state.identity?.playerIndex) {
                    // Self: approx bottom left
                    x = 60;
                    y = window.innerHeight - 60;
                } else {
                    // Enemy: Find index in enemies array to calculate position
                    const enemyIdx = enemies.findIndex(e => e.name === player.name);
                    if (enemyIdx !== -1) {
                        const leftPos = (enemyIdx + 1) * (100 / (enemies.length + 1));
                        x = (leftPos / 100) * window.innerWidth;
                        y = 0.08 * window.innerHeight + 50; // offset from top
                    }
                }

                if (x > 0) {
                    const id = `coin-${Date.now()}-${Math.random()}`;
                    setParticles(prev => [...prev, { id, x, y }]);
                    playSfx(sfxRef.current, state.sfxMuted);
                }
            }
            lastCoinsRef.current.set(player.index, player.coins);
        });
    }, [state.players, state.identity?.playerIndex, state.sfxMuted]);

    const removeParticle = (id: string) => {
        setParticles(prev => prev.filter(p => p.id !== id));
    };

    return (
        <>
            {particles.map(p => (
                <CoinParticle key={p.id} id={p.id} x={p.x} y={p.y} onComplete={removeParticle} />
            ))}
        </>
    );
};
