'use client';

import React from 'react';
import {useRouter} from 'next/navigation';
import {fetchReplay, type ReplayPayload} from '@/lib/replay';
import type {GameIdentity, HandCard, PlayerSnapshot} from '@/store/types';
import {ReplayGameView} from '@/components/replay/ReplayGameView';
import {ReplayControls} from '@/components/replay/ReplayControls';
import {Card, CardContent} from '@/components/ui/card';
import {ScrollArea} from '@/components/ui/scroll-area';
import {Button} from '@/components/ui/button';
import {cardImageForRole} from '@/lib/cards';
import {cn} from '@/lib/utils';
import {Coins, Crown, Eye, HandCoins, Repeat, Shield, Skull, Sword, Target,} from 'lucide-react';

type ReplayLoadState =
    | { status: 'idle' }
    | { status: 'loading' }
    | { status: 'error'; message: string }
    | { status: 'success'; payload: ReplayPayload };

type ReplayActionCue = {
    text: string;
    seq: number;
    role?: string;
    actionId?: string;
    sourceIndex?: number;
    targetIndex?: number;
};

type ActionPayload = {
    ID?: string;
    id?: string;
    SourceIndex?: number;
    TargetIndex?: number;
    Guess?: string;
};

type TurnResolvedPayload = {
    main_action?: ActionPayload;
    counter_results?: Array<{ Action?: ActionPayload }>;
    turn_number?: number;
    pass?: boolean;
    player_index?: number;
    action?: ActionPayload;
};

const toRecord = (value: unknown): Record<string, unknown> => {
    if (value && typeof value === 'object' && !Array.isArray(value)) {
        return value as Record<string, unknown>;
    }
    return {};
};

const normalizeAvatar = (value?: string | null) => {
    if (!value) return '';
    return value.replace(/\s+/g, '');
};

type TurnTimelineEntry = {
    turn: number;
    snapshotIndex: number;
    main?: { playerIndex: number; role?: string; actionId?: string };
    counters: Array<{ playerIndex: number; role?: string; actionId?: string }>;
};

const humanize = (value: string) =>
    value
        .replace(/_/g, ' ')
        .replace(/-/g, ' ')
        .replace(/\b\w/g, (char) => char.toUpperCase());

const ROLE_BY_ACTION: Record<string, string> = {
    businesswoman: 'Businesswoman',
    tax: 'TaxCollector',
    investigate: 'Policewoman',
    accuse: 'Colonel',
    assassinate: 'Terrorist',
    steal: 'Thief',
    exchange: 'Politician',
    block_steal: 'Thief',
    block_investigate: 'Policewoman',
    block_terrorist: 'Terrorist',
    block_foreign_aid: 'TaxCollector',
    tax_business_woman: 'TaxCollector',
};

const ACTION_LABELS: Record<string, string> = {
    income: 'Income',
    foreign_aid: 'Foreign Aid',
    coup: 'Coup',
    businesswoman: 'Businesswoman',
    tax: 'Tax',
    investigate: 'Investigate',
    accuse: 'Accuse',
    assassinate: 'Assassinate',
    steal: 'Steal',
    exchange: 'Exchange',
    block_steal: 'Block Steal',
    block_investigate: 'Block Investigate',
    block_terrorist: 'Block Assassinate',
    block_foreign_aid: 'Block Foreign Aid',
    tax_business_woman: 'Tax Businesswoman',
    escape: 'Escape',
};

const ACTION_ICONS: Record<string, React.ElementType> = {
    income: Coins,
    foreign_aid: HandCoins,
    coup: Sword,
    businesswoman: Coins,
    tax: Crown,
    investigate: Eye,
    accuse: Target,
    assassinate: Skull,
    steal: HandCoins,
    exchange: Repeat,
    block_steal: Shield,
    block_investigate: Shield,
    block_terrorist: Shield,
    block_foreign_aid: Shield,
    tax_business_woman: Crown,
    escape: Repeat,
};

