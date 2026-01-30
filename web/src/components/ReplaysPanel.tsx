'use client';

import React from 'react';
import {useRouter} from 'next/navigation';
import {History, Search} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {Card, CardContent} from '@/components/ui/card';
import {Input} from '@/components/ui/input';
import {ScrollArea} from '@/components/ui/scroll-area';
import {cn} from '@/lib/utils';
import type {ReplayEntry} from '@/store/types';

const dateFormatter = new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
});

interface ReplaysPanelProps {
    playerId: string | null;
    replays: ReplayEntry[];
}

export function ReplaysPanel({ playerId, replays }: ReplaysPanelProps) {
    const router = useRouter();
    const [query, setQuery] = React.useState('');
    const [manualId, setManualId] = React.useState('');
    const [manualError, setManualError] = React.useState<string | null>(null);

    const eligibleReplays = React.useMemo(() => {
        if (!playerId) return [];
        return replays
            .filter((entry) => entry.participantIds.includes(playerId))
            .sort((a, b) => b.endedAt - a.endedAt);
    }, [playerId, replays]);

    const filteredReplays = React.useMemo(() => {
        if (!query) return eligibleReplays;
        const needle = query.toLowerCase();
        return eligibleReplays.filter((entry) => {
            const names = entry.playerNames.join(' ').toLowerCase();
            return entry.matchId.toLowerCase().includes(needle) || names.includes(needle);
        });
    }, [eligibleReplays, query]);

    const openReplay = React.useCallback(
        (matchId: string) => {
            if (!playerId) {
                setManualError('Connect to view replays.');
                return;
            }
            const isEligible = eligibleReplays.some((entry) => entry.matchId === matchId);
            if (!isEligible) {
                setManualError('You can only view replays you participated in.');
                return;
            }
            setManualError(null);
            router.push(`/replay/${matchId}?viewer_id=${encodeURIComponent(playerId)}` as any);
        },
        [eligibleReplays, playerId, router]
    );

    return (
        <div className="space-y-6">
            <header className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                <div>
                    <div className="flex items-center gap-3">
                        <History className="h-6 w-6 text-accent" />
                        <h2 className="text-xl sm:text-2xl font-serif text-foreground">Match Replays</h2>
                    </div>
                    <p className="text-sm text-muted-foreground">
                        Revisit past matches you played. Private information stays private.
                    </p>
                </div>
                <div className="flex w-full max-w-full sm:max-w-sm items-center gap-2">
                    <div className="relative w-full">
                        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                        <Input
                            value={query}
                            onChange={(event) => setQuery(event.target.value)}
                            placeholder="Search by player or match ID"
                            className="h-10 pl-9"
                        />
                    </div>
                </div>
            </header>

            <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]">
                <section className="space-y-4">
                    <Card className="border-border bg-card/60 shadow-xl">
                        <CardContent className="space-y-4 p-4 sm:p-5 md:p-6">
                            <div className="flex flex-wrap items-center justify-between gap-3">
                                <div>
                                    <h3 className="text-lg font-semibold">Your archive</h3>
                                    <p className="text-xs text-muted-foreground">Only matches you joined appear here.</p>
                                </div>
                                <div className="flex w-full sm:w-auto items-center gap-2">
                                    <Input
                                        value={manualId}
                                        onChange={(event) => {
                                            setManualId(event.target.value);
                                            if (manualError) {
                                                setManualError(null);
                                            }
                                        }}
                                        placeholder="Match ID"
                                        className="h-9 w-full sm:max-w-[180px]"
                                    />
                                    <Button
                                        size="sm"
                                        variant="outline"
                                        onClick={() => {
                                            const trimmed = manualId.trim();
                                            if (trimmed) {
                                                openReplay(trimmed);
                                            }
                                        }}
                                    >
                                        Load
                                    </Button>
                                </div>
                            </div>
                            {manualError && (
                                <div className="rounded-lg border border-destructive/40 bg-destructive/10 p-3 text-xs text-destructive">
                                    {manualError}
                                </div>
                            )}
                            {filteredReplays.length === 0 ? (
                                <div className="rounded-xl border border-dashed border-border/70 bg-background/40 p-6 text-center text-sm text-muted-foreground">
                                    No replays found yet. Finish a match to see it here.
                                </div>
                            ) : (
                                <ScrollArea className="h-[360px] sm:h-[420px] pr-2">
                                    <div className="space-y-3">
                                        {filteredReplays.map((entry) => {
                                            return (
                                                <button
                                                    key={entry.matchId}
                                                    type="button"
                                                    onClick={() => openReplay(entry.matchId)}
                                                    className={cn(
                                                        'w-full rounded-xl border border-border/70 bg-background/60 p-4 text-left transition hover:border-accent/60',
                                                        'hover:bg-accent/10'
                                                    )}
                                                >
                                                    <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                                                        <div>
                                                            <div className="text-xs uppercase tracking-[0.2em] text-muted-foreground">
                                                                {entry.matchId.slice(0, 8)}
                                                            </div>
                                                            <div className="text-base font-semibold text-foreground">
                                                                {entry.playerNames.join(' vs ')}
                                                            </div>
                                                        </div>
                                                        <div className="text-xs text-muted-foreground">
                                                            {dateFormatter.format(new Date(entry.endedAt))}
                                                        </div>
                                                    </div>
                                                    <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                                                        {typeof entry.winnerIndex === 'number' && (
                                                            <span className="rounded-full bg-emerald-400/10 px-2 py-0.5 text-emerald-400">
                                                                Winner: {entry.winnerName ?? `Player ${entry.winnerIndex + 1}`}
                                                            </span>
                                                        )}
                                                        <span className="rounded-full bg-secondary/50 px-2 py-0.5">
                                                            Lobby {entry.lobbyId}
                                                        </span>
                                                    </div>
                                                </button>
                                            );
                                        })}
                                    </div>
                                </ScrollArea>
                            )}
                        </CardContent>
                    </Card>
                </section>

                <section className="space-y-4">
                    <Card className="border-border bg-card/60 shadow-xl">
                        <CardContent className="space-y-3 p-4 sm:p-5 md:p-6">
                            <div>
                                <h3 className="text-lg font-semibold">Replay Viewer</h3>
                                <p className="text-xs text-muted-foreground">
                                    Replays now open in a full game-board view with controls and timeline.
                                </p>
                            </div>
                            <div className="rounded-xl border border-dashed border-border/70 bg-background/40 p-6 text-center text-sm text-muted-foreground">
                                Pick a replay on the left to open the full board.
                            </div>
                        </CardContent>
                    </Card>
                </section>
            </div>
        </div>
    );
}
