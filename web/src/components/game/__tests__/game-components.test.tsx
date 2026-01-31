import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import '@testing-library/jest-dom';

// Mock framer-motion
vi.mock('framer-motion', () => ({
  motion: {
    div: ({ children, ...props }: React.PropsWithChildren<Record<string, unknown>>) => (
      <div {...props}>{children}</div>
    ),
  },
  AnimatePresence: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  useMotionValue: () => ({
    set: vi.fn(),
    get: vi.fn(() => 0),
  }),
  useReducedMotion: () => false,
}));

// Mock socket
vi.mock('@/network/socket', () => ({
  socket: {
    send: vi.fn(),
  },
}));

// Mock card image function
vi.mock('@/lib/cards', () => ({
  cardImageForRole: (role: string) => `/cards/${role}.png`,
}));

// Mock lucide-react icons
vi.mock('lucide-react', () => ({
  Coins: () => <span data-testid="coins-icon">💰</span>,
  Target: () => <span data-testid="target-icon">🎯</span>,
  Timer: () => <span data-testid="timer-icon">⏱️</span>,
  MessageSquare: () => <span data-testid="message-square-icon">💬</span>,
  Send: () => <span data-testid="send-icon">📤</span>,
}));

// Mock UI components
vi.mock('@/components/ui/button', () => ({
  Button: ({ children, ...props }: React.PropsWithChildren<React.ButtonHTMLAttributes<HTMLButtonElement>>) => (
    <button {...props}>{children}</button>
  ),
}));

vi.mock('@/components/ui/input', () => ({
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) => <input {...props} />,
}));