const describeAction = (action: ActionPayload | undefined, nameForIndex: (index?: number) => string) => {
    if (!action || typeof action !== 'object') return 'an action';
    const id = typeof action.ID === 'string' ? action.ID : typeof action.id === 'string' ? action.id : '';
    const label = id ? humanize(id) : 'an action';
    const sourceIndex = typeof action.SourceIndex === 'number' ? action.SourceIndex : undefined;
    const targetIndex = typeof action.TargetIndex === 'number' ? action.TargetIndex : undefined;
    const guess = typeof action.Guess === 'string' ? action.Guess : undefined;
    const sourceName = sourceIndex != null ? nameForIndex(sourceIndex) : 'A player';
    const targetName = targetIndex != null ? nameForIndex(targetIndex) : undefined;
    if (guess && targetName) {
        return `${sourceName} accuses ${targetName} of being ${guess}.`;
    }
    if (targetName) {
        return `${sourceName} uses ${label} on ${targetName}.`;
    }
    return `${sourceName} uses ${label}.`;
};

const extractActionCue = (
    payload: ReplayPayload,
    currentSeq: number,
    previousSeq: number
): ReplayActionCue | null => {
    const participantsByIndex = new Map<number, ReplayPayload['participants'][number]>();
    payload.participants.forEach((p) => participantsByIndex.set(p.PlayerIndex, p));
    const nameForIndex = (index?: number) => {
        if (typeof index !== 'number') return 'Unknown player';
        const participant = participantsByIndex.get(index);
        return participant?.Nick || `Player ${index + 1}`;
    };

    const candidates = payload.events.filter(
        (event) =>
            event.Seq > previousSeq &&
            event.Seq <= currentSeq &&
            (event.Type === 'action_submitted' || event.Type === 'counter_response' || event.Type === 'turn_resolved')
    );
    const latest = candidates[candidates.length - 1];
    if (!latest) return null;
    const payloadObj = toRecord(latest.Payload);
    if (latest.Type === 'turn_resolved') {
        const main = (payloadObj as TurnResolvedPayload).main_action;
        const actionId = typeof main?.ID === 'string' ? main.ID : typeof main?.id === 'string' ? main.id : '';
        const role = actionId ? ROLE_BY_ACTION[actionId] : undefined;
        const sourceIndex = typeof main?.SourceIndex === 'number' ? main.SourceIndex : undefined;
        const targetIndex = typeof main?.TargetIndex === 'number' ? main.TargetIndex : undefined;
        return { text: describeAction(main, nameForIndex), seq: latest.Seq, role, actionId: actionId || undefined, sourceIndex, targetIndex };
    }
    if (latest.Type === 'counter_response' && payloadObj.pass === true) {
        const playerIndex = typeof payloadObj.player_index === 'number' ? payloadObj.player_index : undefined;
        return { text: `${nameForIndex(playerIndex)} passes on countering.`, seq: latest.Seq };
    }
    const action = (payloadObj as TurnResolvedPayload).action;
    const actionId = typeof action?.ID === 'string' ? action.ID : typeof action?.id === 'string' ? action.id : '';
    const role = actionId ? ROLE_BY_ACTION[actionId] : undefined;
    const sourceIndex = typeof action?.SourceIndex === 'number' ? action.SourceIndex : undefined;
    const targetIndex = typeof action?.TargetIndex === 'number' ? action.TargetIndex : undefined;
    return { text: describeAction(action, nameForIndex), seq: latest.Seq, role, actionId: actionId || undefined, sourceIndex, targetIndex };
};

