import React, {useEffect, useState} from 'react';
import {useGame} from '../store/gameContext';
import {socket} from '../network/socket';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Crown, Flame, History, LogOut, Play, Plus, RefreshCcw, Trophy, Users} from 'lucide-react';
import {cn} from '@/lib/utils';
import {ReplaysPanel} from '@/components/ReplaysPanel';

export const LobbyView: React.FC = () => {
    const { state, dispatch } = useGame();
    const [nickname, setNickname] = useState(socket.getNickname() || "");
    const [avatar, setAvatar] = useState<string | null>(socket.getAvatar() || null);
    const [refreshing, setRefreshing] = useState(false);
    const [activeSection, setActiveSection] = useState<'lobbies' | 'replays'>('lobbies');

    // New local state for loading buttons
    const [joiningLobbyId, setJoiningLobbyId] = useState<string | null>(null);
    const [creatingLobby, setCreatingLobby] = useState(false);

    const hasRegistered = !!state.playerId;

    // Ensure we don't have stale lobbies if we are just registering
    useEffect(() => {
        if (!hasRegistered) {
            setRefreshing(false);
        }
    }, [hasRegistered]);

    const handleRegister = () => {
        if (!nickname) return;
        socket.register(nickname);
        // If registration fails to ACK within reason, we might want to let the user retry
        // But for now, socket logic handles re-sending hello.
    };

    const resetIdentity = () => {
        socket.resetIdentity();
        socket.disconnect();
        socket.connect();
        setNickname("");
        setAvatar(null);
        dispatch({ type: "RESET" });
    };

    const handleAvatarChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        const file = event.target.files?.[0];
        if (!file) return;
        const reader = new FileReader();
        reader.onload = () => {
            const result = typeof reader.result === "string" ? reader.result : null;
            if (!result) return;
            setAvatar(result);
            socket.setAvatar(result);
        };
        reader.readAsDataURL(file);
    };

    const refreshLobbies = React.useCallback(() => {
        if (!state.isConnected || !state.isHandshakeComplete) return;
        setRefreshing(true);
        socket.request("lobby_list").catch(err => {
            if (err?.error !== "not_connected") {
                dispatch({ type: "ERROR", error: err.error || "Failed to refresh lobbies" });
            }
        }).finally(() => {
            setTimeout(() => setRefreshing(false), 300);
        });
    }, [state.isConnected, state.isHandshakeComplete, dispatch]);

    useEffect(() => {
        if (state.isConnected && state.isHandshakeComplete && hasRegistered) {
            refreshLobbies();
        }
    }, [state.isConnected, state.isHandshakeComplete, hasRegistered, refreshLobbies]);

    const createLobby = () => {
        if (!state.isConnected || !state.isHandshakeComplete) return;
        setCreatingLobby(true);
        socket.request("lobby_create").catch(err => {
            dispatch({ type: "ERROR", error: err.error || "Failed to create lobby" });
        }).finally(() => {
            setCreatingLobby(false);
        });
    };

    const joinLobby = (lobbyId: string) => {
        if (!state.isConnected || !state.isHandshakeComplete) return;
        setJoiningLobbyId(lobbyId);
        socket.request("lobby_join", { lobby_id: lobbyId }).catch(err => {
            dispatch({ type: "ERROR", error: err.error || "Failed to join lobby" });
            setJoiningLobbyId(null);
        });
        // Note: successful join will be handled by socket message 'lobby_state', no need to clear joiningLobbyId here immediately
        // But to be safe against timeouts, we could clear it. 
        // For now, if we successfully join, the view changes, so state is unmounted/ignored.
    };

    const startGame = (lobbyId: string) => {
        if (!state.isConnected || !state.isHandshakeComplete) return;
        socket.send("lobby_start", { lobby_id: lobbyId });
    };

    if (state.currentLobby) {
        const isLeader = state.currentLobby.leaderId && state.playerId
            ? state.currentLobby.leaderId === state.playerId
            : state.currentLobby.leaderNick === nickname;

        return (
            <div className="min-h-screen bg-background p-3 sm:p-4 md:p-6 lg:p-8">
                {/* Decorative backgrounds */}
                <div className="fixed inset-0 pointer-events-none overflow-hidden">
                    <div className="absolute top-20 left-10 w-64 h-64 bg-accent/10 rounded-full blur-3xl" />
                    <div className="absolute bottom-20 right-10 w-48 h-48 bg-primary/10 rounded-full blur-3xl" />
                </div>

                <div className="relative mx-auto w-full max-w-6xl">
                    <header className="flex flex-col md:flex-row items-center justify-between gap-6 mb-10 md:mb-12">
                        <div className="text-center md:text-left">
                            <div className="flex items-center gap-3 justify-center md:justify-start">
                                <Flame className="w-8 h-8 text-accent" aria-hidden="true" />
                                <h1 className="text-3xl sm:text-4xl md:text-5xl font-serif tracking-wider text-foreground">
                                    ElMakina
                                </h1>
                            </div>
                            <p className="text-muted-foreground mt-2 text-base sm:text-lg">
                                Lobby {state.currentLobby.lobbyId} • Readying for intrigue...
                            </p>
                        </div>
                        <Button variant="outline" size="sm" onClick={resetIdentity} className="border-border text-muted-foreground hover:text-foreground">
                            <LogOut className="w-4 h-4 mr-2" /> Change Identity
                        </Button>
                    </header>

                    <div className="grid gap-6 lg:grid-cols-[1.4fr_1fr]">
                        <section className="bg-card/50 backdrop-blur-sm rounded-xl border border-border p-4 sm:p-5 lg:p-6 shadow-2xl">
                            <h2 className="text-xl sm:text-2xl font-serif text-foreground mb-5 flex items-center gap-3">
                                <Users className="w-6 h-6 text-accent" />
                                Players seated
                            </h2>
                            <div className="grid gap-3">
                                {state.currentLobby.playerNicks.map((nick, idx) => (
                                    <div
                                        key={`${nick}-${idx}`}
                                        className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-white/5 bg-white/5 px-3 py-2.5 sm:px-5 sm:py-3.5 transition-colors hover:bg-white/10"
                                    >
                                        <div className="flex items-center gap-4">
                                            <div className="w-9 h-9 sm:w-10 sm:h-10 rounded-full bg-secondary flex items-center justify-center font-serif text-base sm:text-lg border border-accent/20">
                                                {nick.charAt(0)}
                                            </div>
                                            <span className="font-medium text-base sm:text-lg">{nick}</span>
                                        </div>
                                        {nick === state.currentLobby!.leaderNick && (
                                            <span className="flex items-center gap-2 text-accent text-xs uppercase tracking-[0.2em] font-semibold bg-accent/10 px-3 py-1 rounded-full">
                                                <Crown className="h-4 w-4" /> Leader
                                            </span>
                                        )}
                                    </div>
                                ))}
                            </div>
                        </section>

                        <section className="space-y-6">
                            <div className="bg-card/50 backdrop-blur-sm rounded-xl border border-border p-4 sm:p-5 lg:p-6 shadow-2xl">
                                <h3 className="text-lg sm:text-xl font-serif text-foreground mb-4">Status</h3>
                                <div className="text-xl sm:text-2xl font-semibold text-foreground">
                                    {isLeader ? "You are hosting" : "Waiting for the host"}
                                </div>
                                <p className="mt-2 text-muted-foreground">
                                    Players ready: <span className="text-accent font-bold">{state.currentLobby.playerCount}</span> / 9
                                </p>
                                <div className="mt-8">
                                    {isLeader ? (
                                        <Button
                                            onClick={() => startGame(state.currentLobby!.lobbyId)}
                                            className="w-full h-11 sm:h-12 md:h-14 text-base sm:text-lg bg-primary text-primary-foreground hover:bg-primary/90 shadow-xl shadow-primary/20"
                                            disabled={state.currentLobby.playerCount < 2}
                                        >
                                            <Play className="w-5 h-5 mr-3" /> Start Game
                                        </Button>
                                    ) : (
                                        <div className="rounded-xl border border-white/5 bg-white/5 px-4 py-3 text-xs sm:text-sm uppercase tracking-[0.2em] text-muted-foreground text-center">
                                            The game begins shortly...
                                        </div>
                                    )}
                                </div>
                            </div>
                        </section>
                    </div>
                </div>
            </div>
        );
    }

    if (!hasRegistered) {
        return (
            <div className="min-h-screen bg-background flex flex-col items-center justify-center p-3 sm:p-4">
                <div className="fixed inset-0 pointer-events-none overflow-hidden">
                    <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[800px] bg-accent/5 rounded-full blur-[120px]" />
                </div>

                <div className="relative w-full max-w-[500px]">
                    <div className="text-center mb-10">
                        <div className="flex items-center justify-center gap-3 mb-4">
                            <Flame className="w-10 h-10 text-accent" />
                            <h1 className="text-3xl sm:text-4xl md:text-5xl font-serif tracking-widest text-foreground">EL MAKINA</h1>
                        </div>
                        <p className="text-muted-foreground text-base sm:text-lg">Enter the parlour of intrigue</p>
                    </div>

                    <div className="bg-card rounded-2xl border border-border p-5 sm:p-6 md:p-8 shadow-2xl space-y-6 sm:space-y-8">
                        <div className="flex flex-col items-center gap-6">
                            <div className="relative group">
                                <div className="w-20 h-20 sm:w-24 sm:h-24 md:w-28 md:h-28 aspect-square overflow-hidden rounded-full border-2 border-accent/20 bg-secondary/50 shadow-xl transition-all group-hover:border-accent/50">
                                    {avatar ? (
                                        <img src={avatar} alt="Profile" className="h-full w-full object-cover" />
                                    ) : (
                                        <div className="flex h-full w-full items-center justify-center text-[10px] uppercase tracking-[0.2em] text-muted-foreground text-center px-2">
                                            Upload Identity
                                        </div>
                                    )}
                                </div>
                                <label className="absolute inset-0 cursor-pointer">
                                    <input type="file" accept="image/*" onChange={handleAvatarChange} className="hidden" />
                                </label>
                            </div>

                            <div className="w-full space-y-4">
                                <div className="space-y-2">
                                    <Label htmlFor="nickname" className="text-muted-foreground">Chosen Alias</Label>
                                    <Input
                                        id="nickname"
                                        value={nickname}
                                        onChange={e => setNickname(e.target.value)}
                                        placeholder="Enter your name..."
                                        className="h-10 sm:h-11 md:h-12 text-center text-base sm:text-lg font-medium"
                                        onKeyDown={(e) => e.key === 'Enter' && handleRegister()}
                                    />
                                </div>
                                <Button className="w-full h-10 sm:h-11 md:h-12 text-base sm:text-lg bg-primary hover:bg-primary/90" onClick={handleRegister} isLoading={hasRegistered && !state.isHandshakeComplete}>
                                    Take a Seat
                                </Button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        );
    }

    const openLobbies = state.lobbies.filter(lobby => lobby.status === "open");

    return (
        <div className="flex min-h-screen bg-background">
            {/* Sidebar Navigation */}
            <aside className="w-14 sm:w-16 lg:w-56 border-r border-border bg-card/30 backdrop-blur-md flex flex-col z-50 shrink-0">
                <div className="p-4 sm:p-6 flex items-center justify-center lg:justify-start gap-4">
                    <Flame className="w-7 h-7 sm:w-8 sm:h-8 text-accent shrink-0" />
                    <span className="hidden lg:block font-serif text-xl sm:text-2xl tracking-tighter text-foreground">ElMakina</span>
                </div>

                <nav className="flex-1 px-2 sm:px-3 md:px-4 py-4 sm:py-5 space-y-3">
                    <button
                        onClick={() => setActiveSection('lobbies')}
                        className={cn(
                            "w-full flex items-center justify-center lg:justify-start gap-3 p-3 rounded-xl transition-all duration-300",
                            activeSection === 'lobbies'
                                ? "bg-accent text-accent-foreground shadow-xl shadow-accent/20"
                                : "text-muted-foreground hover:bg-muted hover:text-foreground"
                        )}
                        title="Available Lobbies"
                    >
                        <Users className="w-5 h-5" />
                        <span className="hidden lg:block font-bold uppercase tracking-widest text-xs">Lobbies</span>
                    </button>

                    <button
                        onClick={() => setActiveSection('replays')}
                        className={cn(
                            "w-full flex items-center justify-center lg:justify-start gap-3 p-3 rounded-xl transition-all duration-300",
                            activeSection === 'replays'
                                ? "bg-accent text-accent-foreground shadow-xl shadow-accent/20"
                                : "text-muted-foreground hover:bg-muted hover:text-foreground"
                        )}
                        title="Match Replays"
                    >
                        <History className="w-5 h-5" />
                        <span className="hidden lg:block font-bold uppercase tracking-widest text-xs">Replays</span>
                    </button>

                    <div className="pt-6 opacity-40">
                        <div className="border-t border-border mx-2" />
                    </div>

                    <button
                        className="w-full flex items-center justify-center lg:justify-start gap-3 p-3 rounded-xl text-muted-foreground hover:bg-muted hover:text-foreground opacity-50 cursor-not-allowed"
                        disabled
                        title="Achievements (Coming Soon)"
                    >
                        <Trophy className="w-5 h-5" />
                        <span className="hidden lg:block font-bold uppercase tracking-widest text-xs tracking-tighter">Awards</span>
                    </button>
                </nav>

                <div className="p-3 sm:p-4 border-t border-border">
                    <button
                        onClick={resetIdentity}
                        className="w-full flex items-center justify-center lg:justify-start gap-3 p-3 rounded-xl text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors"
                        title="Logout"
                    >
                        <LogOut className="w-5 h-5" />
                        <span className="hidden lg:block font-bold uppercase tracking-widest text-xs">Logout</span>
                    </button>
                </div>
            </aside>

            {/* Main Content Area */}
            <main className="flex-1 min-w-0 overflow-auto p-3 sm:p-4 md:p-6 lg:p-8">
                <div className="mx-auto w-full max-w-7xl">
                    <header className="flex flex-col md:flex-row items-center justify-between gap-6 mb-10 md:mb-12">
                        <div className="text-center md:text-left">
                            <div className="flex items-center gap-3 justify-center md:justify-start">
                                <Flame className="w-8 h-8 text-accent" aria-hidden="true" />
                                <h1 className="text-3xl sm:text-4xl md:text-5xl font-serif tracking-wider text-foreground">
                                    ElMakina
                                </h1>
                            </div>
                            <p className="text-muted-foreground mt-2 text-base sm:text-lg">
                                The Parlour of Intrigue
                            </p>
                        </div>

                        <div className="flex flex-wrap items-center gap-3">
                            <Button
                                variant="outline"
                                size="icon"
                                onClick={refreshLobbies}
                                disabled={refreshing}
                                className="h-10 w-10 sm:h-11 sm:w-11 md:h-12 md:w-12 border-border"
                            >
                                <RefreshCcw className={cn("h-5 w-5", refreshing && "animate-spin")} />
                            </Button>
                            <Button
                                onClick={createLobby}
                                className="h-10 sm:h-11 md:h-12 px-4 sm:px-5 md:px-6 text-base sm:text-lg bg-primary hover:bg-primary/90 shadow-lg shadow-primary/20 dark:text-black"
                                isLoading={creatingLobby}
                            >
                                <Plus className="w-5 h-5 mr-2" /> Create Room
                            </Button>
                        </div>
                    </header>

                    <section>
                        {activeSection === 'lobbies' ? (
                            <>
                                <div className="flex items-center gap-2 mb-8">
                                    <Crown className="w-6 h-6 text-accent" />
                                    <h2 className="text-xl sm:text-2xl font-serif text-foreground">Available Rooms</h2>
                                </div>

                                {openLobbies.length === 0 ? (
                                    <div className="bg-card/30 border border-border/50 rounded-2xl py-20 text-center">
                                        <Users className="w-10 h-10 sm:w-12 sm:h-12 text-muted-foreground/30 mx-auto mb-4" />
                                        <p className="text-muted-foreground text-base sm:text-lg">The parlour is quiet. Create a room to start the intrigue.</p>
                                    </div>
                                ) : (
                                    <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
                                        {openLobbies.map(lobby => {
                                            const isBusy = lobby.playerCount >= 6;
                                            return (
                                                <article
                                                    key={lobby.id}
                                                    className="group relative bg-card rounded-xl border border-border p-6 shadow-lg transition-all hover:border-accent/50 hover:shadow-accent/5"
                                                >
                                                    <div className="flex items-center justify-between mb-6">
                                                        <div>
                                                            <h3 className="font-serif text-xl text-foreground group-hover:text-accent transition-colors">
                                                                {lobby.leaderNick}&apos;s Room
                                                            </h3>
                                                            <span className="text-xs text-muted-foreground uppercase tracking-widest">
                                                                #{lobby.id.slice(0, 8)}
                                                            </span>
                                                        </div>
                                                        <span className={cn(
                                                            "px-3 py-1 rounded-full text-xs font-bold uppercase tracking-widest",
                                                            isBusy ? "bg-amber-400/10 text-amber-400" : "bg-emerald-400/10 text-emerald-400"
                                                        )}>
                                                            {isBusy ? "Busy" : "Joinable"}
                                                        </span>
                                                    </div>

                                                    <div className="flex flex-wrap items-center gap-4 mb-6">
                                                        <div className="flex -space-x-3">
                                                            {Array.from({ length: Math.min(lobby.playerCount, 5) }).map((_, i) => (
                                                                <div
                                                                    key={i}
                                                                    className="w-10 h-10 rounded-full bg-secondary border-2 border-card flex items-center justify-center text-xs font-serif"
                                                                >
                                                                    {i + 1}
                                                                </div>
                                                            ))}
                                                            {lobby.playerCount > 5 && (
                                                                <div className="w-10 h-10 rounded-full bg-accent/20 border-2 border-card flex items-center justify-center text-xs font-bold">
                                                                    +{lobby.playerCount - 5}
                                                                </div>
                                                            )}
                                                        </div>
                                                        <span className="text-sm font-medium text-muted-foreground">
                                                            {lobby.playerCount} / 9 Players
                                                        </span>
                                                    </div>

                                                    <Button
                                                        onClick={() => joinLobby(lobby.id)}
                                                        className="w-full h-11 bg-secondary text-secondary-foreground hover:bg-accent hover:text-accent-foreground"
                                                        isLoading={joiningLobbyId === lobby.id}
                                                    >
                                                        <Play className="w-4 h-4 mr-2" /> Enter Game
                                                    </Button>
                                                </article>
                                            );
                                        })}
                                    </div>
                                )}
                            </>
                        ) : (
                            <ReplaysPanel playerId={state.playerId} replays={state.replayHistory} />
                        )}
                    </section>
                </div>
            </main>
        </div>
    );
};
