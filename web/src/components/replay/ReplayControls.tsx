import {ChevronFirst, ChevronLast, FastForward, Pause, Play, Rewind} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {Card, CardContent} from '@/components/ui/card';
import {cn} from '@/lib/utils';

type ReplayControlsProps = {
    snapshotIndex: number;
    maxIndex: number;
    turnNumber: number;
    isPlaying: boolean;
    speed: number;
    onTogglePlay: () => void;
    onStep: (direction: 'prev' | 'next') => void;
    onJump: (target: 'start' | 'end') => void;
    onSeek: (index: number) => void;
    onSpeedChange: (next: number) => void;
};

const speeds = [0.5, 1, 1.5, 2];

export function ReplayControls({
    snapshotIndex,
    maxIndex,
    turnNumber,
    isPlaying,
    speed,
    onTogglePlay,
    onStep,
    onJump,
    onSeek,
    onSpeedChange,
}: ReplayControlsProps) {
    return (
        <Card className="border-border bg-card/70 shadow-xl">
            <CardContent className="space-y-4 p-5">
                <div className="flex items-center justify-between">
                    <div>
                        <div className="text-xs uppercase tracking-[0.2em] text-muted-foreground">Control Panel</div>
                        <div className="text-sm font-semibold">Turn {turnNumber}</div>
                    </div>
                    <div className="rounded-full border border-accent/40 bg-accent/10 px-3 py-1 text-[11px] uppercase tracking-[0.3em] text-accent">
                        Replay
                    </div>
                </div>

                <div className="flex flex-wrap items-center gap-2">
                    <Button variant="outline" size="icon" onClick={() => onJump('start')} disabled={snapshotIndex === 0}>
                        <ChevronFirst className="h-4 w-4" />
                    </Button>
                    <Button variant="outline" size="icon" onClick={() => onStep('prev')} disabled={snapshotIndex === 0}>
                        <Rewind className="h-4 w-4" />
                    </Button>
                    <Button variant="default" size="icon" onClick={onTogglePlay} disabled={maxIndex === 0}>
                        {isPlaying ? <Pause className="h-4 w-4" /> : <Play className="h-4 w-4" />}
                    </Button>
                    <Button
                        variant="outline"
                        size="icon"
                        onClick={() => onStep('next')}
                        disabled={snapshotIndex >= maxIndex}
                    >
                        <FastForward className="h-4 w-4" />
                    </Button>
                    <Button
                        variant="outline"
                        size="icon"
                        onClick={() => onJump('end')}
                        disabled={snapshotIndex >= maxIndex}
                    >
                        <ChevronLast className="h-4 w-4" />
                    </Button>
                </div>

                <div className="space-y-2">
                    <div className="flex items-center justify-between text-xs text-muted-foreground">
                        <span>
                            Snapshot {snapshotIndex + 1} / {maxIndex + 1}
                        </span>
                        <span>{Math.round(speed * 100)}% speed</span>
                    </div>
                    <input
                        type="range"
                        min={0}
                        max={maxIndex}
                        value={snapshotIndex}
                        onChange={(event) => onSeek(Number(event.target.value))}
                        className="w-full accent-[hsl(var(--accent))]"
                    />
                </div>

                <div className="flex flex-wrap items-center gap-2">
                    {speeds.map((value) => (
                        <button
                            key={value}
                            type="button"
                            onClick={() => onSpeedChange(value)}
                            className={cn(
                                'rounded-full border border-border/60 px-3 py-1 text-xs font-semibold transition',
                                Math.abs(speed - value) < 0.01
                                    ? 'border-accent bg-accent/15 text-accent'
                                    : 'bg-background/60 text-muted-foreground hover:border-accent/50'
                            )}
                        >
                            {value}x
                        </button>
                    ))}
                </div>
            </CardContent>
        </Card>
    );
}