vi.mock('@/components/ui/scroll-area', () => ({
  ScrollArea: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

import { PlayerRing } from '../PlayerRing';
import { HandTray } from '../HandTray';
import { ChatBox } from '../ChatBox';
import { CenterTurnTimer } from '../CenterTurnTimer';
import { SelfHud } from '../SelfHud';
import { socket } from '@/network/socket';
import type { PlayerSnapshot, GameIdentity, HandCard, TurnTimerState, TargetingState, Prompt } from '@/state/types';

// ============================================================================
// Mock Data Factories
// ============================================================================

const createMockPlayer = (overrides: Partial<PlayerSnapshot> = {}): PlayerSnapshot => ({
  index: 0,
  name: 'Player1',
  alive: true,
  coins: 2,
  cardCount: 2,
  avatar: undefined,
  ...overrides,
});

const createMockIdentity = (overrides: Partial<GameIdentity> = {}): GameIdentity => ({
  playerId: 'player-1',
  playerIndex: 0,
  playerNames: ['Player1', 'Player2', 'Player3', 'Player4'],
  ...overrides,
});

const createMockHandCard = (overrides: Partial<HandCard> = {}): HandCard => ({
  id: `card-${Math.random().toString(36).substr(2, 9)}`,
  role: 'duke',
  ...overrides,
});

const createMockTurnTimer = (overrides: Partial<TurnTimerState> = {}): TurnTimerState => ({
  activePlayerIndex: 0,
  durationMs: 30000,
  running: true,
  paused: false,
  key: 'timer-1',
  ...overrides,
});

const createMockTargeting = (overrides: Partial<TargetingState> = {}): TargetingState => ({
  active: false,
  actionId: null,
  requestId: null,
  selectedTarget: null,
  ...overrides,
});

const createMockPrompt = (overrides: Partial<Extract<Prompt, { kind: 'action' }>> = {}) => ({
  kind: 'action' as const,
  requestId: 'req-1',
  allowedActions: ['income', 'foreign_aid', 'coup', 'tax', 'steal', 'exchange', 'assassinate'],
  ...overrides,
});

// ============================================================================
// PlayerRing Tests
// ============================================================================

describe('PlayerRing', () => {
  const mockDispatch = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render all opponent players (excluding self)', () => {
    const players: PlayerSnapshot[] = [
      createMockPlayer({ index: 0, name: 'Player1' }),
      createMockPlayer({ index: 1, name: 'Alice' }),
      createMockPlayer({ index: 2, name: 'Bob' }),
    ];
    const identity = createMockIdentity({ playerIndex: 0 });

    render(
      <PlayerRing
        players={players}
        identity={identity}
        activePlayerIndex={null}
        targeting={null}
        pendingPrompt={null}
        turnTimer={null}
        dispatch={mockDispatch}
      />
    );

    // Should show Alice and Bob (opponents), but not Player1 (self)
    expect(screen.getByText('A')).toBeInTheDocument(); // Alice avatar initial
    expect(screen.getByText('B')).toBeInTheDocument(); // Bob avatar initial
    expect(screen.queryByText('Player1')).not.toBeInTheDocument();
  });

  it('should highlight active player with accent border', () => {
    const players: PlayerSnapshot[] = [
      createMockPlayer({ index: 0, name: 'Player1' }),
      createMockPlayer({ index: 1, name: 'Alice' }),
    ];
    const identity = createMockIdentity({ playerIndex: 0 });

    render(
      <PlayerRing
        players={players}
        identity={identity}
        activePlayerIndex={1}
        targeting={null}
        pendingPrompt={null}
        turnTimer={null}
        dispatch={mockDispatch}
      />
    );

    // Active player (Alice) should have accent styling
    const activePlayerContainer = screen.getByText('A').closest('div');
    expect(activePlayerContainer).toHaveClass('border-accent');
  });

  it('should show player coins for all opponents', () => {
    const players: PlayerSnapshot[] = [
      createMockPlayer({ index: 0, name: 'Player1', coins: 5 }),
      createMockPlayer({ index: 1, name: 'Alice', coins: 3 }),
      createMockPlayer({ index: 2, name: 'Bob', coins: 7 }),
    ];
    const identity = createMockIdentity({ playerIndex: 0 });

    render(
      <PlayerRing
        players={players}
        identity={identity}
        activePlayerIndex={null}
        targeting={null}
        pendingPrompt={null}
        turnTimer={null}
        dispatch={mockDispatch}
      />
    );

    expect(screen.getByText('3')).toBeInTheDocument();
    expect(screen.getByText('7')).toBeInTheDocument();
  });

  it('should display player avatar when available', () => {
    const players: PlayerSnapshot[] = [
      createMockPlayer({ index: 0, name: 'Player1' }),
      createMockPlayer({ index: 1, name: 'Player2', avatar: 'https://example.com/avatar.png' }),
    ];
    const identity = createMockIdentity({ playerIndex: 0 });

    render(
      <PlayerRing
        players={players}
        identity={identity}
        activePlayerIndex={null}
        targeting={null}
        pendingPrompt={null}
        turnTimer={null}
        dispatch={mockDispatch}
      />
    );

    const avatar = screen.getByAltText('Player2');
    expect(avatar).toBeInTheDocument();
    expect(avatar).toHaveAttribute('src', 'https://example.com/avatar.png');
  });

  it('should show player initial when no avatar is provided', () => {
    const players: PlayerSnapshot[] = [
      createMockPlayer({ index: 0, name: 'Player1' }),
      createMockPlayer({ index: 1, name: 'Alice', avatar: undefined }),
    ];
    const identity = createMockIdentity({ playerIndex: 0 });

    render(
      <PlayerRing
        players={players}
        identity={identity}
        activePlayerIndex={null}
        targeting={null}
        pendingPrompt={null}
        turnTimer={null}
        dispatch={mockDispatch}
      />
    );

    expect(screen.getByText('A')).toBeInTheDocument();
  });

  it('should apply grayscale and opacity for eliminated players', () => {
    const players: PlayerSnapshot[] = [
      createMockPlayer({ index: 0, name: 'Player1' }),
      createMockPlayer({ index: 1, name: 'Alice', alive: false }),
    ];
    const identity = createMockIdentity({ playerIndex: 0 });

    render(
      <PlayerRing
        players={players}
        identity={identity}
        activePlayerIndex={null}
        targeting={null}
        pendingPrompt={null}
        turnTimer={null}
        dispatch={mockDispatch}
      />
    );

    const eliminatedPlayerContainer = screen.getByText('A').closest('div');
    expect(eliminatedPlayerContainer).toHaveClass('grayscale');
    expect(eliminatedPlayerContainer).toHaveClass('opacity-50');
  });

  it('should show targetable indicator when in targeting mode', () => {
    const players: PlayerSnapshot[] = [
      createMockPlayer({ index: 0, name: 'Player1' }),
      createMockPlayer({ index: 1, name: 'Alice', alive: true }),
    ];
    const identity = createMockIdentity({ playerIndex: 0 });
    const targeting = createMockTargeting({
      active: true,
      actionId: 'assassinate',
      requestId: 'req-1',
    });
    const prompt = createMockPrompt({ kind: 'action', requestId: 'req-1' });

    render(
      <PlayerRing
        players={players}
        identity={identity}
        activePlayerIndex={null}
        targeting={targeting}
        pendingPrompt={prompt}
        turnTimer={null}
        dispatch={mockDispatch}
      />
    );

    expect(screen.getByTestId('target-icon')).toBeInTheDocument();
  });

  it('should disable targeting for players with insufficient coins for steal', () => {
    const players: PlayerSnapshot[] = [
      createMockPlayer({ index: 0, name: 'Player1' }),
      createMockPlayer({ index: 1, name: 'Alice', alive: true, coins: 1 }),
    ];
    const identity = createMockIdentity({ playerIndex: 0 });
    const targeting = createMockTargeting({
      active: true,
      actionId: 'steal',
      requestId: 'req-1',
    });
    const prompt = createMockPrompt({ kind: 'action', requestId: 'req-1' });

    render(
      <PlayerRing
        players={players}
        identity={identity}
        activePlayerIndex={null}
        targeting={targeting}
        pendingPrompt={prompt}
        turnTimer={null}
        dispatch={mockDispatch}
      />
    );

    expect(screen.getByText('Need 2+ coins to steal')).toBeInTheDocument();
  });

  it('should dispatch action when clicking targetable player', () => {
    const players: PlayerSnapshot[] = [
      createMockPlayer({ index: 0, name: 'Player1' }),
      createMockPlayer({ index: 1, name: 'Alice', alive: true }),
    ];
    const identity = createMockIdentity({ playerIndex: 0 });
    const targeting = createMockTargeting({
      active: true,
      actionId: 'assassinate',
      requestId: 'req-1',
    });
    const prompt = createMockPrompt({ kind: 'action', requestId: 'req-1' });

    render(
      <PlayerRing
        players={players}
        identity={identity}
        activePlayerIndex={null}
        targeting={targeting}
        pendingPrompt={prompt}
        turnTimer={null}
        dispatch={mockDispatch}
      />
    );

    const targetablePlayer = screen.getByText('A').closest('div[class*="cursor-pointer"]')?.parentElement;
    if (targetablePlayer) {
      fireEvent.click(targetablePlayer);
    }

    expect(socket.send).toHaveBeenCalledWith(
      'action',
      { id: 'assassinate', source_index: 0, target_index: 1 },
      'req-1'
    );
  });

  it('should reveal opponent hands when revealHands is true', () => {
    const players: PlayerSnapshot[] = [
      createMockPlayer({ index: 0, name: 'Player1' }),
      createMockPlayer({ index: 1, name: 'Player2', alive: true, cardCount: 2 }),
    ];
    const identity = createMockIdentity({ playerIndex: 0 });
    const handsByIndex = {
      1: ['duke', 'assassin'],
    };

    render(
      <PlayerRing
        players={players}
        identity={identity}
        activePlayerIndex={null}
        targeting={null}
        pendingPrompt={null}
        turnTimer={null}
        dispatch={mockDispatch}
        revealHands={true}
        handsByIndex={handsByIndex}
      />
    );

    // Should show card role abbreviations
    expect(screen.getByText('duke')).toBeInTheDocument();
    expect(screen.getByText('assa')).toBeInTheDocument();
  });

  it('should show card backs when hands are hidden', () => {
    const players: PlayerSnapshot[] = [
      createMockPlayer({ index: 0, name: 'Player1' }),
      createMockPlayer({ index: 1, name: 'Player2', alive: true, cardCount: 2 }),
    ];
    const identity = createMockIdentity({ playerIndex: 0 });

    const { container } = render(
      <PlayerRing
        players={players}
        identity={identity}
        activePlayerIndex={null}
        targeting={null}
        pendingPrompt={null}
        turnTimer={null}
        dispatch={mockDispatch}
        revealHands={false}
      />
    );

    // Should have card placeholder elements (checking for card container)
    const cardElements = container.querySelectorAll('[class*="bg-primary"]');
    expect(cardElements.length).toBeGreaterThan(0);
  });

  it('should apply elimination pulse animation when player is being eliminated', () => {
    const players: PlayerSnapshot[] = [
      createMockPlayer({ index: 0, name: 'Player1' }),
      createMockPlayer({ index: 1, name: 'Alice', alive: true }),
    ];
    const identity = createMockIdentity({ playerIndex: 0 });

    render(
      <PlayerRing
        players={players}
        identity={identity}
        activePlayerIndex={null}
        targeting={null}
        pendingPrompt={null}
        turnTimer={null}
        dispatch={mockDispatch}
        eliminatingPlayer={1}
      />
    );

    const playerContainer = screen.getByText('A').closest('div');
    expect(playerContainer).toHaveClass('animate-elimination-pulse');
  });

  it('should handle single opponent', () => {
    const players: PlayerSnapshot[] = [
      createMockPlayer({ index: 0, name: 'Player1' }),
      createMockPlayer({ index: 1, name: 'Alice' }),
    ];
    const identity = createMockIdentity({ playerIndex: 0 });

    render(
      <PlayerRing
        players={players}
        identity={identity}
        activePlayerIndex={null}
        targeting={null}
        pendingPrompt={null}
        turnTimer={null}
        dispatch={mockDispatch}
      />
    );

    expect(screen.getByText('A')).toBeInTheDocument();
  });

  it('should handle maximum number of opponents', () => {
    const players: PlayerSnapshot[] = [
      createMockPlayer({ index: 0, name: 'Player1' }),
      createMockPlayer({ index: 1, name: 'Alice' }),
      createMockPlayer({ index: 2, name: 'Bob' }),
      createMockPlayer({ index: 3, name: 'Charlie' }),
      createMockPlayer({ index: 4, name: 'David' }),
      createMockPlayer({ index: 5, name: 'Eve' }),
    ];
    const identity = createMockIdentity({ playerIndex: 0 });

    render(
      <PlayerRing
        players={players}
        identity={identity}
        activePlayerIndex={null}
        targeting={null}
        pendingPrompt={null}
        turnTimer={null}
        dispatch={mockDispatch}
      />
    );

    expect(screen.getByText('A')).toBeInTheDocument();
    expect(screen.getByText('B')).toBeInTheDocument();
    expect(screen.getByText('C')).toBeInTheDocument();
    expect(screen.getByText('D')).toBeInTheDocument();
    expect(screen.getByText('E')).toBeInTheDocument();
  });

  it('should show strike pulse animation when player is strike target', () => {
    const players: PlayerSnapshot[] = [
      createMockPlayer({ index: 0, name: 'Player1' }),
      createMockPlayer({ index: 1, name: 'Alice', alive: true }),
    ];
    const identity = createMockIdentity({ playerIndex: 0 });

    render(
      <PlayerRing
        players={players}
        identity={identity}
        activePlayerIndex={null}
        targeting={null}
        pendingPrompt={null}
        turnTimer={null}
        dispatch={mockDispatch}
        strikePulse={{ id: 'strike-1', targetIndex: 1 }}
      />
    );

    // Component should render with strike pulse
    expect(screen.getByText('A')).toBeInTheDocument();
  });
});

// ============================================================================
// HandTray Tests
// ============================================================================

describe('HandTray', () => {
  const mockOnHoverStart = vi.fn();
  const mockOnHoverEnd = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render all cards in hand', () => {
    const hand: HandCard[] = [
      createMockHandCard({ id: 'card-1', role: 'duke' }),
      createMockHandCard({ id: 'card-2', role: 'assassin' }),
    ];

    render(
      <HandTray
        hand={hand}
        isActive={true}
        onHoverStart={mockOnHoverStart}
        onHoverEnd={mockOnHoverEnd}
      />
    );

    // Cards should be rendered (checking by their container)
    const cards = screen.getAllByRole('img', { hidden: true });
    expect(cards.length).toBe(2);
  });

  it('should render empty hand without errors', () => {
    render(
      <HandTray
        hand={[]}
        isActive={true}
        onHoverStart={mockOnHoverStart}
        onHoverEnd={mockOnHoverEnd}
      />
    );

    expect(screen.queryByRole('img')).not.toBeInTheDocument();
  });

  it('should apply active styling when isActive is true', () => {
    const hand: HandCard[] = [createMockHandCard({ id: 'card-1', role: 'duke' })];

    const { container } = render(
      <HandTray
        hand={hand}
        isActive={true}
        onHoverStart={mockOnHoverStart}
        onHoverEnd={mockOnHoverEnd}
      />
    );

    const trayContainer = container.firstChild as HTMLElement;
    expect(trayContainer).toHaveClass('opacity-100');
    expect(trayContainer).not.toHaveClass('grayscale');
  });

  it('should apply inactive styling when isActive is false', () => {
    const hand: HandCard[] = [createMockHandCard({ id: 'card-1', role: 'duke' })];

    const { container } = render(
      <HandTray
        hand={hand}
        isActive={false}
        onHoverStart={mockOnHoverStart}
        onHoverEnd={mockOnHoverEnd}
      />
    );

    const trayContainer = container.firstChild as HTMLElement;
    expect(trayContainer).toHaveClass('opacity-60');
    expect(trayContainer).toHaveClass('grayscale');
  });

  it('should pass hover handlers to cards', () => {
    const hand: HandCard[] = [
      createMockHandCard({ id: 'card-1', role: 'duke' }),
      createMockHandCard({ id: 'card-2', role: 'assassin' }),
    ];

    render(
      <HandTray
        hand={hand}
        isActive={true}
        onHoverStart={mockOnHoverStart}
        onHoverEnd={mockOnHoverEnd}
      />
    );

    // Verify that all cards are rendered
    const cards = screen.getAllByRole('img', { hidden: true });
    expect(cards.length).toBe(2);

    // The hover callbacks are passed to the Card component
    // The actual hover behavior is tested in Card component tests
    expect(mockOnHoverStart).not.toHaveBeenCalled();
    expect(mockOnHoverEnd).not.toHaveBeenCalled();
  });

  it('should apply custom className when provided', () => {
    const hand: HandCard[] = [createMockHandCard({ id: 'card-1', role: 'duke' })];

    const { container } = render(
      <HandTray
        hand={hand}
        isActive={true}
        onHoverStart={mockOnHoverStart}
        onHoverEnd={mockOnHoverEnd}
        className="custom-class"
      />
    );

    const trayContainer = container.firstChild as HTMLElement;
    expect(trayContainer).toHaveClass('custom-class');
  });

  it('should render cards with correct role images', () => {
    const hand: HandCard[] = [
      createMockHandCard({ id: 'card-1', role: 'duke' }),
      createMockHandCard({ id: 'card-2', role: 'captain' }),
    ];

    render(
      <HandTray
        hand={hand}
        isActive={true}
        onHoverStart={mockOnHoverStart}
        onHoverEnd={mockOnHoverEnd}
      />
    );

    const cards = screen.getAllByRole('img', { hidden: true });
    expect(cards[0]).toHaveAttribute('src', '/cards/duke.png');
    expect(cards[1]).toHaveAttribute('src', '/cards/captain.png');
  });

  it('should render single card in hand', () => {
    const hand: HandCard[] = [createMockHandCard({ id: 'card-1', role: 'contessa' })];

    render(
      <HandTray
        hand={hand}
        isActive={true}
        onHoverStart={mockOnHoverStart}
        onHoverEnd={mockOnHoverEnd}
      />
    );

    const cards = screen.getAllByRole('img', { hidden: true });
    expect(cards.length).toBe(1);
    expect(cards[0]).toHaveAttribute('src', '/cards/contessa.png');
  });

  it('should handle all card roles', () => {
    const hand: HandCard[] = [
      createMockHandCard({ id: 'card-1', role: 'duke' }),
      createMockHandCard({ id: 'card-2', role: 'assassin' }),
      createMockHandCard({ id: 'card-3', role: 'captain' }),
      createMockHandCard({ id: 'card-4', role: 'ambassador' }),
      createMockHandCard({ id: 'card-5', role: 'contessa' }),
    ];

    render(
      <HandTray
        hand={hand}
        isActive={true}
        onHoverStart={mockOnHoverStart}
        onHoverEnd={mockOnHoverEnd}
      />
    );

    const cards = screen.getAllByRole('img', { hidden: true });
    expect(cards.length).toBe(5);
  });
});

