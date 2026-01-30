import React from 'react';
import {useGame} from './store/gameContext';
import {LobbyView} from './components/LobbyView';
import {GameView} from './components/GameView';
import {Button} from '@/components/ui/button';
import {Card, CardContent} from '@/components/ui/card';
import {Moon, Sun, Volume2, VolumeX} from 'lucide-react';
import {socket} from '@/network/socket';

function App() {
  const { state, dispatch } = useGame();
  const [mounted, setMounted] = React.useState(false);
  React.useEffect(() => {
    setMounted(true);
  }, []);

  // Initialize theme from state
  const isDarkMode = state.theme === "dark";

  // Sync with document element
  React.useEffect(() => {
    if (isDarkMode) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }, [isDarkMode]);

  const setTheme = React.useCallback((nextTheme: "light" | "dark") => {
    dispatch({ type: "SET_THEME", theme: nextTheme });
  }, [dispatch]);

  const hasCredentials = Boolean(socket.getNickname() || socket.hasReconnectToken());

  const [showConnectionLost, setShowConnectionLost] = React.useState(false);

  React.useEffect(() => {
    if (state.isConnected) {
      setShowConnectionLost(false);
      return;
    }
    // Delay showing the "Connecting..." screen only for connection lost, 
    // but not for initial sync if we have credentials.
    const timer = setTimeout(() => {
      setShowConnectionLost(true);
    }, 5000);
    return () => clearTimeout(timer);
  }, [state.isConnected]);

  // Mandatory loading ONLY for initial sync (where we don't have a playerId yet)
  // Reconnections for already logged-in users happen in the background to avoid flashes.
  const needsAuthSync = mounted && hasCredentials && !state.isHandshakeComplete && !state.playerId;

  // Grace period loading for when connection is actually lost for a while
  const isHardDisconnect = mounted && !state.isConnected && showConnectionLost;

  // Before mounting, we show a neutral loading state matching the server
  const shouldShowLoadingScreen = !mounted || needsAuthSync || isHardDisconnect;

  if (shouldShowLoadingScreen && state.currentLobby?.status !== 'in_game') {
    return (
      <div className="min-h-[100svh] w-full flex items-center justify-center bg-background text-foreground transition-colors duration-500">
        <Card className="min-w-[280px]">
          <CardContent className="p-6 text-center">
            <div className="text-sm font-semibold uppercase tracking-widest">
              {needsAuthSync ? "Syncing Session..." : "Connecting..."}
            </div>
            {state.error && <div className="mt-3 text-xs text-destructive">{state.error}</div>}
          </CardContent>
        </Card>
      </div>
    );
  }

  const isGameReady = Boolean(
    state.currentLobby?.status === 'in_game' &&
    state.playerId &&
    state.currentMatch &&
    state.players.length > 0
  );

  return (
    <div className="min-h-[100svh] w-full bg-background text-foreground app-shell transition-colors duration-500 relative">
      <div className="relative z-0">
        {isGameReady ? (
          <GameView />
        ) : (
          <LobbyView />
        )}
      </div>

      {!isGameReady && (
        <header className="fixed bottom-3 right-3 z-[10001] sm:bottom-6 sm:right-6 pointer-events-none" aria-label="Quick settings">
          <nav className="pointer-events-auto flex max-w-[94vw] flex-wrap items-center justify-end gap-3 rounded-full border border-border/70 bg-card/90 p-2 shadow-lg backdrop-blur">
            <Button
              onClick={() => setTheme(isDarkMode ? "light" : "dark")}
              variant="ghost"
              size="icon"
              className="relative z-10 h-10 w-10 rounded-full transition-all duration-300 hover:bg-accent/20"
              aria-label="Toggle Theme"
            >
              {isDarkMode ? (
                <Moon className="h-5 w-5 text-indigo-400" />
              ) : (
                <Sun className="h-5 w-5 text-amber-500" />
              )}
            </Button>

            <Button
              onClick={() => dispatch({ type: "SET_SFX_MUTED", muted: !state.sfxMuted })}
              variant="ghost"
              size="icon"
              className="relative z-10 h-10 w-10 rounded-full transition-all duration-300 hover:bg-accent/20"
              aria-label={state.sfxMuted ? "Unmute SFX" : "Mute SFX"}
            >
              {state.sfxMuted ? <VolumeX className="h-5 w-5" /> : <Volume2 className="h-5 w-5" />}
            </Button>
          </nav>
        </header>
      )}
    </div>
  )
}

export default App
