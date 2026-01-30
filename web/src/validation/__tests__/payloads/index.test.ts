import { describe, it, expect } from 'vitest';
import {
  validateHelloAckPayload,
  validateHelloErrorPayload,
} from '../../payloads/hello';
import {
  validateLobbyCreatedPayload,
  validateLobbyJoinedPayload,
  validateLobbyListResultPayload,
  validateLobbyStatePayload,
} from '../../payloads/lobby';
import {
  validateGameStatePayload,
  validateGameOverPayload,
  validatePlayerEliminatedPayload,
} from '../../payloads/game';
import {
  validateInvestigateResultPayload,
  validateChatMessagePayload,
  validateGameLogPayload,
} from '../../payloads/ui';
import {
  validateGamePausedPayload,
  validateGameResumedPayload,
} from '../../payloads/pause';

describe('Hello Payload Validators', () => {
  describe('validateHelloAckPayload', () => {
    it('should validate valid hello_ack payload', () => {
      const payload = { player_id: 'player-123', token: 'token-abc' };
      const result = validateHelloAckPayload(payload);

      expect(result.valid).toBe(true);
      expect(result.data).toEqual(payload);
    });

    it('should reject missing player_id', () => {
      const payload = { token: 'token-abc' };
      const result = validateHelloAckPayload(payload);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('MISSING_FIELD');
    });

    it('should reject missing token', () => {
      const payload = { player_id: 'player-123' };
      const result = validateHelloAckPayload(payload);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('MISSING_FIELD');
    });

    it('should reject wrong types', () => {
      const payload = { player_id: 123, token: true };
      const result = validateHelloAckPayload(payload);

      expect(result.valid).toBe(false);
      expect(result.errors.some((e) => e.code === 'TYPE_MISMATCH')).toBe(true);
    });
  });

  describe('validateHelloErrorPayload', () => {
    it('should validate valid hello_error payload', () => {
      const payload = { error: 'Nickname taken' };
      const result = validateHelloErrorPayload(payload);

      expect(result.valid).toBe(true);
    });

    it('should reject missing error', () => {
      const payload = {};
      const result = validateHelloErrorPayload(payload);

      expect(result.valid).toBe(false);
    });
  });
});

describe('Lobby Payload Validators', () => {
  describe('validateLobbyCreatedPayload', () => {
    it('should validate valid lobby_created payload', () => {
      const payload = { lobby_id: 'lobby-123' };
      const result = validateLobbyCreatedPayload(payload);

      expect(result.valid).toBe(true);
    });
  });

  describe('validateLobbyJoinedPayload', () => {
    it('should validate valid lobby_joined payload', () => {
      const payload = { lobby_id: 'lobby-123' };
      const result = validateLobbyJoinedPayload(payload);

      expect(result.valid).toBe(true);
    });
  });

  describe('validateLobbyListResultPayload', () => {
    it('should validate valid lobby_list_result payload', () => {
      const payload = { lobbies: [] };
      const result = validateLobbyListResultPayload(payload);

      expect(result.valid).toBe(true);
    });

    it('should validate with lobbies array', () => {
      const payload = {
        lobbies: [
          { id: 'lobby-1', leader_nick: 'Alice', leader_id: 'player-1', player_count: 2, status: 'open' },
        ],
      };
      const result = validateLobbyListResultPayload(payload);

      expect(result.valid).toBe(true);
    });
  });

  describe('validateLobbyStatePayload', () => {
    it('should validate valid lobby_state payload', () => {
      const payload = {
        lobby_id: 'lobby-123',
        leader_nick: 'Alice',
        leader_id: 'player-1',
        player_nicks: ['Alice', 'Bob'],
        player_ids: ['player-1', 'player-2'],
        player_count: 2,
        status: 'open',
      };
      const result = validateLobbyStatePayload(payload);

      expect(result.valid).toBe(true);
    });

    it('should validate with optional player_avatars', () => {
      const payload = {
        lobby_id: 'lobby-123',
        leader_nick: 'Alice',
        leader_id: 'player-1',
        player_nicks: ['Alice', 'Bob'],
        player_ids: ['player-1', 'player-2'],
        player_count: 2,
        status: 'open',
        player_avatars: ['avatar1', 'avatar2'],
      };
      const result = validateLobbyStatePayload(payload);

      expect(result.valid).toBe(true);
    });

    it('should reject missing required fields', () => {
      const payload = {
        lobby_id: 'lobby-123',
        // missing leader_nick, leader_id, etc.
      };
      const result = validateLobbyStatePayload(payload);

      expect(result.valid).toBe(false);
    });
  });
});

