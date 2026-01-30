"use client";

import {cn} from "@/lib/utils";
import {useState} from "react";
import {useGame} from "@/lib/game-context";
import type {ActionType, Player, Role} from "@/lib/game-types";
import {ACTION_ICONS, GAME_ACTIONS} from "@/lib/game-types";
import {PlayerCard} from "@/components/player-card";
import {ActionOverlay} from "@/components/action-overlay";
import {Button} from "@/components/ui/button";
import {ArrowLeft, Coins, Flame, Shuffle, Timer} from "lucide-react";
import {AnimatePresence, motion} from "framer-motion";

const MOCK_PLAYERS: Player[] = [
  {
    id: "1",
    name: "Duke Maximilian",
    avatar: "",
    coins: 3,
    cards: ["Duke", "Assassin"],
    revealedCards: [],
    isActive: true,
    isEliminated: false,
  },
  {
    id: "2",
    name: "Lady Isabella",
    avatar: "",
    coins: 5,
    cards: ["Captain"],
    revealedCards: ["Contessa"],
    isActive: false,
    isEliminated: false,
  },
  {
    id: "3",
    name: "Baron Von Richter",
    avatar: "",
    coins: 2,
    cards: ["Ambassador", "Duke"],
    revealedCards: [],
    isActive: false,
    isEliminated: false,
  },
  {
    id: "4",
    name: "Countess Elara",
    avatar: "",
    coins: 7,
    cards: ["Assassin"],
    revealedCards: ["Captain"],
    isActive: false,
    isEliminated: false,
  },
  {
    id: "5",
    name: "Sir Edmund",
    avatar: "",
    coins: 0,
    cards: [],
    revealedCards: ["Contessa", "Ambassador"],
    isActive: false,
    isEliminated: true,
  },
];

type ActionState = "idle" | "selecting-target" | "exchange-animation" | "waiting-response" | "resolving";

