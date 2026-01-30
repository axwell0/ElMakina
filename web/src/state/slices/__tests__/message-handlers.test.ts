import { describe, it, expect } from 'vitest';
import { gameReducer, gameActions, initialGameSliceState } from '@/state/slices/game';
import { uiReducer, uiActions, initialUIState } from '@/state/slices/ui';
import { rootReducer, initialSlicedState } from '@/state/slices';
import type { WsEnvelope } from '@/network/socket';

describe('Message Handlers', () => {
  describe('game_paused', () => {
    it('should update pause state when game is paused', () => {
      const payload = {
        paused_by_player_id: 'player-123',
        paused_by_index: 0,
        paused_by_name: 'Alice',
        deadline_ms: Date.now() + 60000,
        duration_ms: 60000,
        pause_reason: 'Player disconnected',
        eligible_voters: [1, 2, 3],
        kick_votes: [],
      };

      const action = gameActions.gamePaused(payload);
      const newState = gameReducer(initialGameSliceState, action);

      expect(newState.pause.status).toBe('paused');
      expect((newState.pause as any).pausedByName).toBe('Alice');
      expect((newState.pause as any).eligibleVoters).toEqual([1, 2, 3]);
    });
  });

  describe('game_resumed', () => {
    it('should update pause state when game resumes', () => {
      // First pause the game
      const pausedState = gameReducer(
        initialGameSliceState,
        gameActions.gamePaused({
          paused_by_player_id: 'player-123',
          paused_by_index: 0,
          paused_by_name: 'Alice',
          deadline_ms: Date.now() + 60000,
          duration_ms: 60000,
          pause_reason: 'Player disconnected',
          eligible_voters: [1, 2, 3],
          kick_votes: [],
        })
      );

      // Then resume
      const resumePayload = {
        resumed_by_player_id: 'player-123',
        resumed_by_index: 0,
        resumed_by_name: 'Alice',
        resume_reason: 'Player reconnected',
      };

      const action = gameActions.gameResumed(resumePayload);
      const newState = gameReducer(pausedState, action);

      expect(newState.pause.status).toBe('resumed');
    });
  });

  describe('kick_vote_update', () => {
    it('should update kick votes', () => {
      // Start with paused state
      const pausedState = gameReducer(
        initialGameSliceState,
        gameActions.gamePaused({
          paused_by_player_id: 'player-123',
          paused_by_index: 0,
          paused_by_name: 'Alice',
          deadline_ms: Date.now() + 60000,
          duration_ms: 60000,
          pause_reason: 'Player disconnected',
          eligible_voters: [1, 2, 3],
          kick_votes: [],
        })
      );

      const payload = {
        eligible_voters: [1, 2, 3],
        kick_votes: [1, 2],
      };

      const action = gameActions.kickVoteUpdate(payload);
      const newState = gameReducer(pausedState, action);

      expect((newState.pause as any).kickVotes).toEqual([1, 2]);
    });
  });

  describe('player_kicked', () => {
    it('should mark kicked player as eliminated', () => {
      const stateWithPlayers = {
        ...initialGameSliceState,
        players: [
          { index: 0, name: 'Alice', alive: true, coins: 5, cardCount: 2 },
          { index: 1, name: 'Bob', alive: true, coins: 3, cardCount: 2 },
        ],
      };

      const payload = {
        player_index: 1,
        reason: 'Kicked by vote',
      };

      const action = gameActions.playerKicked(payload);
      const newState = gameReducer(stateWithPlayers, action);

      expect(newState.players[1].alive).toBe(false);
      expect(newState.players[1].cardCount).toBe(0);
    });
  });

  describe('investigate_result', () => {
    it('should store investigation result in UI state', () => {
      const payload = {
        targetName: 'Bob',
        role: 'Thief',
      };

      const action = uiActions.investigateResult(payload);
      const newState = uiReducer(initialUIState, action);

      expect(newState.investigateResult).toEqual({
        targetName: 'Bob',
        role: 'Thief',
      });
    });
  });

  describe('chat_message', () => {
    it('should add message to chat array', () => {
      const payload = {
        id: 'msg-1',
        senderIndex: 0,
        senderName: 'Alice',
        text: 'Hello!',
        timestamp: Date.now(),
      };

      const action = uiActions.chatMessage(payload);
      const newState = uiReducer(initialUIState, action);

      expect(newState.chat).toHaveLength(1);
      expect(newState.chat[0].text).toBe('Hello!');
    });

    it('should limit chat to 100 messages', () => {
      let state = initialUIState;
      
      // Add 105 messages
      for (let i = 0; i < 105; i++) {
        state = uiReducer(state, uiActions.chatMessage({
          id: `msg-${i}`,
          senderIndex: 0,
          senderName: 'Alice',
          text: `Message ${i}`,
          timestamp: Date.now(),
        }));
      }

      expect(state.chat).toHaveLength(100);
      // Should keep the most recent messages
      expect(state.chat[99].text).toBe('Message 104');
    });
  });

  describe('rootReducer MESSAGE dispatch', () => {
    it('should handle game_paused envelope', () => {
      const envelope: WsEnvelope = {
        type: 'game_paused',
        request_id: 'req-1',
        payload: {
          paused_by_player_id: 'player-123',
          paused_by_index: 0,
          paused_by_name: 'Alice',
          deadline_ms: Date.now() + 60000,
          duration_ms: 60000,
          pause_reason: 'Player disconnected',
          eligible_voters: [1, 2, 3],
          kick_votes: [],
        },
      };

      const newState = rootReducer(initialSlicedState, { type: 'MESSAGE', envelope });

      expect(newState.game.pause.status).toBe('paused');
    });

    it('should handle investigate_result envelope', () => {
      const envelope: WsEnvelope = {
        type: 'investigate_result',
        payload: {
          target_name: 'Bob',
          role: 'Thief',
        },
      };

      const newState = rootReducer(initialSlicedState, { type: 'MESSAGE', envelope });

      expect(newState.ui.investigateResult).toEqual({
        targetName: 'Bob',
        role: 'Thief',
      });
    });

    it('should handle chat_message envelope', () => {
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

      const newState = rootReducer(initialSlicedState, { type: 'MESSAGE', envelope });

      expect(newState.ui.chat).toHaveLength(1);
    });
  });
});