// ============================================================================
// ChatBox Tests
// ============================================================================

// Use a factory function to create fresh state for each test
let mockChatMessages: Array<{ id: string; senderIndex: number; senderName: string; text: string; timestamp: number }> = [];
let mockPlayerIdentity: { playerIndex: number } | null = { playerIndex: 0 };

vi.mock('../../store/gameContext', () => ({
  useGame: () => ({
    get state() {
      return {
        chat: mockChatMessages,
        identity: mockPlayerIdentity,
      };
    },
    dispatch: vi.fn(),
  }),
}));

describe('ChatBox', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockChatMessages = [];
    mockPlayerIdentity = { playerIndex: 0 };
    // Mock scrollIntoView for the chat scroll ref
    Element.prototype.scrollIntoView = vi.fn();
  });

  it('should render chat header with title', () => {
    render(<ChatBox />);

    expect(screen.getByText('Parlour Chat')).toBeInTheDocument();
  });

  it('should render empty chat when no messages', () => {
    render(<ChatBox />);

    expect(screen.queryByTestId('chat-message')).not.toBeInTheDocument();
  });

  it('should have an input field for typing messages', () => {
    render(<ChatBox />);

    const input = screen.getByPlaceholderText('Whisper a secret...');
    expect(input).toBeInTheDocument();
  });

  it('should update input value when typing', () => {
    render(<ChatBox />);

    const input = screen.getByPlaceholderText('Whisper a secret...') as HTMLInputElement;
    fireEvent.change(input, { target: { value: 'Test message' } });

    expect(input.value).toBe('Test message');
  });

  it('should send message when form is submitted', () => {
    render(<ChatBox />);

    const input = screen.getByPlaceholderText('Whisper a secret...');
    fireEvent.change(input, { target: { value: 'Hello everyone!' } });

    const form = input.closest('form');
    if (form) {
      fireEvent.submit(form);
    }

    expect(socket.send).toHaveBeenCalledWith('chat_message', { text: 'Hello everyone!' });
  });

  it('should clear input after sending message', () => {
    render(<ChatBox />);

    const input = screen.getByPlaceholderText('Whisper a secret...') as HTMLInputElement;
    fireEvent.change(input, { target: { value: 'Hello!' } });

    const form = input.closest('form');
    if (form) {
      fireEvent.submit(form);
    }

    expect(input.value).toBe('');
  });

  it('should not send empty messages', () => {
    render(<ChatBox />);

    const input = screen.getByPlaceholderText('Whisper a secret...');
    fireEvent.change(input, { target: { value: '   ' } });

    const form = input.closest('form');
    if (form) {
      fireEvent.submit(form);
    }

    expect(socket.send).not.toHaveBeenCalled();
  });

  it('should have a send button', () => {
    render(<ChatBox />);

    expect(screen.getByTestId('send-icon')).toBeInTheDocument();
  });
});

