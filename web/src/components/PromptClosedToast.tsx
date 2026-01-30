import React from 'react';
import {useGame} from '../store/gameContext';

export const PromptClosedToast: React.FC = () => {
    const { state } = useGame();
    const reason = state.promptClosedReason;

    if (!reason) return null;

    const label = reason === 'window_closed'
        ? 'Window closed'
        : reason === 'action_submitted'
            ? 'Action submitted'
            : reason === 'step_submitted'
                ? 'Selection submitted'
                : reason;

    return (
        <div className="fixed bottom-5 left-1/2 z-[2500] -translate-x-1/2 rounded-md border border-border bg-card px-4 py-2 text-xs text-foreground shadow-sm">
            {label}
        </div>
    );
};
