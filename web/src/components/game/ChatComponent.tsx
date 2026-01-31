import React, { useState, useEffect, useCallback } from 'react';
import { MessageSquare, X, Send } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useGame } from '@/store/gameContext';
import { socket } from '@/network/socket';
import { motion, AnimatePresence } from 'framer-motion';

// Storage key for chat state persistence
const CHAT_STATE_KEY = 'elmakina.chatOpen';

/**
 * Chat toggle button with unread count badge
 * Positioned in bottom right corner
 */
export const ChatToggleButton: React.FC<{
    unreadCount: number;
    isOpen: boolean;
    onClick: () => void;
}> = ({ unreadCount, isOpen, onClick }) => {
    return (
        <motion.button
            initial={{ scale: 0.8, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            onClick={onClick}
            className={cn(
                "fixed bottom-4 right-4 z-[2000]",
                "flex items-center justify-center",
                "w-14 h-14 rounded-full",
                "bg-accent hover:bg-accent/90",
                "text-accent-foreground shadow-lg",
                "border-2 border-accent/20",
                "transition-colors duration-200"
            )}
            aria-label={isOpen ? "Close chat" : "Open chat"}
            aria-expanded={isOpen}
        >
            {isOpen ? (
                <X className="w-6 h-6" />
            ) : (
                <MessageSquare className="w-6 h-6" />
            )}

            {/* Unread count badge */}
            {!isOpen && unreadCount > 0 && (
                <motion.span
                    initial={{ scale: 0 }}
                    animate={{ scale: 1 }}
                    className={cn(
                        "absolute -top-1 -right-1",
                        "flex items-center justify-center",
                        "min-w-[20px] h-5 px-1.5",
                        "rounded-full bg-destructive text-destructive-foreground",
                        "text-xs font-bold shadow-md"
                    )}
                >
                    {unreadCount > 99 ? '99+' : unreadCount}
                </motion.span>
            )}
        </motion.button>
    );
};

/**
 * Glassmorphism chat popover component
 * Opens as overlay from bottom right
 */
export const ChatPopover: React.FC<{
    onClose: () => void;
}> = ({ onClose }) => {
    const { state } = useGame();
    const [message, setMessage] = useState("");
    const [lastReadCount, setLastReadCount] = useState(state.chat.length);
    const scrollRef = React.useRef<HTMLDivElement>(null);

    // Auto-scroll to bottom on new messages
    useEffect(() => {
        if (scrollRef.current) {
            scrollRef.current.scrollIntoView({ behavior: 'smooth' });
        }
    }, [state.chat]);

    // Update last read count when opening
    useEffect(() => {
        setLastReadCount(state.chat.length);
    }, []);

    // Handle keyboard shortcut (Escape to close)
    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                onClose();
            }
        };
        window.addEventListener('keydown', handleKeyDown);
        return () => window.removeEventListener('keydown', handleKeyDown);
    }, [onClose]);

    const handleSendMessage = useCallback((e: React.FormEvent) => {
        e.preventDefault();
        if (!message.trim()) return;

        socket.send("chat_message", { text: message });
        setMessage("");
    }, [message]);

    // Calculate unread messages
    const unreadCount = Math.max(0, state.chat.length - lastReadCount);

    return (
        <motion.div
            initial={{ opacity: 0, y: 20, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 20, scale: 0.95 }}
            transition={{ type: "spring", stiffness: 300, damping: 30 }}
            className={cn(
                "fixed bottom-20 right-4 z-[2000]",
                "w-[min(380px,90vw)] h-[min(500px,70vh)]",
                "flex flex-col rounded-2xl overflow-hidden",
                "bg-card/40 backdrop-blur-xl",
                "border border-white/10 shadow-2xl"
            )}
            role="dialog"
            aria-label="Chat"
            aria-modal="true"
        >
            {/* Header */}
            <div className="flex items-center justify-between px-4 py-3 border-b border-white/10 bg-accent/20">
                <div className="flex items-center gap-2">
                    <MessageSquare className="h-4 w-4 text-accent" />
                    <span className="text-sm font-bold uppercase tracking-wider text-accent">
                        Parlour Chat
                    </span>
                    {unreadCount > 0 && (
                        <span className="px-2 py-0.5 rounded-full bg-destructive/80 text-destructive-foreground text-xs font-semibold">
                            {unreadCount} new
                        </span>
                    )}
                </div>
                <button
                    onClick={onClose}
                    className="p-1.5 rounded-full hover:bg-white/10 transition-colors"
                    aria-label="Close chat"
                >
                    <X className="h-4 w-4 text-muted-foreground" />
                </button>
            </div>

            {/* Messages */}
            <ScrollArea className="flex-1 px-4 py-3">
                <div className="space-y-3">
                    {state.chat.length === 0 ? (
                        <div className="text-center py-8 text-muted-foreground/60">
                            <MessageSquare className="h-8 w-8 mx-auto mb-2 opacity-50" />
                            <p className="text-sm">No messages yet</p>
                            <p className="text-xs mt-1">Start the conversation!</p>
                        </div>
                    ) : (
                        state.chat.map((msg, index) => {
                            const isSelf = msg.senderIndex === state.identity?.playerIndex;
                            const isUnread = index >= lastReadCount;

                            return (
                                <motion.div
                                    key={msg.id}
                                    initial={{ opacity: 0, y: 10 }}
                                    animate={{ opacity: 1, y: 0 }}
                                    className={cn(
                                        "flex flex-col max-w-[85%]",
                                        isSelf ? "ml-auto items-end" : "items-start"
                                    )}
                                >
                                    <span className="text-[10px] font-semibold text-muted-foreground/70 px-1 mb-0.5">
                                        {isSelf ? "You" : msg.senderName}
                                    </span>
                                    <div
                                        className={cn(
                                            "px-3.5 py-2 rounded-2xl text-sm leading-relaxed shadow-sm",
                                            isSelf
                                                ? "bg-accent/80 text-accent-foreground rounded-tr-none"
                                                : "bg-white/10 text-foreground rounded-tl-none",
                                            isUnread && "ring-2 ring-accent/30"
                                        )}
                                    >
                                        {msg.text}
                                    </div>
                                </motion.div>
                            );
                        })
                    )}
                    <div ref={scrollRef} />
                </div>
            </ScrollArea>

            {/* Input */}
            <form
                onSubmit={handleSendMessage}
                className="p-3 border-t border-white/10 bg-accent/10 flex gap-2"
            >
                <Input
                    value={message}
                    onChange={(e) => setMessage(e.target.value)}
                    placeholder="Whisper a secret..."
                    className="h-10 text-sm bg-white/5 border-white/10 focus-visible:ring-accent/30 placeholder:text-muted-foreground/50"
                    aria-label="Message input"
                />
                <Button
                    type="submit"
                    size="icon"
                    className="h-10 w-10 shrink-0 bg-accent hover:bg-accent/90"
                    disabled={!message.trim()}
                    aria-label="Send message"
                >
                    <Send className="h-4 w-4" />
                </Button>
            </form>
        </motion.div>
    );
};

