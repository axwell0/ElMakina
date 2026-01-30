import React from 'react';
import {Button} from '@/components/ui/button';
import {cn} from '@/lib/utils';
import {Clock, UserX, Wifi, WifiOff} from 'lucide-react';
import {socket} from '@/network/socket';
import { useGameSlice } from '@/state/hooks';
import { PauseState } from '@/state/types';

const formatSeconds = (seconds: number) => {
    const clamped = Math.max(0, seconds);
    const mins = Math.floor(clamped / 60);
    const secs = clamped % 60;
    if (mins <= 0) {
        return `${secs}s`;
    }
    return `${mins}m ${secs.toString().padStart(2, '0')}s`;
};

type PausedState = Extract<PauseState, { status: 'paused' }>;

function isPausedState(pause: PauseState): pause is PausedState {
    return pause.status === 'paused';
}

export const GamePausedOverlay: React.FC = () => {
    const { game } = useGameSlice();
    const pause = game.pause;

    const [secondsLeft, setSecondsLeft] = React.useState<number>(0);

    React.useEffect(() => {
        if (!isPausedState(pause)) {
            return;
        }
        const update = () => {
            const now = Date.now();
            const remainingMs = Math.max(0, pause.deadlineMs - now);
            setSecondsLeft(Math.ceil(remainingMs / 1000));
        };
        update();
        const interval = window.setInterval(update, 250);
        return () => window.clearInterval(interval);
    }, [pause]);

    if (!isPausedState(pause)) {
        return null;
    }

    const identity = game.identity;
    const isEligible = typeof identity?.playerIndex === 'number'
        ? pause.eligibleVoters.includes(identity.playerIndex)
        : false;
    const hasVoted = typeof identity?.playerIndex === 'number'
        ? pause.kickVotes.includes(identity.playerIndex)
        : false;
    const isPausedPlayer = identity?.playerIndex === pause.pausedByIndex;
    const votesTotal = pause.eligibleVoters.length;
    const votesCast = pause.kickVotes.length;
    const isConnected = game.isConnected;

    const handleKickVote = () => {
        if (!isEligible || hasVoted) {
            return;
        }
        socket.send('kick_vote', { target_index: pause.pausedByIndex });
    };

    return (
        <div className="absolute inset-0 z-[4000] flex items-center justify-center px-3 sm:px-4 py-5 sm:py-7">
            <div className="absolute inset-0 bg-background/70 backdrop-blur-sm" />
            <div className="relative w-full max-w-2xl overflow-hidden rounded-3xl border-2 border-accent/40 bg-card/95 p-6 shadow-[0_30px_120px_rgba(0,0,0,0.45)]">
                <div className="absolute -right-16 -top-16 h-40 w-40 rounded-full bg-accent/10 blur-3xl" />
                <div className="absolute -left-12 -bottom-12 h-40 w-40 rounded-full bg-primary/10 blur-3xl" />

                <div className="relative flex flex-col gap-4">
                    <div className="flex flex-wrap items-center gap-3">
                        <div className="flex h-11 w-11 items-center justify-center rounded-full bg-accent/15">
                            <Clock className="h-6 w-6 text-accent" aria-hidden="true" />
                        </div>
                        <div>
                            <h2 className="text-lg sm:text-xl md:text-2xl font-serif text-foreground">Game Paused</h2>
                            <p className="text-sm text-muted-foreground">
                                Waiting for {pause.pausedByName} to reconnect.
                            </p>
                        </div>
                    </div>

                    <div className="rounded-2xl border border-border/60 bg-background/60 p-4">
                        <div className="flex flex-wrap items-center justify-between gap-3">
                            <div className="flex items-center gap-3">
                                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-accent/10">
                                    <UserX className="h-5 w-5 text-accent" />
                                </div>
                                <div>
                                    <div className="text-sm uppercase tracking-[0.25em] text-muted-foreground">Reconnect window</div>
                                    <div className="text-lg sm:text-xl font-semibold text-foreground">{formatSeconds(secondsLeft)}</div>
                                </div>
                            </div>
                            <div className="text-xs text-muted-foreground">
                                {votesCast}/{votesTotal} votes to kick
                            </div>
                        </div>
                    </div>

                    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                        <div className="flex items-center gap-2 text-sm text-muted-foreground">
                            {isConnected ? (
                                <Wifi className="h-4 w-4 text-emerald-400" aria-hidden="true" />
                            ) : (
                                <WifiOff className="h-4 w-4 text-amber-400" aria-hidden="true" />
                            )}
                            <span>{isConnected ? 'Connected to server' : 'Reconnecting…'}</span>
                        </div>
                        <div className="flex flex-wrap gap-2">
                            <Button
                                variant="destructive"
                                onClick={handleKickVote}
                                disabled={!isEligible || hasVoted || isPausedPlayer}
                                className={cn("h-11 px-6", hasVoted && "opacity-80")}
                            >
                                {hasVoted ? "Vote cast" : `Kick ${pause.pausedByName}`}
                            </Button>
                        </div>
                    </div>

                    {isPausedPlayer && (
                        <div className="rounded-2xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-200">
                            You were disconnected. Reconnect within {formatSeconds(secondsLeft)} to stay in the game.
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};
