import React from 'react';
import {useGame} from '../store/gameContext';
import {socket} from '../network/socket';
import {motion, useReducedMotion} from 'framer-motion';
import {cardImageForRole} from '@/lib/cards';
import {Check, Shuffle} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {cn} from '@/lib/utils';

export const InteractionModal: React.FC = () => {
    const { state, dispatch } = useGame();
    const prompt = state.pendingPrompt;
    const [selectedIndices, setSelectedIndices] = React.useState<number[]>([]);
    const prefersReducedMotion = useReducedMotion();

    React.useEffect(() => {
        setSelectedIndices([]);
    }, [prompt?.requestId]);

    if (!prompt || prompt.kind !== 'step') return null;

    const playerIndex = state.identity?.playerIndex;
    const canRespond = typeof playerIndex === "number";

    const toggleIndex = (idx: number, max: number) => {
        setSelectedIndices(prev => {
            if (prev.includes(idx)) {
                return prev.filter(i => i !== idx);
            }
            if (prev.length >= max) {
                return prev;
            }
            return [...prev, idx];
        });
    };

    const renderStep = (p: Extract<typeof prompt, { kind: 'step' }>) => {
        const context = (p.context || "step").toLowerCase();
        const isExchange = context.includes("exchange");
        const title = isExchange ? "Exchange Influence" : "Select Influence";
        const subtitle = `Select ${p.count} card${p.count === 1 ? "" : "s"} to keep.`;
        const returnsCount = Math.max(0, (p.options?.length ?? 0) - p.count);

        return (
            <div className="bg-card/95 backdrop-blur-md p-6 sm:p-8 rounded-2xl border-2 border-accent shadow-2xl w-full max-w-2xl max-h-[calc(100vh-4rem)] flex flex-col">
                <div className="text-center mb-6 shrink-0">
                    <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-accent/20 text-accent">
                        <Shuffle className="h-6 w-6" />
                    </div>
                    <h2 className="text-2xl sm:text-3xl font-serif text-foreground mb-2">{title}</h2>
                    <p className="text-muted-foreground text-sm sm:text-lg italic">{subtitle}</p>
                </div>

                <div className="relative px-1 sm:px-6 py-2 sm:py-6 grow overflow-y-auto min-h-0">
                    <div className="grid grid-cols-2 gap-2 sm:gap-4 sm:grid-cols-3 md:grid-cols-4">
                        {(p.options || []).map((opt, i) => {
                            const isSelected = selectedIndices.includes(i);
                            const image = cardImageForRole(opt);
                            const handleClick = () => {
                                if (!canRespond) return;
                                if (p.count > 1) {
                                    toggleIndex(i, p.count);
                                    return;
                                }
                                socket.send("step_result", [i], p.requestId);
                                dispatch({ type: "CLEAR_PROMPT" });
                            };
                            return (
                                <button
                                    key={`${opt}-${i}`}
                                    className={cn(
                                        "group relative aspect-[2/3] overflow-hidden rounded-xl border-2 border-transparent bg-secondary/20 text-white shadow-md transition-transform duration-200 hover:-translate-y-1",
                                        isSelected && "border-accent shadow-[0_0_18px_rgba(251,191,36,0.2)]",
                                        !canRespond && "opacity-60"
                                    )}
                                    onClick={handleClick}
                                    disabled={!canRespond}
                                    aria-label={opt}
                                >
                                    {image ? (
                                        <img src={image} alt={opt} className="h-full w-full object-cover" />
                                    ) : (
                                        <div className="flex h-full w-full items-center justify-center px-2 text-center text-sm font-semibold">
                                            {opt}
                                        </div>
                                    )}
                                    {isSelected && (
                                        <div className="absolute right-2 top-2 rounded-full bg-emerald-500 p-1 text-white shadow-lg z-10">
                                            <Check className="h-3 w-3" />
                                        </div>
                                    )}
                                    <div className="absolute inset-0 bg-black/20 opacity-0 transition-opacity group-hover:opacity-100" />
                                </button>
                            );
                        })}
                    </div>
                </div>

                <div className="flex flex-wrap items-center justify-between gap-3 mt-4 sm:mt-8 pt-4 sm:pt-6 border-t border-border shrink-0">
                    <div className="text-xs sm:text-sm uppercase tracking-widest text-muted-foreground font-semibold">
                        Returns {returnsCount}
                    </div>
                    <div className="flex items-center gap-3 sm:gap-4">
                        <Button
                            variant="ghost"
                            onClick={() => setSelectedIndices([])}
                            className="text-muted-foreground hover:text-foreground h-11 px-6"
                            disabled={!canRespond}
                        >
                            Reset
                        </Button>
                        {p.count > 1 && (
                            <Button
                                className="h-11 px-8 bg-primary text-primary-foreground hover:bg-primary/90 shadow-lg shadow-primary/20 text-sm sm:text-base"
                                onClick={() => {
                                    if (!canRespond) return;
                                    if (selectedIndices.length !== p.count) return;
                                    socket.send("step_result", selectedIndices, p.requestId);
                                    dispatch({ type: "CLEAR_PROMPT" });
                                }}
                                disabled={!canRespond || selectedIndices.length !== p.count}
                            >
                                <Check className="h-5 w-5 mr-2" /> Confirm
                            </Button>
                        )}
                    </div>
                </div>
            </div>
        );
    };

    return (
        <div className="fixed inset-0 z-[3100] flex items-center justify-center bg-black/70 px-2 sm:px-4 py-3 sm:py-8">
            <motion.div
                initial={prefersReducedMotion ? { opacity: 0 } : { scale: 0.9, opacity: 0 }}
                animate={prefersReducedMotion ? { opacity: 1 } : { scale: 1, opacity: 1 }}
                transition={{ duration: prefersReducedMotion ? 0.2 : 0.3 }}
                className="w-full flex justify-center"
            >
                {renderStep(prompt)}
            </motion.div>
        </div>
    );
};