/**
 * Main Chat component that manages toggle state with localStorage persistence
 */
export const ChatComponent: React.FC = () => {
    const { state } = useGame();
    const [isOpen, setIsOpen] = useState(false);
    const [lastReadCount, setLastReadCount] = useState(state.chat.length);

    // Load saved state from localStorage on mount
    useEffect(() => {
        if (typeof window !== 'undefined') {
            const saved = localStorage.getItem(CHAT_STATE_KEY);
            if (saved) {
                setIsOpen(saved === 'true');
            }
        }
    }, []);

    // Save state to localStorage when changed
    useEffect(() => {
        if (typeof window !== 'undefined') {
            localStorage.setItem(CHAT_STATE_KEY, String(isOpen));
        }
    }, [isOpen]);

    // Update last read count when opening
    const handleOpen = useCallback(() => {
        setIsOpen(true);
        setLastReadCount(state.chat.length);
    }, [state.chat.length]);

    const handleClose = useCallback(() => {
        setIsOpen(false);
    }, []);

    const toggleChat = useCallback(() => {
        if (isOpen) {
            handleClose();
        } else {
            handleOpen();
        }
    }, [isOpen, handleOpen, handleClose]);

    // Keyboard shortcut (C) to toggle chat
    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'c' || e.key === 'C') {
                // Don't trigger if typing in an input
                if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
                    return;
                }
                e.preventDefault();
                toggleChat();
            }
        };
        window.addEventListener('keydown', handleKeyDown);
        return () => window.removeEventListener('keydown', handleKeyDown);
    }, [toggleChat]);

    // Calculate unread messages
    const unreadCount = isOpen ? 0 : Math.max(0, state.chat.length - lastReadCount);

    return (
        <>
            <AnimatePresence>
                {isOpen && (
                    <ChatPopover onClose={handleClose} />
                )}
            </AnimatePresence>
            <ChatToggleButton
                unreadCount={unreadCount}
                isOpen={isOpen}
                onClick={toggleChat}
            />
        </>
    );
};
