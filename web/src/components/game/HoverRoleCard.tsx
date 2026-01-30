import React from 'react';
import {cn} from '@/lib/utils';
import {AnimatePresence, motion} from 'framer-motion';

export type RoleDetailsMap = Record<string, { main: string; counter: string }>;

type HoverRoleCardProps = {
    role: string | null;
    image: string | null;
    details: RoleDetailsMap;
    className?: string;
};

const HoverRoleCardBase: React.FC<HoverRoleCardProps> = ({ role, image, details, className }) => {
    return (
        <AnimatePresence>
            {role && (
                <motion.div
                    initial={{ opacity: 0, x: -30, scale: 0.9 }}
                    animate={{ opacity: 1, x: 0, scale: 1 }}
                    exit={{ opacity: 0, x: -20, scale: 0.95 }}
                    transition={{ duration: 0.15, ease: [0.23, 1, 0.32, 1] }} // Snappier
                    className={cn(
                        "pointer-events-none z-[40] hidden lg:block", // Extremely high z-index
                        className
                    )}
                >
                    <div className="w-[min(90vw,320px)] overflow-hidden rounded-2xl border-2 border-accent/50 bg-card/95 shadow-[0_20px_50px_rgba(0,0,0,0.5)] backdrop-blur-xl">
                        {/* Header Area */}
                        <div className="bg-accent/15 px-5 py-2.5 border-b border-accent/20">
                            <div className="text-[10px] font-bold uppercase tracking-[0.4em] text-accent/90">Intelligence File</div>
                        </div>

                        <div className="p-0">
                            {/* Full width card image */}
                            {image && (
                                <div className="relative w-full aspect-[2/2.8] overflow-hidden border-b border-accent/20">
                                    <img src={image} alt={role} className="h-full w-full object-cover" />
                                    <div className="absolute inset-0 bg-gradient-to-t from-card via-transparent to-transparent opacity-40" />
                                </div>
                            )}

                            <div className="p-6">
                                {/* Name Underneath */}
                                <div className="mb-6 text-center">
                                    <h4 className="font-serif text-3xl text-foreground tracking-tight">{role}</h4>
                                    <div className="mx-auto mt-2 h-0.5 w-12 bg-accent/60" />
                                </div>

                                <div className="space-y-6">
                                    <div className="relative pl-4 border-l-2 border-primary/30">
                                        <div className="mb-1.5 text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground">Primary Directive</div>
                                        <p className="text-[15px] text-foreground/90 italic leading-relaxed font-sans">
                                            {details[role]?.main ?? 'Details classified.'}
                                        </p>
                                    </div>

                                    <div className="relative pl-4 border-l-2 border-destructive/30">
                                        <div className="mb-1.5 text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground">Counter Intelligence</div>
                                        <p className="text-[15px] text-foreground/90 italic leading-relaxed font-sans">
                                            {details[role]?.counter ?? 'No defensive measures.'}
                                        </p>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div className="bg-accent/5 px-5 py-3 border-t border-accent/10">
                            <div className="text-[10px] text-center font-medium tracking-widest text-muted-foreground/60">SECTOR-7 PARLOUR ARCHIVE</div>
                        </div>
                    </div>
                </motion.div>
            )}
        </AnimatePresence>
    );
};

export const HoverRoleCard = React.memo(HoverRoleCardBase, (prev, next) => (
    prev.role === next.role && prev.image === next.image && prev.details === next.details
));
