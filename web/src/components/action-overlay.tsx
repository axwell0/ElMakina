"use client";

import {AnimatePresence, motion} from "framer-motion";
import type {ActionType, Player, Role} from "@/lib/game-types";
import {ACTION_ICONS, GAME_ACTIONS} from "@/lib/game-types";
import {Button} from "@/components/ui/button";
import {Shield, Swords, X} from "lucide-react";
import {ShineEffect} from "@/components/particles";

interface ActionOverlayProps {
  action: ActionType | null;
  actor: Player | null;
  target?: Player | null;
  claimedRole?: Role;
  onChallenge?: () => void;
  onBlock?: () => void;
  onPass?: () => void;
  canBlock?: boolean;
  showResponses?: boolean;
  message?: string;
}

export function ActionOverlay({
  action,
  actor,
  target,
  claimedRole,
  onChallenge,
  onBlock,
  onPass,
  canBlock = false,
  showResponses = false,
  message,
}: ActionOverlayProps) {
  if (!action || !actor) return null;

  const actionData = GAME_ACTIONS.find((a) => a.type === action);
  const Icon = actionData ? ACTION_ICONS[action] : null;

  return (
    <AnimatePresence>
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        className="absolute inset-0 flex items-center justify-center z-50 pointer-events-none"
      >
        {/* Radial energy burst */}
        <motion.div
          className="absolute inset-0 bg-gradient-radial from-accent/20 via-transparent to-transparent"
          initial={{ scale: 0, opacity: 0 }}
          animate={{ scale: 3, opacity: [0, 0.5, 0] }}
          transition={{ duration: 0.8, ease: "easeOut" }}
        />

        <div className="relative pointer-events-auto">
          {/* Main announcement card - Hearthstone style entrance */}
            <motion.div
            className="relative w-[min(92vw,420px)] max-w-md min-w-0 bg-gradient-to-b from-card via-card/95 to-card/90 backdrop-blur-md border-4 border-accent rounded-2xl shadow-2xl p-4 sm:p-6 md:p-8 overflow-hidden"
            initial={{ 
              scale: 0.3, 
              rotateX: 90, 
              y: -100, 
              opacity: 0,
              filter: "brightness(3) blur(10px)"
            }}
            animate={{ 
              scale: 1, 
              rotateX: 0, 
              y: 0, 
              opacity: 1,
              filter: "brightness(1) blur(0px)"
            }}
            exit={{ 
              scale: 0.3, 
              rotateY: 90, 
              opacity: 0,
              filter: "brightness(0) blur(10px)"
            }}
            transition={{ 
              type: "spring", 
              stiffness: 300, 
              damping: 25,
              mass: 0.8 
            }}
            style={{ 
              transformStyle: "preserve-3d",
              perspective: 1000
            }}
          >
            {/* Animated golden border glow - Hearthstone style */}
            <motion.div
              className="absolute -inset-[2px] bg-gradient-to-r from-accent via-primary to-accent rounded-2xl opacity-75 blur-md -z-10"
              animate={{
                opacity: [0.5, 1, 0.5],
                scale: [1, 1.05, 1],
              }}
              transition={{
                duration: 2,
                repeat: Number.POSITIVE_INFINITY,
                ease: "easeInOut",
              }}
            />
            
            {/* Light rays effect */}
            <motion.div
              className="absolute inset-0 overflow-hidden rounded-2xl pointer-events-none"
              initial={{ opacity: 0 }}
              animate={{ opacity: [0, 0.4, 0] }}
              transition={{ duration: 1.5, delay: 0.2 }}
            >
              {[...Array(8)].map((_, i) => (
                <motion.div
                  key={i}
                  className="absolute top-1/2 left-1/2 w-1 h-full bg-gradient-to-t from-transparent via-accent/50 to-transparent origin-bottom"
                  style={{ 
                    transform: `rotate(${i * 45}deg)`,
                  }}
                  initial={{ scaleY: 0, opacity: 0 }}
                  animate={{ scaleY: 2, opacity: [0, 1, 0] }}
                  transition={{ 
                    duration: 0.8, 
                    delay: 0.1 + i * 0.05,
                    ease: "easeOut" 
                  }}
                />
              ))}
            </motion.div>

            {/* Shine effect */}
            <ShineEffect />

            {/* Header with dramatic icon */}
              <div className="flex items-center gap-4 mb-6">
              {Icon && (
                <motion.div
                  className="relative w-12 h-12 sm:w-14 sm:h-14 rounded-full bg-gradient-to-br from-accent/30 to-primary/30 flex items-center justify-center"
                  initial={{ scale: 0, rotate: -180 }}
                  animate={{ 
                    scale: 1, 
                    rotate: 0,
                  }}
                  transition={{ 
                    type: "spring",
                    stiffness: 200,
                    damping: 15,
                    delay: 0.2
                  }}
                >
                  {/* Pulsing glow behind icon */}
                  <motion.div
                    className="absolute inset-0 rounded-full bg-accent/40 blur-lg"
                    animate={{ 
                      scale: [1, 1.3, 1],
                      opacity: [0.5, 0.8, 0.5]
                    }}
                    transition={{ 
                      duration: 1.5, 
                      repeat: Number.POSITIVE_INFINITY 
                    }}
                  />
                  
                  {/* Spinning ring */}
                  <motion.div
                    className="absolute inset-0 border-2 border-accent/60 rounded-full"
                    animate={{ rotate: 360 }}
                    transition={{ 
                      duration: 3, 
                      repeat: Number.POSITIVE_INFINITY,
                      ease: "linear" 
                    }}
                  />
                  
                  <motion.div
                    animate={{ 
                      scale: [1, 1.1, 1],
                      rotate: [0, 5, -5, 0]
                    }}
                    transition={{ 
                      duration: 2, 
                      repeat: Number.POSITIVE_INFINITY 
                    }}
                  >
                    <Icon className="w-6 h-6 sm:w-7 sm:h-7 text-accent drop-shadow-[0_0_8px_rgba(0,0,0,0.25)]" />
                  </motion.div>
                </motion.div>
              )}
              <div className="flex-1">
                <motion.h3 
                  className="font-serif text-xl sm:text-2xl text-foreground drop-shadow-md"
                  initial={{ x: -20, opacity: 0 }}
                  animate={{ x: 0, opacity: 1 }}
                  transition={{ delay: 0.3 }}
                >
                  {actionData?.label}
                </motion.h3>
                <motion.p 
                  className="text-sm text-muted-foreground"
                  initial={{ x: -20, opacity: 0 }}
                  animate={{ x: 0, opacity: 1 }}
                  transition={{ delay: 0.4 }}
                >
                  by <span className="text-accent font-medium">{actor.name}</span>
                </motion.p>
              </div>
            </div>

            {/* Action details */}
            <div className="space-y-2 mb-4">
              {claimedRole && (
                <motion.div
                  initial={{ x: -20, opacity: 0 }}
                  animate={{ x: 0, opacity: 1 }}
                  transition={{ delay: 0.2 }}
                  className="flex items-center gap-2 p-2 bg-primary/10 rounded-lg"
                >
                  <Shield className="w-4 h-4 text-primary" />
                  <span className="text-sm text-foreground">
                    Claiming <span className="font-medium">{claimedRole}</span>
                  </span>
                </motion.div>
              )}

              {target && (
                <motion.div
                  initial={{ x: -20, opacity: 0 }}
                  animate={{ x: 0, opacity: 1 }}
                  transition={{ delay: 0.3 }}
                  className="flex items-center gap-2 p-2 bg-secondary/50 rounded-lg"
                >
                  <Swords className="w-4 h-4 text-accent" />
                  <span className="text-sm text-foreground">
                    Targeting <span className="font-medium">{target.name}</span>
                  </span>
                </motion.div>
              )}

              {message && (
                <motion.p
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: 0.4 }}
                  className="text-sm text-muted-foreground text-center italic"
                >
                  {message}
                </motion.p>
              )}
            </div>

            {/* Response buttons */}
            {showResponses && (
              <motion.div
                initial={{ y: 20, opacity: 0 }}
                animate={{ y: 0, opacity: 1 }}
                transition={{ delay: 0.5 }}
                className="flex flex-col gap-2"
              >
                <div className="flex gap-2">
                  {onChallenge && claimedRole && (
                    <Button
                      onClick={onChallenge}
                      variant="outline"
                      className="flex-1 bg-primary/20 border-primary text-primary hover:bg-primary hover:text-primary-foreground bg-transparent"
                    >
                      <Swords className="w-4 h-4 mr-2" />
                      Challenge
                    </Button>
                  )}

                  {canBlock && onBlock && (
                    <Button
                      onClick={onBlock}
                      variant="outline"
                      className="flex-1 bg-accent/20 border-accent text-accent hover:bg-accent hover:text-accent-foreground bg-transparent"
                    >
                      <Shield className="w-4 h-4 mr-2" />
                      Block
                    </Button>
                  )}
                </div>

                {onPass && (
                  <Button
                    onClick={onPass}
                    variant="ghost"
                    className="w-full text-muted-foreground hover:text-foreground"
                  >
                    <X className="w-4 h-4 mr-2" />
                    Pass
                  </Button>
                )}
              </motion.div>
            )}
          </motion.div>
        </div>
      </motion.div>
    </AnimatePresence>
  );
}
