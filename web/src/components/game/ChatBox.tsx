import React, {useEffect, useRef, useState} from 'react';
import {useGame} from '../../store/gameContext';
import {socket} from '../../network/socket';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {MessageSquare, Send} from 'lucide-react';
import {cn} from '@/lib/utils';
import {ScrollArea} from '@/components/ui/scroll-area';

export const ChatBox: React.FC = () => {
    const { state } = useGame();
    const [message, setMessage] = useState("");
    const scrollRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (scrollRef.current) {
            scrollRef.current.scrollIntoView({ behavior: 'smooth' });
        }
    }, [state.chat]);

    const handleSendMessage = (e: React.FormEvent) => {
        e.preventDefault();
        if (!message.trim()) return;

        socket.send("chat_message", { text: message });
        setMessage("");
    };

    return (
        <div className="flex flex-col h-[min(280px,30vh)] w-full bg-card/80 rounded-2xl border-2 border-accent/20 shadow-2xl overflow-hidden backdrop-blur-md transition-all duration-300 hover:border-accent/40">
            <div className="flex items-center justify-between px-4 py-2.5 border-b border-border/10 bg-accent/10">
                <div className="flex items-center gap-2">
                    <MessageSquare className="h-3.5 w-3.5 text-accent" />
                    <span className="text-[10px] font-black uppercase tracking-[0.25em] text-accent/80">Parlour Chat</span>
                </div>
                <div className="flex gap-1">
                    <div className="w-1 h-1 rounded-full bg-accent/40" />
                    <div className="w-1 h-1 rounded-full bg-accent/20" />
                </div>
            </div>

            <ScrollArea className="flex-1 px-3 py-2">
                <div className="space-y-3">
                    {(state.chat || []).map((msg) => {
                        const isSelf = msg.senderIndex === state.identity?.playerIndex;
                        return (
                            <div key={msg.id} className={cn(
                                "flex flex-col max-w-[90%] animate-in fade-in slide-in-from-bottom-1 duration-300",
                                isSelf ? "ml-auto items-end" : "items-start"
                            )}>
                                <span className="text-[9px] font-bold text-muted-foreground/70 px-1 mb-0.5">
                                    {isSelf ? "You" : msg.senderName}
                                </span>
                                <div className={cn(
                                    "px-3.5 py-2 rounded-2xl text-xs leading-relaxed shadow-sm",
                                    isSelf
                                        ? "bg-accent text-accent-foreground rounded-tr-none border-b border-l border-black/5"
                                        : "bg-secondary/60 text-secondary-foreground rounded-tl-none border-b border-r border-black/5"
                                )}>
                                    {msg.text}
                                </div>
                            </div>
                        );
                    })}
                    <div ref={scrollRef} />
                </div>
            </ScrollArea>

            <form onSubmit={handleSendMessage} className="p-3 border-t border-border/10 bg-accent/5 flex gap-2">
                <Input
                    value={message}
                    onChange={(e) => setMessage(e.target.value)}
                    placeholder="Whisper a secret..."
                    className="h-9 text-xs bg-background/50 border-accent/10 focus-visible:ring-accent/30 placeholder:text-muted-foreground/50"
                />
                <Button type="submit" size="icon" className="h-9 w-9 shrink-0 bg-accent hover:bg-black hover:text-accent transition-all">
                    <Send className="h-3.5 w-3.5" />
                </Button>
            </form>
        </div>
    );
};
