'use client';

import React from 'react';
import { GameView } from '@/components/GameView';
import { MockGameProvider } from '@/mocks/MockGameProvider';
import { type MockScenario } from '@/mocks/mockState';
import { Button } from '@/components/ui/button';
import { LobbyView } from '@/components/LobbyView';

const scenarios: MockScenario[] = ['game', 'lobby', 'reveal', 'gameover', 'paused', 'assassinate', 'showcase'];

export default function ShowcaseClient() {
    const [showControls, setShowControls] = React.useState(true);
    const [scenario, setScenario] = React.useState<MockScenario>('gameover');

    return (
        <div className="relative h-screen w-full bg-background transition-colors duration-500">
            {/* Control Panel */}
            {showControls && (
                <div className="fixed top-4 right-4 z-[10001] bg-card/95 backdrop-blur-md border-2 border-accent rounded-xl p-4 shadow-2xl max-w-sm w-full">
                    <div className="flex items-center justify-between mb-3">
                        <h2 className="font-serif text-lg font-bold text-foreground">UI Showcase</h2>
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setShowControls(false)}
                            className="h-6 w-6 p-0"
                        >
                            ✕
                        </Button>
                    </div>

                    <div className="mb-4 space-y-2">
                        <label className="text-xs font-bold text-muted-foreground uppercase tracking-wide">Select Scenario</label>
                        <div className="grid grid-cols-2 gap-2">
                            {scenarios.map((s) => (
                                <Button
                                    key={s}
                                    variant={scenario === s ? "default" : "outline"}
                                    size="sm"
                                    onClick={() => setScenario(s)}
                                    className="h-8 text-xs justify-start px-2 capitalize"
                                >
                                    {s}
                                </Button>
                            ))}
                        </div>
                    </div>

                    <p className="text-xs text-muted-foreground mb-4 border-t border-border pt-3">
                        {scenario === 'gameover' && "Displaying Game Over Overlay. Note: Ensure styling matches Phase Overlay."}
                        {scenario === 'showcase' && "Displaying Standard Showcase with active challenge."}
                        {scenario === 'lobby' && "Displaying Lobby View."}
                    </p>
                </div>
            )}

            {!showControls && (
                <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setShowControls(true)}
                    className="fixed top-4 right-4 z-[10001]"
                >
                    Show Info
                </Button>
            )}

            {/* Game View with Mock Provider */}
            <MockGameProvider key={scenario} scenario={scenario}>
                {scenario === 'lobby' ? <LobbyView /> : <GameView />}
            </MockGameProvider>
        </div>
    );
}