const buildTurnTimeline = (
    payload: ReplayPayload,
    maxSeq: number
): TurnTimelineEntry[] => {
    const participantsByIndex = new Map<number, ReplayPayload['participants'][number]>();
    const participantsById = new Map<string, ReplayPayload['participants'][number]>();
    payload.participants.forEach((p) => {
        participantsByIndex.set(p.PlayerIndex, p);
        participantsById.set(p.PlayerID, p);
    });


    const snapshotIndexByTurn = new Map<number, number>();
    payload.snapshots.forEach((snap, index) => {
        if (snap.Seq > maxSeq) return;
        const turnNumber = snap.Payload?.TurnNumber;
        if (typeof turnNumber === 'number') {
            snapshotIndexByTurn.set(turnNumber, index);
        }
    });

    const turnMap = new Map<number, TurnTimelineEntry>();
    const getTurnEntry = (turn: number) => {
        if (!turnMap.has(turn)) {
            turnMap.set(turn, {
                turn,
                snapshotIndex: snapshotIndexByTurn.get(turn) ?? 0,
                counters: [],
            });
        }
        return turnMap.get(turn)!;
    };

    payload.events
        .filter((event) => event.Seq <= maxSeq)
        .forEach((event) => {
            if (event.Type !== 'turn_resolved') return;
            const payloadObj = toRecord(event.Payload) as TurnResolvedPayload;
            const turnNumber = typeof payloadObj.turn_number === 'number' ? payloadObj.turn_number : 1;
            const entry = getTurnEntry(turnNumber);
            const main = payloadObj.main_action;
            if (main) {
                const sourceIndex = typeof main.SourceIndex === 'number' ? main.SourceIndex : -1;
                const actionId = typeof main.ID === 'string' ? main.ID : typeof main.id === 'string' ? main.id : '';
                entry.main = {
                    playerIndex: sourceIndex,
                    role: actionId ? ROLE_BY_ACTION[actionId] : undefined,
                    actionId: actionId || undefined,
                };
            }
            const counterResults = payloadObj.counter_results;
            if (Array.isArray(counterResults)) {
                counterResults.forEach((result) => {
                    const action = result.Action;
                    if (!action) return;
                    const sourceIndex = typeof action.SourceIndex === 'number' ? action.SourceIndex : -1;
                    const actionId = typeof action.ID === 'string' ? action.ID : typeof action.id === 'string' ? action.id : '';
                    entry.counters.push({
                        playerIndex: sourceIndex,
                        role: actionId ? ROLE_BY_ACTION[actionId] : undefined,
                        actionId: actionId || undefined,
                    });
                });
            }
        });

    return Array.from(turnMap.values()).sort((a, b) => a.turn - b.turn);
};

type ReplayClientProps = {
    matchId: string;
    viewerId: string | null;
    initialPayload: ReplayPayload | null;
    initialError: string | null;
};

