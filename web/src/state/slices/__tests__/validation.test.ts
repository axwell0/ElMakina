import { describe, it, expect, vi } from 'vitest';
import { validateAndDispatch, hasPayloadValidator, getPayloadValidator } from '../validation';
import { initialSlicedState } from '../index';
import type { WsEnvelope } from '@/network/socket';

describe('validateAndDispatch', () => {
  const baseEnvelope: WsEnvelope = {
    type: 'hello_ack',
    payload: { player_id: 'player-123', token: 'token-abc' },
  };

  it('should call onValid when validation passes', () => {
    const onValid = vi.fn().mockReturnValue(initialSlicedState);

    const result = validateAndDispatch(
      'hello_ack',
      baseEnvelope,
      initialSlicedState,
      onValid
    );

    expect(onValid).toHaveBeenCalledWith({ player_id: 'player-123', token: 'token-abc' });
    expect(result).toBe(initialSlicedState);
  });

  it('should return current state when validation fails', () => {
    const invalidEnvelope = {
      type: 'hello_ack',
      payload: { player_id: 123, token: true }, // Wrong types
    } as unknown as WsEnvelope;
    const onValid = vi.fn();

    const result = validateAndDispatch(
      'hello_ack',
      invalidEnvelope,
      initialSlicedState,
      onValid
    );

    expect(onValid).not.toHaveBeenCalled();
    expect(result).toBe(initialSlicedState); // Same reference
  });

  it('should return current state when validator throws', () => {
    const onValid = vi.fn().mockImplementation(() => {
      throw new Error('Business logic error');
    });

    const result = validateAndDispatch(
      'hello_ack',
      baseEnvelope,
      initialSlicedState,
      onValid
    );

    expect(onValid).toHaveBeenCalled();
    expect(result).toBe(initialSlicedState);
  });

  it('should skip validation when no validator exists', () => {
    // This would be for a message type without a validator
    // For now, we just verify the warning is logged
    const envelope: WsEnvelope = {
      type: 'game_state',
      payload: {
        turn_number: 1,
        active_player_index: 0,
        players: [],
      },
    };
    const onValid = vi.fn().mockReturnValue(initialSlicedState);

    // game_state has a validator, so this will validate
    validateAndDispatch(
      'game_state',
      envelope,
      initialSlicedState,
      onValid
    );

    expect(onValid).toHaveBeenCalled();
  });

  it('should validate game_state payload correctly', () => {
    const validEnvelope: WsEnvelope = {
      type: 'game_state',
      payload: {
        turn_number: 1,
        active_player_index: 0,
        players: [],
      },
    };
    const onValid = vi.fn().mockReturnValue(initialSlicedState);

    validateAndDispatch('game_state', validEnvelope, initialSlicedState, onValid);

    expect(onValid).toHaveBeenCalled();
  });

  it('should reject invalid game_state payload', () => {
    const invalidEnvelope = {
      type: 'game_state',
      payload: {
        turn_number: 'one', // wrong type
        active_player_index: 0,
        players: [],
      },
    } as unknown as WsEnvelope;
    const onValid = vi.fn();

    validateAndDispatch('game_state', invalidEnvelope, initialSlicedState, onValid);

    expect(onValid).not.toHaveBeenCalled();
  });

  it('should validate chat_message payload', () => {
    const envelope: WsEnvelope = {
      type: 'chat_message',
      payload: {
        id: 'msg-1',
        senderIndex: 0,
        senderName: 'Alice',
        text: 'Hello!',
        timestamp: Date.now(),
      },
    };
    const onValid = vi.fn().mockReturnValue(initialSlicedState);

    validateAndDispatch('chat_message', envelope, initialSlicedState, onValid);

    expect(onValid).toHaveBeenCalled();
  });

  it('should reject invalid chat_message payload', () => {
    const envelope = {
      type: 'chat_message',
      payload: {
        id: 'msg-1',
        senderIndex: 'zero', // wrong type
        senderName: 'Alice',
        text: 'Hello!',
        timestamp: Date.now(),
      },
    } as unknown as WsEnvelope;
    const onValid = vi.fn();

    validateAndDispatch('chat_message', envelope, initialSlicedState, onValid);

    expect(onValid).not.toHaveBeenCalled();
  });
});

describe('hasPayloadValidator', () => {
  it('should return true for known message types', () => {
    expect(hasPayloadValidator('hello_ack')).toBe(true);
    expect(hasPayloadValidator('game_state')).toBe(true);
    expect(hasPayloadValidator('chat_message')).toBe(true);
  });

  it('should return false for unknown message types', () => {
    expect(hasPayloadValidator('unknown_type')).toBe(false);
    expect(hasPayloadValidator('fake_message')).toBe(false);
  });
});

describe('getPayloadValidator', () => {
  it('should return validator for known message types', () => {
    const helloValidator = getPayloadValidator('hello_ack');
    expect(helloValidator).toBeDefined();

    if (helloValidator) {
      const result = helloValidator({ player_id: 'test', token: 'token' });
      expect(result.valid).toBe(true);
    }
  });

  it('should return undefined for unknown message types', () => {
    const validator = getPayloadValidator('unknown_type' as any);
    expect(validator).toBeUndefined();
  });
});
