"use client";

import {ScrollArea} from "@/components/ui/scroll-area";
import {ScrollText} from "lucide-react";

interface GameLogProps {
  entries: string[];
}

export function GameLog({ entries }: GameLogProps) {
  return (
    <div className="bg-card/90 backdrop-blur-sm rounded-lg border border-border p-3 shadow-xl h-48 md:h-64">
      <h3 className="font-serif text-sm text-foreground mb-2 flex items-center gap-2">
        <ScrollText className="w-4 h-4 text-accent" aria-hidden="true" />
        Turn History
      </h3>
      <ScrollArea className="h-[calc(100%-2rem)]">
        <ul className="space-y-1 pr-3" role="log" aria-live="polite">
          {entries.map((entry, index) => (
            <li
              key={index}
              className="text-xs text-muted-foreground py-1 border-b border-border/50 last:border-0"
            >
              <span className="text-accent/60 mr-2">{index + 1}.</span>
              {entry}
            </li>
          ))}
        </ul>
      </ScrollArea>
    </div>
  );
}