function ReplayRoute({ matchId, viewerId, initialPayload, initialError }: ReplayClientProps) {
    const router = useRouter();
    const [loadState, setLoadState] = React.useState<ReplayLoadState>(() => {
        if (initialPayload) return { status: 'success', payload: initialPayload };
        if (initialError) return { status: 'error', message: initialError };
        return { status: 'idle' };
    });
    const [snapshotIndex, setSnapshotIndex] = React.useState(0);
    const [isPlaying, setIsPlaying] = React.useState(false);
    const [speed, setSpeed] = React.useState(1);
    const [actionCue, setActionCue] = React.useState<{ id: string; text: string; role?: string; actionId?: string; sourceIndex?: number; targetIndex?: number } | null>(null);
    const prevSeqRef = React.useRef<number>(-1);
    const [resolvedViewerId, setResolvedViewerId] = React.useState<string | null>(viewerId);
    const [theme, setTheme] = React.useState<'light' | 'dark'>('dark');

    React.useEffect(() => {
        if (typeof window === 'undefined') return;
        const stored = localStorage.getItem('elmakina.theme') as 'light' | 'dark' | null;
        setTheme(stored ?? 'dark');
    }, []);

    React.useEffect(() => {
        if (typeof window === 'undefined') return;
        if (resolvedViewerId) return;
        const stored = localStorage.getItem('elmakina.playerId');
        setResolvedViewerId(stored);
    }, [resolvedViewerId]);

    React.useEffect(() => {
        if (!matchId) return;
        if (!resolvedViewerId) {
            setLoadState({ status: 'error', message: 'Missing viewer_id. Open this replay from inside the game.' });
            return;
        }
        let active = true;
        setLoadState({ status: 'loading' });
        fetchReplay(matchId, resolvedViewerId)
            .then((payload) => {
                if (!active) return;
                const authorized = payload.participants.some((participant) => participant.PlayerID === resolvedViewerId);
                if (!authorized) {
                    setLoadState({ status: 'error', message: 'You are not eligible to view this replay.' });
                    return;
                }
                setLoadState({ status: 'success', payload });
                setSnapshotIndex(0);
                setIsPlaying(false);
            })
            .catch((err) => {
                if (!active) return;
                const message = err instanceof Error ? err.message : 'Failed to load replay.';
                setLoadState({ status: 'error', message });
            });
        return () => {
            active = false;
        };
    }, [matchId, resolvedViewerId]);

    React.useEffect(() => {
        if (typeof document === 'undefined') return;
        if (theme === 'dark') {
            document.documentElement.classList.add('dark');
        } else {
            document.documentElement.classList.remove('dark');
        }
    }, [theme]);

    const payload = loadState.status === 'success' ? loadState.payload : null;
    const snapshots = payload?.snapshots ?? [];
    const maxIndex = Math.max(0, snapshots.length - 1);
    const activeSnapshot = snapshots[snapshotIndex];
    const snapshotState = activeSnapshot?.Payload;
    const activeSeq = activeSnapshot?.Seq ?? 0;

    React.useEffect(() => {
        if (!isPlaying) return;
        if (snapshotIndex >= maxIndex) {
            setIsPlaying(false);
            return;
        }
        const intervalMs = Math.max(200, Math.round(1000 / speed));
        const timer = window.setTimeout(() => {
            setSnapshotIndex((prev) => Math.min(maxIndex, prev + 1));
        }, intervalMs);
        return () => window.clearTimeout(timer);
    }, [isPlaying, maxIndex, snapshotIndex, speed]);

    React.useEffect(() => {
        if (!payload) return;
        const prevSeq = prevSeqRef.current;
        const cue = extractActionCue(payload, activeSeq, prevSeq);
        let timer: number | undefined;
        if (cue) {
            const cueId = `cue-${cue.seq}-${Date.now()}`;
            setActionCue({ id: cueId, text: cue.text, role: cue.role, actionId: cue.actionId, sourceIndex: cue.sourceIndex, targetIndex: cue.targetIndex });
            timer = window.setTimeout(() => {
                setActionCue(null);
            }, 1200);
        }
        prevSeqRef.current = activeSeq;
        return () => {
            if (timer) {
                window.clearTimeout(timer);
            }
        };
    }, [activeSeq, payload]);

    const handleSeek = (nextIndex: number) => {
        setIsPlaying(false);
        setSnapshotIndex(Math.max(0, Math.min(maxIndex, nextIndex)));
    };

    const participantsByIndex = React.useMemo(() => {
        const map = new Map<number, ReplayPayload['participants'][number]>();
        payload?.participants.forEach((participant) => {
            map.set(participant.PlayerIndex, participant);
        });
        return map;
    }, [payload]);

    const identity: GameIdentity | null = payload
        ? {
              playerId: payload.viewer_player_id,
              playerIndex: payload.viewer_index,
              playerNames: payload.participants
                  .sort((a, b) => a.PlayerIndex - b.PlayerIndex)
                  .map((p) => p.Nick || `Player ${p.PlayerIndex + 1}`),
          }
        : null;

    const viewerLabel = identity?.playerNames?.[identity.playerIndex] ?? null;

    const players: PlayerSnapshot[] = React.useMemo(() => {
        if (!snapshotState) return [];
        return snapshotState.Players.map((player, index) => {
            const participant = participantsByIndex.get(index);
            return {
                index,
                name: player.Name || participant?.Nick || `Player ${index + 1}`,
                alive: (player.Hand?.length ?? 0) > 0,
                coins: player.Coins ?? 0,
                cardCount: player.Hand?.length ?? 0,
                avatar: normalizeAvatar(participant?.Avatar) || undefined,
            };
        });
    }, [participantsByIndex, snapshotState]);

    const handsByIndex = React.useMemo(() => {
        if (!snapshotState) return {};
        const map: Record<number, string[]> = {};
        snapshotState.Players.forEach((player, index) => {
            map[index] = (player.Hand ?? []).map((card) => card.Role);
        });
        return map;
    }, [snapshotState]);

    const hand: HandCard[] = React.useMemo(() => {
        if (!snapshotState || payload == null) return [];
        const viewer = snapshotState.Players?.[payload.viewer_index];
        if (!viewer || !Array.isArray(viewer.Hand)) return [];
        return viewer.Hand.map((card) => ({
            id: `replay-${card.ID}`,
            role: card.Role,
        }));
    }, [payload, snapshotState]);

        const turnNumber = snapshotState?.TurnNumber ?? 1;

    return (
        <div className="relative min-h-[100svh] w-full bg-background text-foreground">
            <div className="pointer-events-none absolute inset-0 bg-sky-500/5" />
            <main className="relative mx-auto flex w-full max-w-[1600px] flex-1 flex-col gap-6 px-3 sm:px-4 py-4 sm:py-6 lg:px-6">
                {loadState.status === 'loading' && (
                    <Card className="border-border bg-card/60">
                        <CardContent className="p-4 sm:p-5 md:p-6 text-sm text-muted-foreground">Loading replay...</CardContent>
                    </Card>
                )}

                {loadState.status === 'error' && (
                    <Card className="border-destructive/40 bg-destructive/10">
                        <CardContent className="space-y-3 p-4 sm:p-5 md:p-6 text-sm text-destructive">
                            <div>{loadState.message}</div>
                            <Button variant="outline" onClick={() => router.push('/')}>
                                Back to Lobby
                            </Button>
                        </CardContent>
                    </Card>
                )}

                {payload && snapshotState && (
                    <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_380px] xl:grid-cols-[minmax(0,1fr)_420px] lg:items-start">
                        <ReplayGameView
                            players={players}
                            identity={identity}
                            activePlayerIndex={snapshotState.CurrentPlayerIndex ?? null}
                            hand={hand}
                            handsByIndex={handsByIndex}
                            turnNumber={turnNumber}
                            onBack={() => router.push('/')}
                            actionCue={actionCue}
                            viewerLabel={viewerLabel ?? undefined}
                        />

                        <div className="flex flex-col gap-4">
                            <ReplayControls
                                snapshotIndex={snapshotIndex}
                                maxIndex={maxIndex}
                                turnNumber={turnNumber}
                                isPlaying={isPlaying}
                                speed={speed}
                                onTogglePlay={() => setIsPlaying((prev) => !prev)}
                                onStep={(direction) => {
                                    setIsPlaying(false);
                                    setSnapshotIndex((prev) => {
                                        if (direction === 'prev') return Math.max(0, prev - 1);
                                        return Math.min(maxIndex, prev + 1);
                                    });
                                }}
                                onJump={(target) => {
                                    setIsPlaying(false);
                                    setSnapshotIndex(target === 'start' ? 0 : maxIndex);
                                }}
                                onSeek={handleSeek}
                                onSpeedChange={(next) => {
                                    setSpeed(next);
                                }}
                            />

                            <Card className="border-border bg-card/70 shadow-xl">
                                <CardContent className="space-y-3 p-5">
                                    <div className="text-xs uppercase tracking-[0.2em] text-muted-foreground">
                                        Turn Timeline
                                    </div>
                                    <ScrollArea className="h-[260px] pr-2">
                                        <div className="space-y-3 text-xs text-muted-foreground">
                                            {buildTurnTimeline(payload, activeSnapshot?.Seq ?? Number.MAX_SAFE_INTEGER).map((turn) => {
                                                const mainParticipant = participantsByIndex.get(turn.main?.playerIndex ?? -1);
                                                const mainAvatar = normalizeAvatar(mainParticipant?.Avatar);
                                                const mainName = mainParticipant?.Nick || 'Player';
                                                const mainCard = turn.main?.role ? cardImageForRole(turn.main.role) : null;
                                                const mainActionId = turn.main?.actionId;
                                                const MainIcon = mainActionId ? ACTION_ICONS[mainActionId] : undefined;
                                                const mainActionLabel = mainActionId ? (ACTION_LABELS[mainActionId] || humanize(mainActionId)) : 'Action';
                                                return (
                                                    <button
                                                        key={`turn-${turn.turn}`}
                                                        type="button"
                                                        onClick={() => setSnapshotIndex(turn.snapshotIndex)}
                                                        className={cn(
                                                            'w-full rounded-xl border border-border/60 bg-background/40 p-3 text-left transition hover:border-accent/60 hover:bg-accent/10',
                                                            snapshotIndex === turn.snapshotIndex && 'border-accent bg-accent/15'
                                                        )}
                                                    >
                                                        <div className="text-[11px] font-semibold uppercase tracking-[0.3em] text-foreground">
                                                            Turn {turn.turn}
                                                        </div>
                                                        <div className="mt-2 flex items-center gap-2">
                                                            <div className="h-8 w-8 rounded-full border border-accent/40 bg-secondary overflow-hidden flex items-center justify-center text-[10px] font-semibold text-foreground">
                                                                {mainAvatar ? (
                                                                    <img src={mainAvatar} alt={mainName} className="h-full w-full object-cover" />
                                                                ) : (
                                                                    mainName.charAt(0).toUpperCase()
                                                                )}
                                                            </div>
                                                            <div className="text-foreground">
                                                                Main
                                                            </div>
                                                            {mainCard ? (
                                                                <div className="h-12 w-8 overflow-hidden rounded-md border border-border bg-card">
                                                                    <img src={mainCard} alt={turn.main?.role ?? 'Main action'} className="h-full w-full object-cover" />
                                                                </div>
                                                            ) : (
                                                                <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
                                                                    {MainIcon ? <MainIcon className="h-4 w-4 text-accent" /> : null}
                                                                    <span>{mainActionLabel}</span>
                                                                </div>
                                                            )}
                                                        </div>
                                                        {turn.counters.length > 0 && (
                                                            <div className="mt-2 space-y-2">
                                                                {turn.counters.map((counter, idx) => {
                                                                    const counterParticipant = participantsByIndex.get(counter.playerIndex);
                                                                    const counterAvatar = normalizeAvatar(counterParticipant?.Avatar);
                                                                    const counterName = counterParticipant?.Nick || 'Player';
                                                                    const counterCard = counter.role ? cardImageForRole(counter.role) : null;
                                                                    const counterActionId = counter.actionId;
                                                                    const CounterIcon = counterActionId ? ACTION_ICONS[counterActionId] : undefined;
                                                                    const counterActionLabel = counterActionId ? (ACTION_LABELS[counterActionId] || humanize(counterActionId)) : 'Action';
                                                                    return (
                                                                        <div key={`counter-${turn.turn}-${idx}`} className="flex items-center gap-2 text-foreground">
                                                                            <div className="h-7 w-7 rounded-full border border-accent/30 bg-secondary overflow-hidden flex items-center justify-center text-[9px] font-semibold">
                                                                                {counterAvatar ? (
                                                                                    <img src={counterAvatar} alt={counterName} className="h-full w-full object-cover" />
                                                                                ) : (
                                                                                    counterName.charAt(0).toUpperCase()
                                                                                )}
                                                                            </div>
                                                                            <div>Counter</div>
                                                                            {counterCard ? (
                                                                                <div className="h-10 w-7 overflow-hidden rounded-md border border-border bg-card">
                                                                                    <img src={counterCard} alt={counter.role ?? 'Counter action'} className="h-full w-full object-cover" />
                                                                                </div>
                                                                            ) : (
                                                                                <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
                                                                                    {CounterIcon ? <CounterIcon className="h-4 w-4 text-accent" /> : null}
                                                                                    <span>{counterActionLabel}</span>
                                                                                </div>
                                                                            )}
                                                                        </div>
                                                                    );
                                                                })}
                                                            </div>
                                                        )}
                                                    </button>
                                                );
                                            })}
                                        </div>
                                    </ScrollArea>
                                </CardContent>
                            </Card>
                        </div>
                    </div>
                )}

                {payload && !snapshotState && (
                    <Card className="border-border bg-card/60">
                        <CardContent className="p-4 sm:p-5 md:p-6 text-sm text-muted-foreground">
                            No snapshots recorded for this match yet.
                        </CardContent>
                    </Card>
                )}
            </main>
        </div>
    );
}

export default function ReplayClient(props: ReplayClientProps) {
    return <ReplayRoute {...props} />;
}
