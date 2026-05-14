import { motion } from "framer-motion";
import type { ComponentType } from "react";
import {
  BadgeAlert,
  Banknote,
  CircleDollarSign,
  Eye,
  Hand,
  Landmark,
  RadioTower,
  Shield,
  Swords,
  UserRoundSearch
} from "lucide-react";

type Player = {
  name: string;
  coins: number;
  cards: number;
  color: "green" | "purple" | "orange" | "red";
  current?: boolean;
  self?: boolean;
};

type Action = {
  label: string;
  icon: ComponentType<{ size?: number }>;
  tone: "paper" | "orange" | "red" | "green";
};

const players: Player[] = [
  { name: "You", coins: 2, cards: 2, color: "green", self: true },
  { name: "Amine", coins: 5, cards: 2, color: "purple", current: true },
  { name: "Sarah", coins: 1, cards: 1, color: "orange" },
  { name: "Karim", coins: 7, cards: 2, color: "red" }
];

const actions: Action[] = [
  { label: "Income", icon: CircleDollarSign, tone: "paper" },
  { label: "Tax", icon: Landmark, tone: "green" },
  { label: "Business", icon: Banknote, tone: "orange" },
  { label: "Steal", icon: Hand, tone: "paper" },
  { label: "Investigate", icon: Eye, tone: "paper" },
  { label: "Coup", icon: Swords, tone: "red" }
];

const logItems = [
  "Room SAIL-8645 synchronized",
  "Amine declared Business",
  "Challenge window opened",
  "Machine awaiting response"
];

export function App() {
  return (
    <main className="game-shell">
      <TopBar />
      <section className="tabletop" aria-label="El Makina game table">
        <aside className="player-column player-column-left">
          {players.slice(0, 2).map((player) => (
            <PlayerMonitor key={player.name} player={player} />
          ))}
        </aside>

        <MachineHub />

        <aside className="player-column player-column-right">
          {players.slice(2).map((player) => (
            <PlayerMonitor key={player.name} player={player} />
          ))}
        </aside>
      </section>

      <section className="lower-dock">
        <HandDock />
        <ActionConsole />
        <EventLog />
      </section>
    </main>
  );
}

function TopBar() {
  return (
    <header className="top-bar">
      <div className="brand-lockup">
        <span className="gear-mark" aria-hidden="true">
          ⚙
        </span>
        <div>
          <div className="brand-name">ELMAKINA</div>
          <div className="brand-subtitle">Identité secrète • Déception • Déduction</div>
        </div>
      </div>
      <div className="room-chip">
        <RadioTower size={18} />
        <span>SAIL-8645</span>
      </div>
    </header>
  );
}

function MachineHub() {
  return (
    <motion.section
      className="machine-hub"
      initial={{ opacity: 0, y: 18 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.45, ease: "easeOut" }}
    >
      <div className="machine-antennas" aria-hidden="true">
        <span />
        <span />
        <span />
      </div>
      <div className="machine-body">
        <div className="machine-lamps" aria-hidden="true">
          <span />
          <span />
          <span />
        </div>
        <div className="machine-screen">
          <div className="scanlines" />
          <strong>DEMOCRACY</strong>
          <span>LOADING</span>
        </div>
        <div className="machine-meter">
          <span />
          <span />
          <span />
          <span />
        </div>
        <div className="machine-ticket">بطاقة سرية</div>
        <div className="machine-dial" />
      </div>
      <div className="phase-banner">
        <BadgeAlert size={18} />
        <span>Challenge window</span>
      </div>
    </motion.section>
  );
}

function PlayerMonitor({ player }: { player: Player }) {
  return (
    <motion.article
      className={`player-monitor monitor-${player.color} ${player.current ? "is-current" : ""}`}
      layout
      whileHover={{ y: -2 }}
    >
      <div className="monitor-screen">
        <div className="portrait-silhouette" />
      </div>
      <div className="player-meta">
        <strong>{player.name}</strong>
        <span>{player.self ? "Your signal" : player.current ? "Acting" : "Standing by"}</span>
      </div>
      <div className="player-stats">
        <span>{player.coins} coins</span>
        <span>{"?".repeat(player.cards)}</span>
      </div>
    </motion.article>
  );
}

function HandDock() {
  return (
    <section className="hand-dock" aria-label="Your influence cards">
      <div className="mini-card card-green">
        <span>Businesswoman</span>
      </div>
      <div className="mini-card card-purple">
        <span>Thief</span>
      </div>
    </section>
  );
}

function ActionConsole() {
  return (
    <nav className="action-console" aria-label="Turn actions">
      {actions.map((action) => {
        const Icon = action.icon;
        return (
          <button className={`action-button action-${action.tone}`} key={action.label} type="button">
            <Icon size={18} />
            <span>{action.label}</span>
          </button>
        );
      })}
    </nav>
  );
}

function EventLog() {
  return (
    <aside className="event-log" aria-label="Game log">
      {logItems.map((item) => (
        <div className="event-row" key={item}>
          <Shield size={14} />
          <span>{item}</span>
        </div>
      ))}
    </aside>
  );
}
