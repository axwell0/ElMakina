import { describe, it, expect } from 'vitest';
import {
  validateEnvelopeStructure,
  isValidInboundMessageType,
  INBOUND_MESSAGE_TYPES,
} from '../envelope';
import type { EnvelopeStructure } from '../types';

describe('validateEnvelopeStructure (L1 Validation)', () => {
  describe('valid envelopes', () => {
    it('should validate envelope with type and payload', () => {
      const envelope = {
        type: 'game_state',
        payload: { turn_number: 1 },
      };

      const result = validateEnvelopeStructure(envelope);

      expect(result.valid).toBe(true);
      expect(result.data).toEqual(envelope);
      expect(result.errors).toHaveLength(0);
    });

    it('should validate envelope with type only (no payload)', () => {
      const envelope = { type: 'lobby_list' };

      const result = validateEnvelopeStructure(envelope);

      expect(result.valid).toBe(true);
      expect(result.data).toEqual({ type: 'lobby_list' });
    });

    it('should validate envelope with request_id', () => {
      const envelope = {
        type: 'game_state',
        request_id: 'req-123',
        payload: {},
      };

      const result = validateEnvelopeStructure(envelope);

      expect(result.valid).toBe(true);
      expect(result.data?.request_id).toBe('req-123');
    });

    it('should validate envelope with null payload', () => {
      const envelope = {
        type: 'game_state',
        payload: null,
      };

      const result = validateEnvelopeStructure(envelope);

      // Null payload is allowed (payload is optional)
      expect(result.valid).toBe(true);
    });

    it('should validate envelope with undefined payload', () => {
      const envelope = {
        type: 'game_state',
        payload: undefined,
      };

      const result = validateEnvelopeStructure(envelope);

      expect(result.valid).toBe(true);
    });
  });

  describe('invalid envelopes - not objects', () => {
    it('should reject string', () => {
      const result = validateEnvelopeStructure('not an object');

      expect(result.valid).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].code).toBe('NOT_OBJECT');
      expect(result.errors[0].path).toBe('');
    });

    it('should reject number', () => {
      const result = validateEnvelopeStructure(42);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('NOT_OBJECT');
    });

    it('should reject null', () => {
      const result = validateEnvelopeStructure(null);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('NOT_OBJECT');
    });

    it('should reject array', () => {
      const result = validateEnvelopeStructure(['not', 'valid']);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('NOT_OBJECT');
    });

    it('should reject boolean', () => {
      const result = validateEnvelopeStructure(true);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('NOT_OBJECT');
    });
  });

  describe('invalid envelopes - missing or invalid type', () => {
    it('should reject envelope without type', () => {
      const envelope = { payload: {} };

      const result = validateEnvelopeStructure(envelope);

      expect(result.valid).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].code).toBe('MISSING_TYPE');
      expect(result.errors[0].path).toBe('type');
    });

    it('should reject envelope with empty string type', () => {
      const envelope = { type: '' };

      const result = validateEnvelopeStructure(envelope);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('INVALID_TYPE');
    });

    it('should reject envelope with number type', () => {
      const envelope = { type: 123 };

      const result = validateEnvelopeStructure(envelope);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('INVALID_TYPE');
    });

    it('should reject envelope with boolean type', () => {
      const envelope = { type: true };

      const result = validateEnvelopeStructure(envelope);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('INVALID_TYPE');
    });

    it('should reject envelope with null type', () => {
      const envelope = { type: null };

      const result = validateEnvelopeStructure(envelope);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('INVALID_TYPE');
    });

    it('should reject envelope with object type', () => {
      const envelope = { type: { nested: 'object' } };

      const result = validateEnvelopeStructure(envelope);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('INVALID_TYPE');
    });
  });

  describe('invalid envelopes - invalid request_id', () => {
    it('should reject envelope with number request_id', () => {
      const envelope = {
        type: 'game_state',
        request_id: 123,
      };

      const result = validateEnvelopeStructure(envelope);

      expect(result.valid).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].code).toBe('INVALID_REQUEST_ID');
      expect(result.errors[0].path).toBe('request_id');
    });

    it('should reject envelope with object request_id', () => {
      const envelope = {
        type: 'game_state',
        request_id: { id: '123' },
      };

      const result = validateEnvelopeStructure(envelope);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('INVALID_REQUEST_ID');
    });

    it('should allow string request_id', () => {
      const envelope = {
        type: 'game_state',
        request_id: 'valid-id',
      };

      const result = validateEnvelopeStructure(envelope);

      expect(result.valid).toBe(true);
    });

    it('should allow undefined request_id', () => {
      const envelope = {
        type: 'game_state',
        request_id: undefined,
      };

      const result = validateEnvelopeStructure(envelope);

      expect(result.valid).toBe(true);
    });

    it('should allow missing request_id', () => {
      const envelope = { type: 'game_state' };

      const result = validateEnvelopeStructure(envelope);

      expect(result.valid).toBe(true);
    });
  });

  describe('multiple errors', () => {
    it('should collect multiple validation errors', () => {
      const envelope = {
        type: '',
        request_id: 123,
      };

      const result = validateEnvelopeStructure(envelope);

      expect(result.valid).toBe(false);
      expect(result.errors).toHaveLength(2);
      expect(result.errors.map((e) => e.code)).toContain('INVALID_TYPE');
      expect(result.errors.map((e) => e.code)).toContain('INVALID_REQUEST_ID');
    });
  });
});

