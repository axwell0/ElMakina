"use client";

import {useState} from "react";
import {useGame} from "@/lib/game-context";
import type {GameRoom, Player} from "@/lib/game-types";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {Label} from "@/components/ui/label";
import {Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger,} from "@/components/ui/dialog";
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue,} from "@/components/ui/select";
import {Crown, Flame, Play, Plus, RefreshCw, Settings2, Users,} from "lucide-react";

export function LobbyScreen() {
  const { theme, setTheme, rooms, setScreen, setCurrentPlayer, setPlayers, setActivePlayerId } = useGame();
  const [nickname, setNickname] = useState("");
  const [isCreateRoomOpen, setIsCreateRoomOpen] = useState(false);
  const [newRoomName, setNewRoomName] = useState("");
  const [maxPlayers, setMaxPlayers] = useState("6");

  const handleJoinRoom = (room: GameRoom) => {
    const newPlayer: Player = {
      id: `player-${Date.now()}`,
      name: nickname || "Anonymous",
      avatar: "/placeholder.svg?height=40&width=40",
      coins: 2,
      cards: ["Duke", "Assassin"],
      revealedCards: [],
      isActive: false,
      isEliminated: false,
    };
    setCurrentPlayer(newPlayer);
    setPlayers([...room.players, newPlayer]);
    setActivePlayerId(room.players[0]?.id || newPlayer.id);
    setScreen("game");
  };

  const handleCreateRoom = () => {
    setIsCreateRoomOpen(false);
    setNewRoomName("");
  };

  const handleRefresh = () => {
    // Simulate refresh animation
  };

  return (
    <div className="min-h-screen bg-background p-3 sm:p-4 md:p-6">
      {/* Decorative candle glow effect */}
      <div className="fixed inset-0 pointer-events-none overflow-hidden">
        <div className="absolute top-20 left-10 w-64 h-64 bg-accent/10 rounded-full blur-3xl" />
        <div className="absolute bottom-20 right-10 w-48 h-48 bg-primary/10 rounded-full blur-3xl" />
      </div>

      <div className="relative max-w-6xl mx-auto">
        {/* Header */}
        <header className="flex flex-col md:flex-row items-center justify-between gap-6 mb-8">
          <div className="text-center md:text-left">
            <div className="flex items-center gap-3 justify-center md:justify-start">
              <Flame className="w-8 h-8 text-accent" aria-hidden="true" />
              <h1 className="text-3xl sm:text-4xl md:text-5xl font-serif tracking-wider text-foreground">
                ElMakina
              </h1>
            </div>
            <p className="text-muted-foreground mt-2 text-base sm:text-lg">
              A game of bluff, strategy & intrigue
            </p>
          </div>

          {/* Theme Toggle */}
          <div className="flex items-center gap-4">
            <Label htmlFor="theme" className="text-muted-foreground">
              Theme:
            </Label>
            <Select value={theme} onValueChange={(v) => setTheme(v as 'tabletop' | 'modern')}>
              <SelectTrigger id="theme" className="w-36 bg-card border-border">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="tabletop">Tabletop</SelectItem>
                <SelectItem value="modern">Modern</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </header>

        {/* Player Setup Section */}
        <section
          className="bg-card rounded-lg border border-border p-4 sm:p-5 md:p-6 mb-8 shadow-lg"
          aria-labelledby="player-setup-heading"
        >
          <h2
            id="player-setup-heading"
            className="text-lg sm:text-xl font-serif text-foreground mb-4 flex items-center gap-2"
          >
            <Settings2 className="w-5 h-5 text-accent" aria-hidden="true" />
            Player Setup
          </h2>
          <div className="flex flex-col md:flex-row gap-4 items-end">
            <div className="flex-1 space-y-2">
              <Label htmlFor="nickname" className="text-muted-foreground">
                Your Name
              </Label>
              <Input
                id="nickname"
                placeholder="Enter your name..."
                value={nickname}
                onChange={(e) => setNickname(e.target.value)}
                className="border-border text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div className="flex-shrink-0">
              <Button
                variant="outline"
                className="border-border text-foreground hover:bg-muted bg-transparent"
              >
                Upload Avatar
              </Button>
            </div>
          </div>
        </section>

        {/* Game Rooms Section */}
        <section aria-labelledby="rooms-heading">
          <div className="flex items-center justify-between mb-4">
            <h2
              id="rooms-heading"
              className="text-xl sm:text-2xl font-serif text-foreground flex items-center gap-2"
            >
              <Crown className="w-6 h-6 text-accent" aria-hidden="true" />
              Available Rooms
            </h2>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="icon"
                onClick={handleRefresh}
                className="h-9 w-9 sm:h-10 sm:w-10 border-border text-foreground hover:bg-muted bg-transparent"
                aria-label="Refresh rooms"
              >
                <RefreshCw className="w-4 h-4" />
              </Button>
              <Dialog open={isCreateRoomOpen} onOpenChange={setIsCreateRoomOpen}>
                <DialogTrigger asChild>
                  <Button className="h-9 sm:h-10 text-sm sm:text-base bg-primary text-primary-foreground hover:bg-primary/90 dark:text-black">
                    <Plus className="w-4 h-4 mr-2" aria-hidden="true" />
                    Create Room
                  </Button>
                </DialogTrigger>
                <DialogContent className="bg-card border-border">
                  <DialogHeader>
                    <DialogTitle className="font-serif text-foreground">
                      Create New Room
                    </DialogTitle>
                  </DialogHeader>
                  <div className="space-y-4 pt-4">
                    <div className="space-y-2">
                      <Label htmlFor="room-name" className="text-muted-foreground">
                        Room Name
                      </Label>
                      <Input
                        id="room-name"
                        placeholder="Enter room name..."
                        value={newRoomName}
                        onChange={(e) => setNewRoomName(e.target.value)}
                        className="border-border text-foreground"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="max-players" className="text-muted-foreground">
                        Max Players
                      </Label>
                      <Select value={maxPlayers} onValueChange={setMaxPlayers}>
                        <SelectTrigger
                          id="max-players"
                          className="border-border text-foreground"
                        >
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="2">2 Players</SelectItem>
                          <SelectItem value="3">3 Players</SelectItem>
                          <SelectItem value="4">4 Players</SelectItem>
                          <SelectItem value="5">5 Players</SelectItem>
                          <SelectItem value="6">6 Players</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <Button
                      onClick={handleCreateRoom}
                      className="w-full h-9 sm:h-10 text-sm sm:text-base bg-primary text-primary-foreground hover:bg-primary/90"
                    >
                      Create Room
                    </Button>
                  </div>
                </DialogContent>
              </Dialog>
            </div>
          </div>

          {/* Room Cards Grid */}
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {rooms.map((room) => (
              <article
                key={room.id}
                className="group bg-card rounded-lg border border-border p-4 shadow-lg hover:border-accent/50 transition-all duration-300 hover:shadow-accent/10"
              >
                {/* Card ornate border effect */}
                <div className="relative">
                  <div className="absolute -inset-px bg-gradient-to-br from-accent/20 via-transparent to-primary/20 rounded-lg opacity-0 group-hover:opacity-100 transition-opacity" />
                  <div className="relative">
                    <div className="flex items-center justify-between mb-3">
                      <h3 className="font-serif text-lg text-foreground">
                        {room.name}
                      </h3>
                      <span
                        className={`text-xs px-2 py-1 rounded-full ${room.status === "waiting"
                          ? "bg-accent/20 text-accent"
                          : room.status === "playing"
                            ? "bg-primary/20 text-primary"
                            : "bg-muted text-muted-foreground"
                          }`}
                      >
                        {room.status === "waiting"
                          ? "Waiting"
                          : room.status === "playing"
                            ? "In Progress"
                            : "Finished"}
                      </span>
                    </div>

                    {/* Player Avatars */}
                    <div className="flex items-center gap-2 mb-4">
                      <Users
                        className="w-4 h-4 text-muted-foreground"
                        aria-hidden="true"
                      />
                      <div className="flex -space-x-2">
                        {room.players.map((player) => (
                          <div
                            key={player.id}
                            className="w-8 h-8 rounded-full bg-secondary border-2 border-card flex items-center justify-center text-xs font-medium text-secondary-foreground"
                            title={player.name}
                          >
                            {player.name.charAt(0)}
                          </div>
                        ))}
                      </div>
                      <span className="text-sm text-muted-foreground ml-auto">
                        {room.players.length}/{room.maxPlayers}
                      </span>
                    </div>

                    {/* Host Info */}
                    <p className="text-sm text-muted-foreground mb-4">
                      <span className="text-accent">Host:</span> {room.host}
                    </p>

                    {/* Join Button */}
                    <Button
                      onClick={() => handleJoinRoom(room)}
                      disabled={room.status !== "waiting"}
                      className="w-full bg-secondary text-secondary-foreground hover:bg-secondary/80 disabled:opacity-50"
                    >
                      <Play className="w-4 h-4 mr-2" aria-hidden="true" />
                      {room.status === "waiting" ? "Join Game" : "Spectate"}
                    </Button>
                  </div>
                </div>
              </article>
            ))}
          </div>
        </section>
      </div>
    </div>
  );
}
