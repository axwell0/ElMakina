import React, {useEffect, useRef} from 'react';
import {useGame} from '../store/gameContext';
import {cn} from '@/lib/utils';

type GameLogProps = {
    className?: string;
};

export const GameLog: React.FC<GameLogProps> = ({ className }) => {
    const { state } = useGame();
    const bottomRef = useRef<HTMLDivElement>(null);
    const scrollRef = useRef<HTMLDivElement>(null);
    const shouldAutoScrollRef = useRef(true);

    const handleScroll = () => {
        const el = scrollRef.current;
        if (!el) return;
        const threshold = 24;
        const distanceFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight;
        shouldAutoScrollRef.current = distanceFromBottom < threshold;
    };

    useEffect(() => {
        if (shouldAutoScrollRef.current) {
            bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
        }
    }, [state.logs]);

    return (
        <div
            className={cn(
                "flex h-[calc(var(--scale-base)*14rem)] w-full flex-col overflow-hidden rounded-fluid-xl border border-border bg-card/90 text-foreground shadow-xl backdrop-blur container-fluid-sm",
                className
            )}
        >
            <div className="px-fluid-base py-fluid-sm text-fluid-xs font-semibold uppercase tracking-[0.35em] text-muted-foreground">Activity Log</div>
            <div
                ref={scrollRef}
                onScroll={handleScroll}
                className="flex-1 overflow-y-auto px-fluid-base pb-fluid-base text-fluid-xs"
            >
                {state.logs.length === 0 && <div className="italic text-muted-foreground">Waiting for game start...</div>}
                {state.logs.map((log, i) => (
                    <div
                        key={i}
                        className={`mb-2 border-l-2 pl-2 leading-relaxed ${log.scope === 'private'
                            ? 'border-primary text-primary'
                            : 'border-transparent text-muted-foreground'
                            }`}
                    >
                        {log.message}
                    </div>
                ))}
                <div ref={bottomRef} />
            </div>
        </div>
    );
};