describe('isValidInboundMessageType (L2 Validation)', () => {
  it('should return true for known message types', () => {
    INBOUND_MESSAGE_TYPES.forEach((type) => {
      expect(isValidInboundMessageType(type)).toBe(true);
    });
  });

  it('should return false for unknown message types', () => {
    expect(isValidInboundMessageType('unknown_type')).toBe(false);
    expect(isValidInboundMessageType('fake_message')).toBe(false);
    expect(isValidInboundMessageType('not_real')).toBe(false);
  });

  it('should return false for empty string', () => {
    expect(isValidInboundMessageType('')).toBe(false);
  });

  it('should narrow type correctly', () => {
    const type = 'game_state' as string;

    if (isValidInboundMessageType(type)) {
      // TypeScript should know this is InboundMessageType
      const _inboundType: 'game_state' = type;
      expect(_inboundType).toBe('game_state');
    }
  });

  it('should handle all game-related message types', () => {
    const gameTypes = [
      'game_state',
      'request_action',
      'challenge_window',
      'counter_window',
      'request_step',
      'hand_state',
      'game_over',
      'player_eliminated',
    ];

    gameTypes.forEach((type) => {
      expect(isValidInboundMessageType(type)).toBe(true);
    });
  });

  it('should handle all lobby-related message types', () => {
    const lobbyTypes = [
      'lobby_list_result',
      'lobby_created',
      'lobby_joined',
      'lobby_state',
      'lobby_started',
    ];

    lobbyTypes.forEach((type) => {
      expect(isValidInboundMessageType(type)).toBe(true);
    });
  });

  it('should handle all UI-related message types', () => {
    const uiTypes = ['game_log', 'investigate_result', 'chat_message'];

    uiTypes.forEach((type) => {
      expect(isValidInboundMessageType(type)).toBe(true);
    });
  });
});

describe('Integration: L1 + L2 validation pipeline', () => {
  it('should validate envelope structure before checking type whitelist', () => {
    const invalidEnvelope = { payload: {} };

    // L1 should fail
    const l1Result = validateEnvelopeStructure(invalidEnvelope);
    expect(l1Result.valid).toBe(false);

    // L2 should not be attempted (no type to check)
    if (l1Result.valid && l1Result.data) {
      // This code path should not execute
      expect(isValidInboundMessageType(l1Result.data.type)).toBe(false);
    }
  });

  it('should pass valid envelope through both L1 and L2', () => {
    const envelope: EnvelopeStructure = {
      type: 'game_state',
      payload: { turn_number: 1 },
    };

    // L1: Structure validation
    const l1Result = validateEnvelopeStructure(envelope);
    expect(l1Result.valid).toBe(true);

    // L2: Type whitelist validation
    if (l1Result.valid && l1Result.data) {
      expect(isValidInboundMessageType(l1Result.data.type)).toBe(true);
    }
  });

  it('should reject unknown message type even with valid structure', () => {
    const envelope: EnvelopeStructure = {
      type: 'unknown_type',
      payload: {},
    };

    // L1: Should pass
    const l1Result = validateEnvelopeStructure(envelope);
    expect(l1Result.valid).toBe(true);

    // L2: Should fail
    if (l1Result.valid && l1Result.data) {
      expect(isValidInboundMessageType(l1Result.data.type)).toBe(false);
    }
  });
});
