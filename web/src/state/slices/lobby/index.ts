/**
 * Lobby state slice
 *
 * Manages lobby list and current lobby state.
 */

import type { LobbyState, LobbySummary } from "@/state/types";

export interface LobbySliceState {
  lobbies: LobbySummary[];
  currentLobby: LobbyState | null;
}

export const initialLobbyState: LobbySliceState = {
  lobbies: [],
  currentLobby: null,
};

// Payload types
interface LobbyListPayload {
  lobbies?: Array<{
    id: string;
    leader_nick: string;
    leader_id?: string;
    player_count: number;
    player_nicks?: string[];
    player_ids?: string[];
    player_avatars?: string[];
    status: "open" | "in_game" | "closed";
  }>;
}

interface LobbyStatePayload {
  lobby_id: string;
  leader_nick: string;
  leader_id?: string;
  player_nicks?: string[];
  player_ids?: string[];
  player_avatars?: string[];
  player_count: number;
  status: "open" | "in_game" | "closed";
}

export type LobbyAction =
  | { type: "LOBBY_LIST_RESULT"; payload: LobbyListPayload | undefined }
  | { type: "LOBBY_JOINED" }
  | { type: "LOBBY_STATE"; payload: LobbyStatePayload; currentPlayerId: string | null }
  | { type: "LEAVE_LOBBY" }
  | { type: "RESET" };

export function lobbyReducer(
  state: LobbySliceState,
  action: LobbyAction
): LobbySliceState {
  switch (action.type) {
    case "LOBBY_LIST_RESULT": {
      const payload = action.payload;
      const lobbies = (payload?.lobbies || []).filter((lobby) => {
        if (typeof lobby.player_count === "number" && lobby.player_count <= 0) {
          return false;
        }
        const hasListedPlayers =
          (lobby.player_ids && lobby.player_ids.length > 0) ||
          (lobby.player_nicks && lobby.player_nicks.length > 0);
        return hasListedPlayers || lobby.status !== "open";
      });

      const mapped = lobbies.map((lobby) => ({
        id: lobby.id,
        leaderNick: lobby.leader_nick,
        leaderId: lobby.leader_id,
        playerCount: lobby.player_count,
        playerNicks: lobby.player_nicks || [],
        playerIds: lobby.player_ids || [],
        playerAvatars: lobby.player_avatars || [],
        status: lobby.status,
      }));

      // Check if current lobby still exists in list
      const hasCurrentLobby =
        state.currentLobby &&
        mapped.some((lobby) => lobby.id === state.currentLobby?.lobbyId);

      if (state.currentLobby && !hasCurrentLobby) {
        // Lobby closed or player removed
        return {
          lobbies: mapped,
          currentLobby: null,
        };
      }

      return {
        ...state,
        lobbies: mapped,
      };
    }

    case "LOBBY_JOINED":
      // Clear lobby list when joining
      return {
        ...state,
        lobbies: [],
      };

    case "LOBBY_STATE": {
      const payload = action.payload;
      const playerIds = payload.player_ids || [];

      // Verify player is still in lobby
      if (
        playerIds.length > 0 &&
        action.currentPlayerId &&
        !playerIds.includes(action.currentPlayerId)
      ) {
        // Player was removed from lobby
        return {
          ...state,
          currentLobby: null,
        };
      }

      return {
        ...state,
        currentLobby: {
          lobbyId: payload.lobby_id,
          leaderNick: payload.leader_nick,
          leaderId: payload.leader_id,
          playerNicks: payload.player_nicks || [],
          playerIds: payload.player_ids || [],
          playerAvatars: payload.player_avatars || [],
          playerCount: payload.player_count,
          status: payload.status,
        },
        lobbies: [],
      };
    }

    case "LEAVE_LOBBY":
      return {
        ...state,
        currentLobby: null,
      };

    case "RESET":
      return initialLobbyState;

    default:
      return state;
  }
}

// Action creators
export const lobbyActions = {
  listResult: (payload: LobbyListPayload | undefined): LobbyAction => ({
    type: "LOBBY_LIST_RESULT",
    payload,
  }),
  joined: (): LobbyAction => ({ type: "LOBBY_JOINED" }),
  state: (payload: LobbyStatePayload, currentPlayerId: string | null): LobbyAction => ({
    type: "LOBBY_STATE",
    payload,
    currentPlayerId,
  }),
  leave: (): LobbyAction => ({ type: "LEAVE_LOBBY" }),
  reset: (): LobbyAction => ({ type: "RESET" }),
} as const;