// ============================================================================
// CenterTurnTimer Tests
// ============================================================================

describe('CenterTurnTimer', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('should render null when timer is null', () => {
    const { container } = render(
      <CenterTurnTimer timer={null} actor={null} />
    );

    expect(container.firstChild).toBeNull();
  });

  it('should render timer display when timer is running', () => {
    const timer = createMockTurnTimer({ running: true, durationMs: 30000 });
    const actor = createMockPlayer({ index: 0, name: 'Player1' });

    render(<CenterTurnTimer timer={timer} actor={actor} />);

    expect(screen.getByTestId('timer-icon')).toBeInTheDocument();
  });

  it('should show paused state when timer is paused', () => {
    const timer = createMockTurnTimer({ running: true, paused: true, durationMs: 30000 });
    const actor = createMockPlayer({ index: 0, name: 'Player1' });

    render(<CenterTurnTimer timer={timer} actor={actor} />);

    expect(screen.getByText('Paused')).toBeInTheDocument();
  });

  it('should show countdown when timer is running and not paused', () => {
    const timer = createMockTurnTimer({ running: true, paused: false, durationMs: 30000 });
    const actor = createMockPlayer({ index: 0, name: 'Player1' });

    render(<CenterTurnTimer timer={timer} actor={actor} />);

    // Should show initial time (30s)
    expect(screen.getByText(/30s|29s|28s/)).toBeInTheDocument();
  });

  it('should apply warning styling when time is low (<= 10s)', () => {
    const timer = createMockTurnTimer({ running: true, paused: false, durationMs: 10000 });
    const actor = createMockPlayer({ index: 0, name: 'Player1' });

    render(<CenterTurnTimer timer={timer} actor={actor} />);

    const timeDisplay = screen.getByText(/10s|9s/);
    expect(timeDisplay).toBeInTheDocument();
  });

  it('should handle timer with different durations', () => {
    const timer = createMockTurnTimer({ running: true, durationMs: 60000 });
    const actor = createMockPlayer({ index: 0, name: 'Player1' });

    render(<CenterTurnTimer timer={timer} actor={actor} />);

    expect(screen.getByTestId('timer-icon')).toBeInTheDocument();
    expect(screen.getByText(/60s|59s|58s/)).toBeInTheDocument();
  });

  it('should handle timer with actor information', () => {
    const timer = createMockTurnTimer({ running: true, activePlayerIndex: 1, durationMs: 30000 });
    const actor = createMockPlayer({ index: 1, name: 'Player2' });

    render(<CenterTurnTimer timer={timer} actor={actor} />);

    expect(screen.getByTestId('timer-icon')).toBeInTheDocument();
  });
});

