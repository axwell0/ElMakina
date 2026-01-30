"use client";

import type {Player} from "@/lib/game-types";
import {cn} from "@/lib/utils";
import {Coins, Shield, Target} from "lucide-react";
import {motion} from "framer-motion";

interface PlayerCardProps {
  player: Player;
  isActive: boolean;
  isCurrentPlayer: boolean;
  position: number;
  totalPlayers: number;
  onSelect?: () => void;
  selectable?: boolean;
}

export function PlayerCard({
  player,
  isActive,
  isCurrentPlayer,
  position,
  totalPlayers,
  onSelect,
  selectable = false,
}: PlayerCardProps) {
  // Calculate position around the circular table
  const angle = (position / totalPlayers) * 2 * Math.PI - Math.PI / 2;
  const radius = 38; // percentage from center

  const style = {
    left: `${50 + radius * Math.cos(angle)}%`,
    top: `${50 + radius * Math.sin(angle)}%`,
    transform: "translate(-50%, -50%)",
  };

  return (
    <motion.button
      type="button"
      onClick={selectable ? onSelect : undefined}
      disabled={!selectable}
      style={style}
      className={cn(
        "absolute w-28 md:w-32 p-3 rounded-lg transition-all duration-300",
        "bg-card border-2 shadow-lg",
        isActive && "ring-2 ring-accent ring-offset-2 ring-offset-background",
        isCurrentPlayer && "border-primary",
        !isActive && !isCurrentPlayer && "border-border",
        player.isEliminated && "opacity-50 grayscale",
        selectable && !player.isEliminated && "hover:scale-110 cursor-pointer hover:border-accent hover:shadow-2xl hover:shadow-accent/50",
        !selectable && "cursor-default"
      )}
      aria-label={`${player.name}${isActive ? ", active player" : ""}${player.isEliminated ? ", eliminated" : ""}`}
      whileHover={selectable && !player.isEliminated ? { scale: 1.1, y: -5 } : {}}
      whileTap={selectable && !player.isEliminated ? { scale: 0.95 } : {}}
      animate={selectable && !player.isEliminated ? { y: [0, -4, 0] } : {}}
      transition={{ duration: 0.6, repeat: Number.POSITIVE_INFINITY, repeatType: "reverse" }}
    >
      {/* Selectable Target Indicator - Hearthstone style */}
      {selectable && !player.isEliminated && (
        <>
          {/* Outer pulsing glow */}
          <motion.div 
            className="absolute -inset-3 bg-gradient-to-r from-primary via-accent to-primary rounded-lg blur-xl -z-20"
            animate={{ 
              opacity: [0.4, 0.8, 0.4],
              scale: [1, 1.1, 1]
            }}
            transition={{ 
              duration: 1.2, 
              repeat: Number.POSITIVE_INFINITY,
              ease: "easeInOut"
            }}
          />
          
          {/* Inner sharp glow */}
          <motion.div 
            className="absolute -inset-1 bg-primary/50 rounded-lg blur-md -z-10"
            animate={{ 
              opacity: [0.5, 1, 0.5],
            }}
            transition={{ 
              duration: 0.8, 
              repeat: Number.POSITIVE_INFINITY 
            }}
          />
          
          {/* Rotating border ring */}
          <motion.div
            className="absolute -inset-2 border-2 border-accent/60 rounded-lg -z-10"
            animate={{ rotate: 360 }}
            transition={{ 
              duration: 4, 
              repeat: Number.POSITIVE_INFINITY,
              ease: "linear"
            }}
            style={{
              background: "linear-gradient(45deg, transparent 30%, rgba(0, 0, 0, 0.15) 50%, transparent 70%)"
            }}
          />
        </>
      )}

      {/* Active glow effect - Enhanced */}
      {isActive && !selectable && (
        <>
          <motion.div 
            className="absolute -inset-2 bg-accent/40 rounded-lg blur-lg -z-10"
            animate={{
              opacity: [0.3, 0.6, 0.3],
              scale: [1, 1.05, 1]
            }}
            transition={{
              duration: 2,
              repeat: Number.POSITIVE_INFINITY,
              ease: "easeInOut"
            }}
          />
          <div className="absolute -inset-[1px] bg-gradient-to-r from-accent via-primary to-accent rounded-lg opacity-50 -z-10" />
        </>
      )}

      {/* Target Icon for selectable players - Enhanced */}
      {selectable && !player.isEliminated && (
        <>
          {/* Pulsing glow behind icon */}
          <motion.div 
            className="absolute -top-3 -right-3 w-10 h-10 bg-primary/60 rounded-full blur-lg"
            animate={{ 
              scale: [1, 1.4, 1],
              opacity: [0.6, 0.9, 0.6]
            }}
            transition={{ 
              duration: 1, 
              repeat: Number.POSITIVE_INFINITY 
            }}
          />
          
          <motion.div 
            className="absolute -top-2 -right-2 w-8 h-8 bg-gradient-to-br from-primary to-primary/80 rounded-full flex items-center justify-center shadow-xl border-2 border-accent/50"
            animate={{ 
              scale: [1, 1.15, 1],
              rotate: [0, 360]
            }}
            transition={{ 
              scale: { duration: 0.8, repeat: Number.POSITIVE_INFINITY },
              rotate: { duration: 3, repeat: Number.POSITIVE_INFINITY, ease: "linear" }
            }}
          >
            <Target className="w-5 h-5 text-primary-foreground drop-shadow-lg" />
          </motion.div>
        </>
      )}

      {/* Avatar */}
      <div className="flex flex-col items-center gap-2">
        <div
          className={cn(
            "w-12 h-12 rounded-full bg-secondary flex items-center justify-center text-lg font-serif",
            isActive && "ring-2 ring-accent"
          )}
        >
          {player.name.charAt(0)}
        </div>

        {/* Name */}
        <span className="text-sm font-medium text-foreground truncate w-full text-center">
          {player.name}
        </span>

        {/* Coins */}
        <div className="flex items-center gap-1">
          {Array.from({ length: Math.min(player.coins, 10) }).map((_, i) => (
            <Coins
              key={i}
              className="w-3 h-3 text-accent"
              aria-hidden="true"
            />
          ))}
          {player.coins > 10 && (
            <span className="text-xs text-accent">+{player.coins - 10}</span>
          )}
        </div>
        <span className="text-xs text-muted-foreground">
          {player.coins} coins
        </span>

        {/* Cards */}
        <div className="flex gap-1 mt-1">
          {player.cards.map((_, index) => (
            <div
              key={index}
              className="w-6 h-8 rounded bg-primary/80 border border-accent/30 shadow-inner"
              aria-label="Hidden card"
            />
          ))}
          {player.revealedCards.map((role, index) => (
            <div
              key={`revealed-${index}`}
              className="w-6 h-8 rounded bg-muted border border-border flex items-center justify-center"
              title={role}
            >
              <Shield className="w-3 h-3 text-muted-foreground" />
            </div>
          ))}
        </div>
      </div>
    </motion.button>
  );
}