describe('Game Payload Validators', () => {
  describe('validateGameStatePayload', () => {
    it('should validate valid game_state payload', () => {
      const payload = {
        turn_number: 1,
        active_player_index: 0,
        players: [],
      };
      const result = validateGameStatePayload(payload);

      expect(result.valid).toBe(true);
    });

    it('should validate with players', () => {
      const payload = {
        turn_number: 5,
        active_player_index: 1,
        players: [
          { index: 0, name: 'Alice', coins: 5, card_count: 2, alive: true },
          { index: 1, name: 'Bob', coins: 3, card_count: 2, alive: true },
        ],
      };
      const result = validateGameStatePayload(payload);

      expect(result.valid).toBe(true);
    });

    it('should reject wrong turn_number type', () => {
      const payload = {
        turn_number: 'five',
        active_player_index: 0,
        players: [],
      };
      const result = validateGameStatePayload(payload);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('TYPE_MISMATCH');
    });
  });

  describe('validateGameOverPayload', () => {
    it('should validate valid game_over payload', () => {
      const payload = { winner_index: 0, winner_name: 'Alice' };
      const result = validateGameOverPayload(payload);

      expect(result.valid).toBe(true);
    });

    it('should reject missing winner_index', () => {
      const payload = { winner_name: 'Alice' };
      const result = validateGameOverPayload(payload);

      expect(result.valid).toBe(false);
    });
  });

  describe('validatePlayerEliminatedPayload', () => {
    it('should validate valid player_eliminated payload', () => {
      const payload = { player_index: 1, reason: 'Lost influence', turn: 5 };
      const result = validatePlayerEliminatedPayload(payload);

      expect(result.valid).toBe(true);
    });

    it('should reject missing fields', () => {
      const payload = { player_index: 1 };
      const result = validatePlayerEliminatedPayload(payload);

      expect(result.valid).toBe(false);
    });
  });
});

describe('UI Payload Validators', () => {
  describe('validateInvestigateResultPayload', () => {
    it('should validate valid investigate_result payload', () => {
      const payload = { target_name: 'Bob', role: 'Thief' };
      const result = validateInvestigateResultPayload(payload);

      expect(result.valid).toBe(true);
    });

    it('should reject missing fields', () => {
      const payload = { target_name: 'Bob' };
      const result = validateInvestigateResultPayload(payload);

      expect(result.valid).toBe(false);
    });
  });

  describe('validateChatMessagePayload', () => {
    it('should validate valid chat_message payload', () => {
      const payload = {
        id: 'msg-1',
        senderIndex: 0,
        senderName: 'Alice',
        text: 'Hello!',
        timestamp: Date.now(),
      };
      const result = validateChatMessagePayload(payload);

      expect(result.valid).toBe(true);
    });

    it('should reject missing id', () => {
      const payload = {
        senderIndex: 0,
        senderName: 'Alice',
        text: 'Hello!',
        timestamp: Date.now(),
      };
      const result = validateChatMessagePayload(payload);

      expect(result.valid).toBe(false);
    });

    it('should reject wrong senderIndex type', () => {
      const payload = {
        id: 'msg-1',
        senderIndex: 'zero',
        senderName: 'Alice',
        text: 'Hello!',
        timestamp: Date.now(),
      };
      const result = validateChatMessagePayload(payload);

      expect(result.valid).toBe(false);
    });
  });

  describe('validateGameLogPayload', () => {
    it('should validate valid game_log payload', () => {
      const payload = { turn: 1, scope: 'public', message: 'Game started' };
      const result = validateGameLogPayload(payload);

      expect(result.valid).toBe(true);
    });

    it('should validate with optional player_index', () => {
      const payload = { turn: 1, scope: 'private', message: 'Your turn', player_index: 0 };
      const result = validateGameLogPayload(payload);

      expect(result.valid).toBe(true);
    });

    it('should reject missing required fields', () => {
      const payload = { turn: 1 };
      const result = validateGameLogPayload(payload);

      expect(result.valid).toBe(false);
    });
  });
});

describe('Pause Payload Validators', () => {
  describe('validateGamePausedPayload', () => {
    it('should validate valid game_paused payload', () => {
      const payload = {
        paused_by_player_id: 'player-1',
        paused_by_index: 0,
        paused_by_name: 'Alice',
        deadline_ms: Date.now() + 60000,
        duration_ms: 60000,
        pause_reason: 'Player disconnected',
        eligible_voters: [1, 2, 3],
        kick_votes: [],
      };
      const result = validateGamePausedPayload(payload);

      expect(result.valid).toBe(true);
    });

    it('should reject missing required fields', () => {
      const payload = { paused_by_player_id: 'player-1' };
      const result = validateGamePausedPayload(payload);

      expect(result.valid).toBe(false);
    });
  });

  describe('validateGameResumedPayload', () => {
    it('should validate valid game_resumed payload', () => {
      const payload = {
        resumed_by_player_id: 'player-1',
        resumed_by_index: 0,
        resumed_by_name: 'Alice',
        resume_reason: 'Player reconnected',
      };
      const result = validateGameResumedPayload(payload);

      expect(result.valid).toBe(true);
    });

    it('should reject missing fields', () => {
      const payload = { resume_reason: 'Player reconnected' };
      const result = validateGameResumedPayload(payload);

      expect(result.valid).toBe(false);
    });
  });
});
