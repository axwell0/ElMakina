import React from 'react';
import { useGame } from '../store/gameContext';
import { AnimatePresence, motion } from 'framer-motion';
import { Crown, Home, RotateCcw, Trophy } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { ShineEffect } from '@/components/particles';

export const GameOverModal: React.FC = () => {
    const { state } = useGame();
    const result = state.gameOver;
    const me = state.identity?.playerIndex;

    const ranked = React.useMemo(() => {
        if (!result) return [];

        const players = [...state.players];
        players.sort((a, b) => {
            if (a.index === result.winnerIndex) return -1;
            if (b.index === result.winnerIndex) return 1;
            const aliveScore = (b.alive ? 1 : 0) - (a.alive ? 1 : 0);
            if (aliveScore !== 0) return aliveScore;
            return (b.coins ?? 0) - (a.coins ?? 0);
        });
        return players;
    }, [state.players, result]);

    if (!result || me == null) return null;

    const winner = state.players.find(player => player.index === result.winnerIndex) ?? null;

    const handlePlayAgain = () => {
        window.location.reload();
    };

    const handleReturnToLobby = () => {
        window.location.assign('/');
    };

    return (
        <div className="pointer-events-none absolute inset-0 z-[5000] flex items-center justify-center px-4 py-8">
            <div className="relative pointer-events-auto w-full flex items-center justify-center">
                <AnimatePresence>
                    <motion.div
                        key="game-over-modal"
                        initial={{ scale: 0.8, opacity: 0, y: 50, filter: "blur(10px)" }}
                        animate={{ scale: 1, opacity: 1, y: 0, filter: "blur(0px)" }}
                        exit={{ scale: 0.8, opacity: 0, y: 50, filter: "blur(10px)" }}
                        transition={{ type: "spring", stiffness: 300, damping: 25 }}
                        className="relative w-full max-w-lg rounded-2xl border-2 border-accent/50 bg-card/95 p-6 shadow-[0_0_60px_rgba(0,0,0,0.6)] backdrop-blur-xl flex flex-col gap-6"
                        style={{ maxHeight: '90vh', overflowY: 'auto' }}
                    >
                        {/* Shine Effect Background */}
                        <div className="absolute inset-0 overflow-hidden rounded-2xl pointer-events-none z-0">
                            <div className="absolute -top-[50%] -left-[50%] w-[200%] h-[200%] bg-[radial-gradient(circle_at_center,theme(colors.accent.DEFAULT/0.1),transparent_70%)] animate-pulse" />
                        </div>
                        <ShineEffect />

                        {/* Header Section */}
                        <div className="relative z-10 flex flex-col items-center gap-2 text-center">
                            <motion.div
                                initial={{ scale: 0, rotate: -180 }}
                                animate={{ scale: 1, rotate: 0 }}
                                transition={{ type: "spring", delay: 0.2 }}
                                className="relative flex items-center justify-center w-16 h-16 rounded-full bg-accent/10 border-2 border-accent mb-2"
                            >
                                <Crown className="w-8 h-8 text-accent drop-shadow-[0_0_10px_rgba(0,0,0,0.5)]" />
                            </motion.div>

                            <motion.h2
                                initial={{ opacity: 0, y: 10 }}
                                animate={{ opacity: 1, y: 0 }}
                                transition={{ delay: 0.3 }}
                                className="font-serif text-3xl font-bold text-foreground drop-shadow-md"
                            >
                                Victory!
                            </motion.h2>

                            <motion.p
                                initial={{ opacity: 0 }}
                                animate={{ opacity: 1 }}
                                transition={{ delay: 0.4 }}
                                className="text-sm uppercase tracking-[0.2em] text-muted-foreground"
                            >
                                The Intrigue Has Ended
                            </motion.p>
                        </div>

                        {/* Winner Spotlight */}
                        <motion.div
                            initial={{ opacity: 0, scale: 0.95 }}
                            animate={{ opacity: 1, scale: 1 }}
                            transition={{ delay: 0.5 }}
                            className="relative z-10 flex flex-col items-center gap-4 bg-secondary/30 rounded-xl p-6 border border-border"
                        >
                            <div className="relative">
                                <div className="w-24 h-24 rounded-full border-4 border-accent shadow-xl overflow-hidden bg-background">
                                    {winner?.avatar ? (
                                        <img src={winner.avatar} alt={result.winnerName} className="h-full w-full object-cover" />
                                    ) : (
                                        <div className="h-full w-full flex items-center justify-center text-4xl font-serif font-bold text-foreground">
                                            {result.winnerName.charAt(0).toUpperCase()}
                                        </div>
                                    )}
                                </div>
                                <div className="absolute -bottom-3 left-1/2 -translate-x-1/2 bg-accent text-accent-foreground text-xs font-bold px-3 py-1 rounded-full shadow-lg whitespace-nowrap">
                                    CHAMPION
                                </div>
                            </div>

                            <div className="text-center mt-2">
                                <h3 className="font-serif text-2xl text-foreground">{result.winnerName}</h3>
                                <div className="flex items-center justify-center gap-4 mt-2 text-sm text-muted-foreground">
                                    <span>{winner?.coins ?? 0} Coins</span>
                                    <span>•</span>
                                    <span>Survivor</span>
                                </div>
                            </div>
                        </motion.div>

                        {/* Standings List */}
                        <motion.div
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            transition={{ delay: 0.6 }}
                            className="relative z-10 space-y-3"
                        >
                            <div className="flex items-center gap-2 text-sm font-bold uppercase tracking-wider text-muted-foreground px-1">
                                <Trophy className="w-4 h-4" />
                                <span>Final Standings</span>
                            </div>

                            <div className="space-y-2 max-h-40 overflow-y-auto pr-1 custom-scrollbar">
                                {ranked.slice(0, 5).map((player, index) => (
                                    <div
                                        key={player.index}
                                        className={cn(
                                            "flex items-center gap-3 p-2 rounded-lg transition-colors border",
                                            player.index === result.winnerIndex
                                                ? "bg-accent/10 border-accent/30"
                                                : "bg-background/40 border-transparent hover:bg-background/60"
                                        )}
                                    >
                                        <div className={cn(
                                            "w-6 h-6 rounded-full flex items-center justify-center text-[10px] font-bold",
                                            index === 0 ? "bg-accent text-accent-foreground" : "bg-muted text-muted-foreground"
                                        )}>
                                            {index + 1}
                                        </div>

                                        <div className="w-8 h-8 rounded-full bg-secondary border border-border overflow-hidden shrink-0">
                                            {player.avatar ? (
                                                <img src={player.avatar} alt={player.name} className="h-full w-full object-cover" />
                                            ) : (
                                                <div className="h-full w-full flex items-center justify-center text-xs font-serif font-bold">
                                                    {player.name.charAt(0).toUpperCase()}
                                                </div>
                                            )}
                                        </div>

                                        <span className="flex-1 text-sm font-medium truncate">{player.name}</span>

                                        {player.index !== result.winnerIndex && (
                                            <span className="text-[10px] uppercase tracking-wider text-muted-foreground/60">
                                                {player.alive ? "Survived" : "Eliminated"}
                                            </span>
                                        )}
                                    </div>
                                ))}
                            </div>
                        </motion.div>

                        {/* Actions */}
                        <motion.div
                            initial={{ opacity: 0, y: 10 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: 0.7 }}
                            className="relative z-10 flex gap-3 pt-2"
                        >
                            <Button
                                onClick={handlePlayAgain}
                                className="flex-1 h-11 bg-primary text-primary-foreground hover:bg-primary/90 font-bold tracking-wide uppercase text-xs"
                            >
                                <RotateCcw className="w-4 h-4 mr-2" /> Play Again
                            </Button>
                            <Button
                                variant="outline"
                                onClick={handleReturnToLobby}
                                className="flex-1 h-11 border-border hover:bg-muted/50 font-bold tracking-wide uppercase text-xs"
                            >
                                <Home className="w-4 h-4 mr-2" /> Lobby
                            </Button>
                        </motion.div>
                    </motion.div>
                </AnimatePresence>
            </div>
        </div>
    );
};
