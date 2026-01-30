export {};

declare global {
  interface Window {
    __ELMAKINA_DEV__?: {
      dispatch: (action: { type: string; [key: string]: unknown }) => void;
      getState: () => unknown;
      sendEnvelope: (envelope: import('../network/socket').WsEnvelope) => void;
      pauseGame: (payload: {
        paused_by_player_id?: string;
        paused_by_index?: number;
        paused_by_name?: string;
        deadline_ms?: number;
        duration_ms?: number;
        pause_reason?: string;
        eligible_voters?: number[];
        kick_votes?: number[];
      }) => void;
      updateKickVotes: (payload: { eligible_voters?: number[]; kick_votes?: number[] }) => void;
      resumeGame: () => void;
      connectionLog: () => unknown[];
    };
  }
}
