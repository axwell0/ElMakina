import React from 'react';
import {useGame} from '../store/gameContext';
import {socket} from '../network/socket';
import {actionLabel} from '@/lib/actions';
import {cardImageForRole} from '@/lib/cards';
import {cn} from '@/lib/utils';
import {Button} from '@/components/ui/button';
import {Coins, Crown, Eye, HandCoins, Repeat, Shield, Skull, Sword, Target} from 'lucide-react';

type ActionPanelProps = {
    className?: string;
};

const ACTION_ORDER = [
    'income', 'foreign_aid', 'coup',
    'businesswoman', 'tax', 'investigate',
    'accuse', 'assassinate', 'steal', 'exchange'
];

export const ActionPanel: React.FC<ActionPanelProps> = ({ className }) => {
    const { state, dispatch } = useGame();
    const prompt = state.pendingPrompt;

    // UI State for multi-step actions
    const [selectionStep, setSelectionStep] = React.useState<'none' | 'target' | 'role'>('none');
    const [selectedAction, setSelectedAction] = React.useState<string | null>(null);
    const [selectedTarget, setSelectedTarget] = React.useState<number | null>(null);
    const [inFlight, setInFlight] = React.useState(false);

    const allRoles = React.useMemo(
        () => (state.roles.length > 0
            ? state.roles
            : ["Businesswoman", "TaxCollector", "Policewoman", "Colonel", "Terrorist", "Thief", "Politician"]),
        [state.roles]
    );


    const ACTION_ICONS: Record<string, React.ElementType> = React.useMemo(() => ({
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
    }), []);

    const ACTION_META: Record<string, { cost?: number; role?: string }> = React.useMemo(() => ({
        income: { cost: 0 },
        foreign_aid: { cost: 0 },
        coup: { cost: 7 },
        businesswoman: { cost: 0, role: "Businesswoman" },
        tax: { cost: 0, role: "TaxCollector" },
        investigate: { cost: 0, role: "Policewoman" },
        accuse: { cost: 4, role: "Colonel" },
        assassinate: { cost: 3, role: "Terrorist" },
        steal: { cost: 0, role: "Thief" },
        exchange: { cost: 0, role: "Politician" },
    }), []);

    // Reset selection state when prompt changes
    React.useEffect(() => {
        setSelectionStep('none');
        setSelectedAction(null);
        setSelectedTarget(null);
        setInFlight(false);
    }, [prompt?.requestId]);

    React.useEffect(() => {
        if (!state.targeting?.active) return;
        if (state.targeting.actionId !== "accuse") return;
        if (selectionStep !== "target") return;
        if (state.targeting.selectedTarget == null) return;
        setSelectedAction("accuse");
        setSelectedTarget(state.targeting.selectedTarget);
        setSelectionStep("role");
    }, [state.targeting, selectionStep]);

    const canAct = prompt?.kind === 'action';
    const hasTaxablePlayer = React.useMemo(() => {
        return state.players.some((player) => (player.alive ?? true) && (player.coins ?? 0) >= 7);
    }, [state.players]);

    const initiateAction = (actionId: string) => {
        if (!canAct) return;
        if (!state.identity) return;
        if (inFlight) return;
        if (!prompt) return;

        console.debug(`[ActionPanel] Initiating: ${actionId}`);

        const needsTarget = ['coup', 'steal', 'accuse', 'assassinate', 'investigate'].includes(actionId);

        if (needsTarget) {
            setSelectedAction(actionId);
            setSelectionStep('target');
            dispatch({ type: "SET_TARGETING", actionId, requestId: prompt.requestId });
        } else {
            socket.send("action", { id: actionId, source_index: state.identity?.playerIndex }, prompt.requestId);
            setInFlight(true);
            dispatch({ type: "CLEAR_PROMPT" });

            // Safety timeout to prevent permanent hang
            setTimeout(() => {
                setInFlight(false);
            }, 5000);
        }
    };

    const handleRoleSelect = (role: string) => {
        if (!state.identity || !prompt) return;
        if (inFlight) return;
        if (selectedTarget === null) return;

        socket.send("action", {
            id: selectedAction,
            source_index: state.identity?.playerIndex,
            target_index: selectedTarget,
            guess: role
        }, prompt.requestId);

        setInFlight(true);
        setSelectionStep('none');
        dispatch({ type: "CLEAR_PROMPT" });
        dispatch({ type: "CLEAR_TARGETING" });

        // Safety timeout
        setTimeout(() => {
            setInFlight(false);
        }, 5000);
    };

    const cancelSelection = () => {
        setSelectionStep('none');
        setSelectedAction(null);
        setSelectedTarget(null);
        dispatch({ type: "CLEAR_TARGETING" });
    };

    const displayActions = React.useMemo(() => {
        const allMainActions = Object.keys(ACTION_ICONS).filter(
            k => !k.startsWith('block_') && k !== 'escape' && k !== 'tax_business_woman'
        );

        return allMainActions.sort((a, b) => {
            const idxA = ACTION_ORDER.indexOf(a);
            const idxB = ACTION_ORDER.indexOf(b);
            if (idxA !== -1 && idxB !== -1) return idxA - idxB;
            if (idxA !== -1) return -1;
            if (idxB !== -1) return 1;
            return a.localeCompare(b);
        });
    }, [ACTION_ICONS]);

    const getActionStatus = (action: string) => {
        if (!canAct) return { enabled: false, reason: "Not your turn" };

        const isAllowed = prompt.allowedActions.includes(action);
        if (action === "tax" && !hasTaxablePlayer) {
            return { enabled: false, reason: "No player has 7+ coins" };
        }
        if (isAllowed) return { enabled: true, reason: null };

        const meta = ACTION_META[action];
        const coinCount = state.players.find(p => p.index === state.identity?.playerIndex)?.coins ?? 0;

        if (meta?.cost && coinCount < meta.cost) {
            return { enabled: false, reason: `Need ${meta.cost} coins` };
        }

        if (action === "coup" && coinCount < 7) {
            return { enabled: false, reason: "Need 7 coins" };
        }

        return { enabled: false, reason: "Not available" };
    };

    return (
        <div className={cn("w-full max-w-2xl md:w-auto", className)}>
            {selectionStep !== 'none' && (
                <div className="mb-3 flex items-center justify-center gap-3 text-xs font-semibold uppercase tracking-[0.35em] text-muted-foreground">
                    <span>
                        {selectionStep === 'target' &&
                            (selectedAction ? `Select target for ${actionLabel(selectedAction)}` : "Select target")}
                        {selectionStep === 'role' && (selectedAction === "accuse" ? "Accuse: Choose Card" : "Choose Card")}
                    </span>
                    <button onClick={cancelSelection} className="text-xs font-semibold text-muted-foreground hover:text-foreground">
                        Cancel
                    </button>
                </div>
            )}

            {selectionStep === 'role' && (
                <div className="mx-auto w-full max-w-2xl rounded-2xl border-2 border-accent bg-card/95 shadow-2xl backdrop-blur">
                    <div className="relative border-b border-border px-4 sm:px-6 py-3 sm:py-4 text-center">
                        <div className="text-xs font-semibold uppercase tracking-[0.35em] text-muted-foreground">
                            Accuse Role
                        </div>
                        <div className="mt-1 text-xl sm:text-2xl font-bold uppercase tracking-wide text-foreground font-serif">
                            Choose a Card
                        </div>
                    </div>
                    <div className="relative px-4 sm:px-6 py-4 sm:py-6">
                        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                            {allRoles.map(role => {
                                const image = cardImageForRole(role);
                                return (
                                    <button
                                        key={role}
                                        type="button"
                                        className={cn(
                                            "group relative aspect-[2/3] overflow-hidden rounded-xl border-2 border-transparent bg-primary/80 text-primary-foreground shadow-md transition-transform hover:-translate-y-1",
                                            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60",
                                        )}
                                        onClick={() => handleRoleSelect(role)}
                                        disabled={inFlight}
                                        aria-label={role}
                                    >
                                        {image ? (
                                            <img src={image} alt={role} className="h-full w-full object-cover" />
                                        ) : (
                                            <div className="flex h-full w-full items-center justify-center px-2 text-center text-xs font-semibold">
                                                {role}
                                            </div>
                                        )}
                                        <div className="absolute inset-0 bg-black/20 opacity-0 transition-opacity group-hover:opacity-100" />
                                    </button>
                                );
                            })}
                        </div>
                    </div>
                </div>
            )}

            {selectionStep === 'target' && (
                <div className="mx-auto flex w-fit items-center gap-2 rounded-full border border-border bg-card/80 px-3 py-1.5 sm:px-4 sm:py-2 text-xs font-semibold uppercase tracking-[0.35em] text-muted-foreground">
                    Tap a player on the table
                </div>
            )}

            {selectionStep === 'none' && (
                <div className="w-full">
                    <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 gap-1.5">
                        {displayActions.map((action) => {
                            const Icon = ACTION_ICONS[action] ?? Coins;
                            const meta = ACTION_META[action];
                            const label = actionLabel(action);
                            const { enabled, reason } = getActionStatus(action);

                            return (
                                <div key={action} className="relative group">
                                    <Button
                                        onClick={() => initiateAction(action)}
                                        disabled={inFlight || !enabled}
                                        variant="outline"
                                        className={cn(
                                            "h-auto min-h-[40px] w-full px-0.5 py-1 bg-card/80 border border-accent/40 hover:bg-accent/20 hover:border-accent flex flex-col items-center justify-center gap-0.5 relative",
                                            "transition-all duration-150 backdrop-blur-sm",
                                            !enabled && "opacity-40 grayscale cursor-not-allowed"
                                        )}
                                        aria-label={label}
                                    >
                                        <Icon className="h-3.5 w-3.5 shrink-0 text-accent" aria-hidden="true" />
                                        <span className="text-[7px] font-bold uppercase tracking-wide text-center leading-tight w-full">
                                            {label}
                                        </span>
                                        {meta?.cost ? (
                                            <span className="absolute top-0.5 right-0.5 text-[8px] font-bold text-accent">
                                                {meta.cost}💰
                                            </span>
                                        ) : null}
                                    </Button>

                                    {/* Tooltip for disabled actions */}
                                    {!enabled && (
                                        <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-1 w-max px-1.5 py-0.5 bg-popover text-popover-foreground text-[8px] rounded shadow-md opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-50">
                                            {reason}
                                        </div>
                                    )}
                                </div>
                            );
                        })}
                    </div>
                </div>
            )}
        </div>
    );
};