export function GameBoard() {
  const {
    setScreen,
    addLogEntry,
    setWinner,
    pendingAction,
    setPendingAction,
    pendingCounter,
    setPendingCounter,
  } = useGame();

  const [players] = useState<Player[]>(MOCK_PLAYERS);
  const [currentPlayerId] = useState("1");
  const [activePlayerId] = useState("1");
  const [timeLeft] = useState(30);

  // New interactive action states
  const [actionState, setActionState] = useState<ActionState>("idle");
  const [selectedAction, setSelectedAction] = useState<ActionType | null>(null);
  const [selectablePlayerIds, setSelectablePlayerIds] = useState<string[]>([]);
  const [exchangeCards, setExchangeCards] = useState<Role[]>([]);
  const [showOverlay, setShowOverlay] = useState(false);
  const [overlayMessage, setOverlayMessage] = useState("");

  const currentPlayer = players.find((p) => p.id === currentPlayerId);
  const isMyTurn = currentPlayerId === activePlayerId;
  const alivePlayers = players.filter((p) => !p.isEliminated);

  const handleActionSelect = (action: ActionType) => {
    setSelectedAction(action);
    const actionData = GAME_ACTIONS.find((a) => a.type === action);

    // Handle exchange animation
    if (action === "exchange") {
      setActionState("exchange-animation");
      // Simulate drawing cards
      const mockDrawnCards: Role[] = ["Duke", "Captain"];
      setTimeout(() => {
        setExchangeCards(mockDrawnCards);
      }, 500);
      setTimeout(() => {
        handleActionExecute(action);
      }, 3000);
      return;
    }

    // If action requires target, enter target selection mode
    if (actionData?.targetRequired) {
      setActionState("selecting-target");
      setSelectablePlayerIds(alivePlayers.filter((p) => p.id !== currentPlayerId).map((p) => p.id));
    } else {
      handleActionExecute(action, undefined);
    }
  };

  const handlePlayerSelect = (playerId: string) => {
    if (actionState !== "selecting-target" || !selectedAction) return;

    const target = players.find((p) => p.id === playerId);
    setActionState("idle");
    setSelectablePlayerIds([]);
    handleActionExecute(selectedAction, target);
  };

  const handleActionExecute = (action: ActionType, target?: Player) => {
    const actionData = GAME_ACTIONS.find((a) => a.type === action);
    const claimedRole = actionData?.role;

    setPendingAction({
      type: action,
      actor: currentPlayer!,
      target,
      claimedRole,
    });

    addLogEntry(
      `${currentPlayer?.name} declares ${action}${target ? ` targeting ${target.name}` : ""}${claimedRole ? ` as ${claimedRole}` : ""}`
    );

    // Show overlay and wait for responses
    setShowOverlay(true);
    setActionState("waiting-response");
  };

  const handleChallenge = () => {
    setShowOverlay(false);
    setActionState("resolving");
    if (pendingAction) {
      addLogEntry(`${players.find((p) => p.id === "2")?.name} challenges!`);
      setOverlayMessage("Challenge in progress...");
      setTimeout(() => {
        addLogEntry("Challenge resolved.");
        setPendingAction(null);
        setActionState("idle");
        setOverlayMessage("");
      }, 2000);
    }
  };

  const handleBlock = () => {
    setShowOverlay(false);
    if (pendingAction) {
      const blocker = players.find((p) => p.id === "2")!;
      const actionData = GAME_ACTIONS.find((a) => a.type === pendingAction.type);
      setPendingCounter({
        blocker,
        claimedRole: actionData?.blockableBy?.[0] || "Contessa",
        originalAction: pendingAction,
      });
      addLogEntry(`${blocker.name} blocks with ${actionData?.blockableBy?.[0] || "Contessa"}`);
      setActionState("waiting-response");
      setTimeout(() => {
        setShowOverlay(true);
      }, 500);
    }
  };

  const handlePass = () => {
    setShowOverlay(false);
    if (pendingAction) {
      addLogEntry(`${pendingAction.type} succeeds!`);
      setPendingAction(null);
      setPendingCounter(null);
    }
    setActionState("idle");
  };

  const handleEndGame = () => {
    const winner = alivePlayers[0];
    setWinner(winner);
    setScreen("gameover");
  };

  return (
    <div className="min-h-screen bg-background flex flex-col overflow-hidden">
      {/* Ambient lighting effects */}
      <div className="fixed inset-0 pointer-events-none overflow-hidden">
        <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-accent/5 rounded-full blur-3xl" />
        <div className="absolute bottom-1/4 right-1/4 w-80 h-80 bg-primary/5 rounded-full blur-3xl" />
      </div>

      {/* Header Bar */}
      <header className="relative z-10 bg-card/80 backdrop-blur-sm border-b border-border px-4 py-3">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setScreen("lobby")}
            className="text-muted-foreground hover:text-foreground"
          >
            <ArrowLeft className="w-4 h-4 mr-2" aria-hidden="true" />
            Leave Game
          </Button>

          <div className="flex items-center gap-2">
            <Flame className="w-5 h-5 text-accent" aria-hidden="true" />
            <h1 className="font-serif text-xl text-foreground">ElMakina</h1>
          </div>

          <Button
            variant="outline"
            size="sm"
            onClick={handleEndGame}
            className="border-primary text-primary hover:bg-primary/10 bg-transparent"
          >
            End Game (Demo)
          </Button>
        </div>
      </header>

      {/* Main Game Area */}
      <main className="relative flex-1 flex items-center justify-center p-2 lg:p-4">
        <div className="w-full h-full max-w-7xl flex flex-col lg:flex-row gap-4 items-stretch">
          {/* Center - Enlarged Game Table */}
          <div className="flex-1 flex flex-col gap-4 items-center justify-center">
            {/* Circular Table - Much Larger */}
            <div
              className="relative w-full aspect-square max-w-3xl rounded-full bg-secondary/30 border-4 border-border shadow-2xl"
              style={{
                backgroundImage:
                  "radial-gradient(ellipse at center, var(--secondary) 0%, var(--card) 70%)",
              }}
              role="region"
              aria-label="Game table with players"
            >
              {/* Exchange Animation Overlay */}
              <AnimatePresence>
                {actionState === "exchange-animation" && (
                  <motion.div
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    className="absolute inset-0 flex items-center justify-center z-40"
                  >
                    <motion.div
                      className="bg-card/95 backdrop-blur-md p-8 rounded-xl border-2 border-accent shadow-2xl"
                      initial={{ scale: 0.5, rotate: -180 }}
                      animate={{ scale: 1, rotate: 0 }}
                      transition={{ type: "spring", duration: 0.8 }}
                    >
                      <h3 className="font-serif text-xl mb-4 text-foreground text-center">
                        Exchanging Cards
                      </h3>
                      <div className="flex gap-4">
                        {exchangeCards.map((role, index) => (
                          <motion.div
                            key={index}
                            initial={{ x: index === 0 ? -100 : 100, opacity: 0, rotateY: 90 }}
                            animate={{ x: 0, opacity: 1, rotateY: 0 }}
                            transition={{ delay: 0.3 + index * 0.2, type: "spring" }}
                            className="w-24 h-32 rounded-lg bg-primary/80 border-2 border-accent flex items-center justify-center shadow-xl cursor-pointer hover:scale-105 transition-transform"
                          >
                            <span className="text-sm font-serif text-primary-foreground text-center px-2">
                              {role}
                            </span>
                          </motion.div>
                        ))}
                      </div>
                      <p className="text-sm text-muted-foreground mt-4 text-center">
                        Select 2 cards to keep...
                      </p>
                    </motion.div>
                  </motion.div>
                )}
              </AnimatePresence>

              {/* Action Overlay in center */}
              {showOverlay && pendingAction && (
                <ActionOverlay
                  action={pendingAction.type}
                  actor={pendingAction.actor}
                  target={pendingAction.target}
                  claimedRole={pendingAction.claimedRole}
                  onChallenge={handleChallenge}
                  onBlock={handleBlock}
                  onPass={handlePass}
                  canBlock={GAME_ACTIONS.find((a) => a.type === pendingAction.type)?.blockableBy !== undefined}
                  showResponses={actionState === "waiting-response"}
                  message={overlayMessage}
                />
              )}

              {/* Counter Overlay */}
              {showOverlay && pendingCounter && (
                <ActionOverlay
                  action={pendingCounter.originalAction.type}
                  actor={pendingCounter.blocker}
                  claimedRole={pendingCounter.claimedRole}
                  onChallenge={() => {
                    setShowOverlay(false);
                    addLogEntry("Counter challenged!");
                    setTimeout(() => {
                      setPendingCounter(null);
                      setPendingAction(null);
                      setActionState("idle");
                    }, 2000);
                  }}
                  onPass={() => {
                    setShowOverlay(false);
                    addLogEntry("Block succeeds!");
                    setPendingCounter(null);
                    setPendingAction(null);
                    setActionState("idle");
                  }}
                  showResponses={actionState === "waiting-response"}
                  message="Blocking action..."
                />
              )}

              {/* Central Info */}
              <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 flex flex-col items-center gap-3 z-10">
                <div className="w-32 h-32 md:w-40 md:h-40 rounded-full bg-card/90 backdrop-blur-sm border-2 border-accent shadow-xl flex flex-col items-center justify-center">
                  <Flame
                    className="w-8 h-8 text-accent mb-2"
                    aria-hidden="true"
                  />
                  <span className="text-sm text-muted-foreground">
                    {alivePlayers.length} Players
                  </span>
                  <span className="text-2xl font-serif text-foreground">
                    Round 5
                  </span>
                </div>

                {/* Timer below center */}
                <motion.div
                  className="bg-card/90 backdrop-blur-sm rounded-lg border border-border px-6 py-3 shadow-lg"
                  animate={{ scale: timeLeft <= 10 ? [1, 1.05, 1] : 1 }}
                  transition={{ duration: 0.5, repeat: timeLeft <= 10 ? Number.POSITIVE_INFINITY : 0 }}
                >
                  <div className="flex items-center gap-2">
                    <Timer className="w-5 h-5 text-accent" aria-hidden="true" />
                    <span
                      className={`text-2xl font-mono ${timeLeft <= 10 ? "text-primary" : "text-foreground"}`}
                    >
                      {timeLeft}s
                    </span>
                  </div>
                </motion.div>
              </div>

              {/* Player Positions */}
              {players.map((player, index) => (
                <PlayerCard
                  key={player.id}
                  player={player}
                  isActive={player.id === activePlayerId}
                  isCurrentPlayer={player.id === currentPlayerId}
                  position={index}
                  totalPlayers={players.length}
                  selectable={selectablePlayerIds.includes(player.id)}
                  onSelect={() => handlePlayerSelect(player.id)}
                />
              ))}
            </div>

            {/* Current Player Card Area - Compact */}
            {currentPlayer && (
              <motion.div
                className="w-full max-w-3xl bg-card/90 backdrop-blur-sm rounded-lg border-2 border-accent p-4 shadow-xl"
                layout
              >
                <div className="flex flex-wrap items-center justify-between gap-4">
                  {/* Player Info */}
                  <div className="flex items-center gap-3">
                    <div className="w-12 h-12 rounded-full bg-secondary flex items-center justify-center text-xl font-serif border-2 border-accent">
                      {currentPlayer.name.charAt(0)}
                    </div>
                    <div>
                      <p className="font-medium text-foreground">
                        {currentPlayer.name}
                      </p>
                      <div className="flex items-center gap-1">
                        <Coins
                          className="w-4 h-4 text-accent"
                          aria-hidden="true"
                        />
                        <span className="text-sm font-serif text-accent">
                          {currentPlayer.coins} coins
                        </span>
                      </div>
                    </div>
                  </div>

                  {/* Cards */}
                  <div className="flex gap-2">
                    {currentPlayer.cards.map((role, index) => (
                      <motion.div
                        key={index}
                        className="w-16 h-20 rounded bg-primary/80 border border-accent/50 flex items-center justify-center shadow-lg cursor-pointer"
                        title={role}
                        whileHover={{ scale: 1.05, y: -5 }}
                        whileTap={{ scale: 0.95 }}
                      >
                        <span className="text-xs font-serif text-primary-foreground text-center px-1">
                          {role}
                        </span>
                      </motion.div>
                    ))}
                    {currentPlayer.revealedCards.map((role, index) => (
                      <div
                        key={`revealed-${index}`}
                        className="w-16 h-20 rounded bg-muted border border-border flex items-center justify-center opacity-50"
                        title={`${role} (revealed)`}
                      >
                        <span className="text-xs text-muted-foreground text-center px-1">
                          {role}
                        </span>
                      </div>
                    ))}
                  </div>

                  {/* Compact Actions - Only on your turn */}
                  {isMyTurn && actionState === "idle" && (
                    <div className="flex flex-wrap gap-1">
                      {GAME_ACTIONS.map((action) => {
                        const Icon = ACTION_ICONS[action.type];
                        const canAfford = currentPlayer.coins >= action.cost;
                        const mustCoup =
                          currentPlayer.coins >= 10 && action.type !== "coup";

                        return (
                          <Button
                            key={action.type}
                            variant="outline"
                            size="sm"
                            onClick={() => handleActionSelect(action.type)}
                            disabled={!canAfford || mustCoup}
                            className={cn(
                              "h-8 w-8 p-0",
                              "bg-secondary/50 border-border hover:bg-secondary hover:border-accent bg-transparent",
                              "transition-all duration-200",
                              action.role && "border-l-2 border-l-primary",
                              !canAfford && "opacity-50"
                            )}
                            title={`${action.label}${action.cost > 0 ? ` (${action.cost} coins)` : ""}`}
                          >
                            <Icon
                              className="w-4 h-4 text-accent"
                              aria-hidden="true"
                            />
                          </Button>
                        );
                      })}
                    </div>
                  )}

                  {/* Action State Indicators */}
                  {actionState === "selecting-target" && (
                    <motion.div
                      initial={{ opacity: 0, x: -20 }}
                      animate={{ opacity: 1, x: 0 }}
                      className="flex items-center gap-2 bg-primary/20 px-4 py-2 rounded-lg"
                    >
                      <Shuffle className="w-4 h-4 text-primary animate-pulse" />
                      <span className="text-sm font-medium text-primary">
                        Select a target...
                      </span>
                    </motion.div>
                  )}
                </div>

                {/* Must Coup Warning */}
                {isMyTurn && currentPlayer.coins >= 10 && (
                  <motion.p
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    className="text-center text-primary text-xs mt-2 font-medium"
                  >
                    You must perform a Coup!
                  </motion.p>
                )}
              </motion.div>
            )}
          </div>

        </div>
      </main>
    </div>
  );
}