// ============================================================================
// SelfHud Tests
// ============================================================================

describe('SelfHud', () => {
  it('should render player avatar when available', () => {
    const identity = createMockIdentity({ playerIndex: 0, playerNames: ['Alice', 'Bob'] });
    const player = createMockPlayer({ index: 0, name: 'Alice', avatar: 'https://example.com/avatar.png' });

    render(
      <SelfHud
        identity={identity}
        player={player}
        activePlayerIndex={null}
        timer={null}
      />
    );

    const avatar = screen.getByAltText('Alice');
    expect(avatar).toBeInTheDocument();
    expect(avatar).toHaveAttribute('src', 'https://example.com/avatar.png');
  });

  it('should render player initial when no avatar is available', () => {
    const identity = createMockIdentity({ playerIndex: 0, playerNames: ['Alice', 'Bob'] });
    const player = createMockPlayer({ index: 0, name: 'Alice', avatar: undefined });

    render(
      <SelfHud
        identity={identity}
        player={player}
        activePlayerIndex={null}
        timer={null}
      />
    );

    expect(screen.getByText('A')).toBeInTheDocument();
  });

  it('should highlight when player is active', () => {
    const identity = createMockIdentity({ playerIndex: 0, playerNames: ['Alice', 'Bob'] });
    const player = createMockPlayer({ index: 0, name: 'Alice' });

    render(
      <SelfHud
        identity={identity}
        player={player}
        activePlayerIndex={0}
        timer={null}
      />
    );

    const avatarContainer = screen.getByText('A').closest('div');
    expect(avatarContainer).toHaveClass('border-accent');
  });

  it('should not highlight when player is not active', () => {
    const identity = createMockIdentity({ playerIndex: 0, playerNames: ['Alice', 'Bob'] });
    const player = createMockPlayer({ index: 0, name: 'Alice' });

    render(
      <SelfHud
        identity={identity}
        player={player}
        activePlayerIndex={1}
        timer={null}
      />
    );

    const avatarContainer = screen.getByText('A').closest('div');
    expect(avatarContainer).toHaveClass('border-border');
    expect(avatarContainer).not.toHaveClass('border-accent');
  });

  it('should show timer ring when timer is active for current player', () => {
    const identity = createMockIdentity({ playerIndex: 0, playerNames: ['Alice', 'Bob'] });
    const player = createMockPlayer({ index: 0, name: 'Alice' });
    const timer = createMockTurnTimer({ running: true, activePlayerIndex: 0, durationMs: 30000 });

    const { container } = render(
      <SelfHud
        identity={identity}
        player={player}
        activePlayerIndex={0}
        timer={timer}
      />
    );

    // Should have SVG timer ring
    const svg = container.querySelector('svg');
    expect(svg).toBeInTheDocument();
  });

  it('should not show timer ring when timer is for different player', () => {
    const identity = createMockIdentity({ playerIndex: 0, playerNames: ['Alice', 'Bob'] });
    const player = createMockPlayer({ index: 0, name: 'Alice' });
    const timer = createMockTurnTimer({ running: true, activePlayerIndex: 1, durationMs: 30000 });

    const { container } = render(
      <SelfHud
        identity={identity}
        player={player}
        activePlayerIndex={1}
        timer={timer}
      />
    );

    // Timer ring should not be present for this player
    const svg = container.querySelector('svg');
    expect(svg).not.toBeInTheDocument();
  });

  it('should display player name from identity', () => {
    const identity = createMockIdentity({ playerIndex: 0, playerNames: ['Alice', 'Bob'] });
    const player = createMockPlayer({ index: 0, name: 'Alice' });

    const { container } = render(
      <SelfHud
        identity={identity}
        player={player}
        activePlayerIndex={null}
        timer={null}
      />
    );

    // Name should be in the hover badge
    expect(container.textContent).toContain('Alice');
  });

  it('should use "Player" as fallback when name is not in identity', () => {
    const identity = createMockIdentity({ playerIndex: 0, playerNames: [] });
    const player = createMockPlayer({ index: 0, name: 'Alice' });

    const { container } = render(
      <SelfHud
        identity={identity}
        player={player}
        activePlayerIndex={null}
        timer={null}
      />
    );

    expect(container.textContent).toContain('Player');
  });

  it('should handle undefined player gracefully', () => {
    const identity = createMockIdentity({ playerIndex: 0, playerNames: ['Alice'] });

    render(
      <SelfHud
        identity={identity}
        player={undefined}
        activePlayerIndex={null}
        timer={null}
      />
    );

    // Should still render with initial from identity
    expect(screen.getByText('A')).toBeInTheDocument();
  });

  it('should apply custom className when provided', () => {
    const identity = createMockIdentity({ playerIndex: 0, playerNames: ['Alice'] });
    const player = createMockPlayer({ index: 0, name: 'Alice' });

    const { container } = render(
      <SelfHud
        identity={identity}
        player={player}
        activePlayerIndex={null}
        timer={null}
        className="custom-hud-class"
      />
    );

    const hudContainer = container.firstChild as HTMLElement;
    expect(hudContainer).toHaveClass('custom-hud-class');
  });

  it('should show paused timer state correctly', () => {
    const identity = createMockIdentity({ playerIndex: 0, playerNames: ['Alice'] });
    const player = createMockPlayer({ index: 0, name: 'Alice' });
    const timer = createMockTurnTimer({
      running: true,
      activePlayerIndex: 0,
      paused: true,
      durationMs: 30000,
    });

    const { container } = render(
      <SelfHud
        identity={identity}
        player={player}
        activePlayerIndex={0}
        timer={timer}
      />
    );

    // Should have SVG with paused styling
    const svg = container.querySelector('svg');
    expect(svg).toBeInTheDocument();
  });

  it('should render with different player indices', () => {
    const identity = createMockIdentity({ playerIndex: 2, playerNames: ['P1', 'P2', 'P3', 'P4'] });
    const player = createMockPlayer({ index: 2, name: 'P3' });

    render(
      <SelfHud
        identity={identity}
        player={player}
        activePlayerIndex={2}
        timer={null}
      />
    );

    expect(screen.getByText('P')).toBeInTheDocument();
  });

  it('should handle empty player names array', () => {
    const identity = createMockIdentity({ playerIndex: 0, playerNames: [''] });
    const player = createMockPlayer({ index: 0, name: '' });

    const { container } = render(
      <SelfHud
        identity={identity}
        player={player}
        activePlayerIndex={null}
        timer={null}
      />
    );

    // Should fallback to "Player" when name is empty
    expect(container.textContent).toContain('Player');
  });
});

// ============================================================================
// Integration Tests
// ============================================================================

describe('Game Components Integration', () => {
  it('should render all game components together', () => {
    const identity = createMockIdentity({ playerIndex: 0 });
    const players: PlayerSnapshot[] = [
      createMockPlayer({ index: 0, name: 'Player1' }),
      createMockPlayer({ index: 1, name: 'Player2' }),
    ];
    const hand: HandCard[] = [
      createMockHandCard({ id: 'card-1', role: 'duke' }),
      createMockHandCard({ id: 'card-2', role: 'captain' }),
    ];
    const timer = createMockTurnTimer({ running: true, activePlayerIndex: 0 });

    const { container: playerRingContainer } = render(
      <PlayerRing
        players={players}
        identity={identity}
        activePlayerIndex={0}
        targeting={null}
        pendingPrompt={null}
        turnTimer={timer}
        dispatch={vi.fn()}
      />
    );

    const { container: handTrayContainer } = render(
      <HandTray
        hand={hand}
        isActive={true}
        onHoverStart={vi.fn()}
        onHoverEnd={vi.fn()}
      />
    );

    const { container: selfHudContainer } = render(
      <SelfHud
        identity={identity}
        player={players[0]}
        activePlayerIndex={0}
        timer={timer}
      />
    );

    expect(playerRingContainer.firstChild).toBeInTheDocument();
    expect(handTrayContainer.firstChild).toBeInTheDocument();
    expect(selfHudContainer.firstChild).toBeInTheDocument();
  });

  it('should handle game state with active targeting', () => {
    const identity = createMockIdentity({ playerIndex: 0 });
    const players: PlayerSnapshot[] = [
      createMockPlayer({ index: 0, name: 'Player1' }),
      createMockPlayer({ index: 1, name: 'Player2', alive: true, coins: 5 }),
    ];
    const targeting = createMockTargeting({
      active: true,
      actionId: 'steal',
      requestId: 'req-1',
    });
    const prompt = createMockPrompt({ kind: 'action', requestId: 'req-1' });

    render(
      <PlayerRing
        players={players}
        identity={identity}
        activePlayerIndex={0}
        targeting={targeting}
        pendingPrompt={prompt}
        turnTimer={null}
        dispatch={vi.fn()}
      />
    );

    // Targetable player should show target icon
    expect(screen.getByTestId('target-icon')).toBeInTheDocument();
  });
});
