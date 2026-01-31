/**
 * Lobby slice tests
 *
 * Tests for lobby state management including list, join, leave, and state updates.
 */

import { describe, it, expect } from "vitest";
import {
  lobbyReducer,
  lobbyActions,
  initialLobbyState,
  type LobbySliceState,
  type LobbyAction,
} from "@/state/slices/lobby";
import type { LobbyState } from "@/state/types";

describe("Lobby Slice", () => {
  describe("Initial State", () => {
    it("should have empty lobbies array", () => {
      expect(initialLobbyState.lobbies).toEqual([]);
    });

    it("should have null currentLobby", () => {
      expect(initialLobbyState.currentLobby).toBeNull();
    });
  });

  describe("LOBBY_LIST_RESULT action", () => {
    it("should populate lobbies from payload", () => {
      const payload = {
        lobbies: [
          {
            id: "lobby-1",
            leader_nick: "Leader1",
            leader_id: "leader-1",
            player_count: 3,
            player_nicks: ["Player1", "Player2", "Player3"],
            player_ids: ["p1", "p2", "p3"],
            player_avatars: ["avatar1", "avatar2", "avatar3"],
            status: "open" as const,
          },
        ],
      };
      const action = lobbyActions.listResult(payload);
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.lobbies).toHaveLength(1);
      expect(newState.lobbies[0].id).toBe("lobby-1");
      expect(newState.lobbies[0].leaderNick).toBe("Leader1");
      expect(newState.lobbies[0].leaderId).toBe("leader-1");
      expect(newState.lobbies[0].playerCount).toBe(3);
      expect(newState.lobbies[0].playerNicks).toEqual(["Player1", "Player2", "Player3"]);
      expect(newState.lobbies[0].playerIds).toEqual(["p1", "p2", "p3"]);
      expect(newState.lobbies[0].playerAvatars).toEqual(["avatar1", "avatar2", "avatar3"]);
      expect(newState.lobbies[0].status).toBe("open");
    });

    it("should map snake_case fields to camelCase", () => {
      const payload = {
        lobbies: [
          {
            id: "lobby-1",
            leader_nick: "Leader",
            leader_id: "l1",
            player_count: 2,
            player_nicks: ["P1", "P2"],
            player_ids: ["id1", "id2"],
            player_avatars: ["a1", "a2"],
            status: "in_game" as const,
          },
        ],
      };
      const action = lobbyActions.listResult(payload);
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.lobbies[0]).toMatchObject({
        id: "lobby-1",
        leaderNick: "Leader",
        leaderId: "l1",
        playerCount: 2,
        playerNicks: ["P1", "P2"],
        playerIds: ["id1", "id2"],
        playerAvatars: ["a1", "a2"],
        status: "in_game",
      });
    });

    it("should handle empty lobbies array", () => {
      const payload = { lobbies: [] };
      const action = lobbyActions.listResult(payload);
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.lobbies).toEqual([]);
    });

    it("should handle undefined payload", () => {
      const action = lobbyActions.listResult(undefined);
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.lobbies).toEqual([]);
    });

    it("should filter out lobbies with zero player_count", () => {
      const payload = {
        lobbies: [
          {
            id: "lobby-1",
            leader_nick: "Leader1",
            player_count: 3,
            status: "open" as const,
            player_nicks: ["P1", "P2", "P3"],
          },
          {
            id: "lobby-2",
            leader_nick: "Leader2",
            player_count: 0,
            status: "open" as const,
          },
          {
            id: "lobby-3",
            leader_nick: "Leader3",
            player_count: 1,
            status: "open" as const,
            player_nicks: ["P1"],
          },
        ],
      };
      const action = lobbyActions.listResult(payload);
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.lobbies).toHaveLength(2);
      expect(newState.lobbies.map((l) => l.id)).toEqual(["lobby-1", "lobby-3"]);
    });

    it("should filter out lobbies with negative player_count", () => {
      const payload = {
        lobbies: [
          {
            id: "lobby-1",
            leader_nick: "Leader1",
            player_count: -1,
            status: "open" as const,
          },
          {
            id: "lobby-2",
            leader_nick: "Leader2",
            player_count: 2,
            status: "open" as const,
            player_nicks: ["P1", "P2"],
          },
        ],
      };
      const action = lobbyActions.listResult(payload);
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.lobbies).toHaveLength(1);
      expect(newState.lobbies[0].id).toBe("lobby-2");
    });

    it("should filter out open lobbies with no listed players", () => {
      const payload = {
        lobbies: [
          {
            id: "lobby-1",
            leader_nick: "Leader1",
            player_count: 3,
            status: "open" as const,
            // No player_nicks or player_ids
          },
          {
            id: "lobby-2",
            leader_nick: "Leader2",
            player_count: 2,
            status: "open" as const,
            player_nicks: ["P1", "P2"],
          },
        ],
      };
      const action = lobbyActions.listResult(payload);
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.lobbies).toHaveLength(1);
      expect(newState.lobbies[0].id).toBe("lobby-2");
    });

    it("should keep in_game lobbies even without listed players (if player_count > 0)", () => {
      const payload = {
        lobbies: [
          {
            id: "lobby-1",
            leader_nick: "Leader1",
            player_count: 1,
            status: "in_game" as const,
            // No player_nicks or player_ids - but still included because status !== "open"
          },
          {
            id: "lobby-2",
            leader_nick: "Leader2",
            player_count: 2,
            status: "in_game" as const,
            player_nicks: ["P1", "P2"],
          },
        ],
      };
      const action = lobbyActions.listResult(payload);
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.lobbies).toHaveLength(2);
    });

    it("should keep closed lobbies even without listed players (if player_count > 0)", () => {
      const payload = {
        lobbies: [
          {
            id: "lobby-1",
            leader_nick: "Leader1",
            player_count: 1,
            status: "closed" as const,
            // No player_nicks or player_ids - but still included because status !== "open"
          },
          {
            id: "lobby-2",
            leader_nick: "Leader2",
            player_count: 2,
            status: "closed" as const,
            player_nicks: ["P1", "P2"],
          },
        ],
      };
      const action = lobbyActions.listResult(payload);
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.lobbies).toHaveLength(2);
    });

    it("should keep open lobby when player_ids is populated", () => {
      const payload = {
        lobbies: [
          {
            id: "lobby-1",
            leader_nick: "Leader1",
            player_count: 2,
            status: "open" as const,
            player_ids: ["id1", "id2"],
            // No player_nicks
          },
        ],
      };
      const action = lobbyActions.listResult(payload);
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.lobbies).toHaveLength(1);
      expect(newState.lobbies[0].playerIds).toEqual(["id1", "id2"]);
    });

    it("should clear currentLobby when it disappears from list", () => {
      const state: LobbySliceState = {
        ...initialLobbyState,
        currentLobby: {
          lobbyId: "lobby-1",
          leaderNick: "Leader1",
          playerNicks: ["P1", "P2"],
          playerIds: ["p1", "p2"],
          playerAvatars: [],
          playerCount: 2,
          status: "open",
        },
      };
      const payload = {
        lobbies: [
          {
            id: "lobby-2",
            leader_nick: "Leader2",
            player_count: 3,
            status: "open" as const,
            player_nicks: ["P3", "P4", "P5"],
          },
        ],
      };
      const action = lobbyActions.listResult(payload);
      const newState = lobbyReducer(state, action);

      expect(newState.currentLobby).toBeNull();
      expect(newState.lobbies).toHaveLength(1);
    });

    it("should keep currentLobby when it exists in new list", () => {
      const currentLobby: LobbyState = {
        lobbyId: "lobby-1",
        leaderNick: "Leader1",
        playerNicks: ["P1", "P2"],
        playerIds: ["p1", "p2"],
        playerAvatars: [],
        playerCount: 2,
        status: "open",
      };
      const state: LobbySliceState = {
        ...initialLobbyState,
        currentLobby,
      };
      const payload = {
        lobbies: [
          {
            id: "lobby-1",
            leader_nick: "Leader1",
            player_count: 2,
            status: "open" as const,
            player_nicks: ["P1", "P2"],
          },
          {
            id: "lobby-2",
            leader_nick: "Leader2",
            player_count: 3,
            status: "open" as const,
            player_nicks: ["P3", "P4", "P5"],
          },
        ],
      };
      const action = lobbyActions.listResult(payload);
      const newState = lobbyReducer(state, action);

      expect(newState.currentLobby).toEqual(currentLobby);
    });

    it("should handle missing optional fields with defaults", () => {
      const payload = {
        lobbies: [
          {
            id: "lobby-1",
            leader_nick: "Leader1",
            player_count: 2,
            status: "in_game" as const, // Use in_game so lobby passes filter without player lists
            // Missing: leader_id, player_nicks, player_ids, player_avatars
          },
        ],
      };
      const action = lobbyActions.listResult(payload);
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.lobbies[0].leaderId).toBeUndefined();
      expect(newState.lobbies[0].playerNicks).toEqual([]);
      expect(newState.lobbies[0].playerIds).toEqual([]);
      expect(newState.lobbies[0].playerAvatars).toEqual([]);
    });

    it.each<
      [
        string,
        {
          id: string;
          leader_nick: string;
          player_count: number;
          status: "open" | "in_game" | "closed";
          player_nicks?: string[];
          player_ids?: string[];
        },
        boolean,
      ]
    >([
      [
        "open lobby with players",
        {
          id: "l1",
          leader_nick: "L1",
          player_count: 2,
          status: "open",
          player_nicks: ["P1", "P2"],
        },
        true,
      ],
      [
        "open lobby without players",
        { id: "l2", leader_nick: "L2", player_count: 2, status: "open" },
        false,
      ],
      [
        "in_game lobby without players",
        { id: "l3", leader_nick: "L3", player_count: 0, status: "in_game" },
        false, // Filtered out because player_count <= 0 is checked first
      ],
      [
        "closed lobby with players",
        {
          id: "l4",
          leader_nick: "L4",
          player_count: 1,
          status: "closed",
          player_ids: ["p1"],
        },
        true,
      ],
      [
        "open lobby with zero count",
        { id: "l5", leader_nick: "L5", player_count: 0, status: "open" },
        false,
      ],
    ])("should handle %s: %s", (_desc: string, lobbyData: { id: string; leader_nick: string; player_count: number; status: "open" | "in_game" | "closed"; player_nicks?: string[]; player_ids?: string[] }, shouldInclude: boolean) => {
      const payload = { lobbies: [lobbyData] };
      const action = lobbyActions.listResult(payload);
      const newState = lobbyReducer(initialLobbyState, action);

      if (shouldInclude) {
        expect(newState.lobbies).toHaveLength(1);
        expect(newState.lobbies[0].id).toBe(lobbyData.id);
      } else {
        expect(newState.lobbies).toHaveLength(0);
      }
    });
  });

  describe("LOBBY_JOINED action", () => {
    it("should clear lobbies list", () => {
      const state: LobbySliceState = {
        ...initialLobbyState,
        lobbies: [
          {
            id: "lobby-1",
            leaderNick: "Leader1",
            playerNicks: ["P1"],
            playerIds: ["p1"],
            playerAvatars: [],
            playerCount: 1,
            status: "open",
          },
        ],
      };
      const action = lobbyActions.joined();
      const newState = lobbyReducer(state, action);

      expect(newState.lobbies).toEqual([]);
    });

    it("should not affect currentLobby", () => {
      const state: LobbySliceState = {
        lobbies: [],
        currentLobby: {
          lobbyId: "lobby-1",
          leaderNick: "Leader1",
          playerNicks: ["P1"],
          playerIds: ["p1"],
          playerAvatars: [],
          playerCount: 1,
          status: "open",
        },
      };
      const action = lobbyActions.joined();
      const newState = lobbyReducer(state, action);

      expect(newState.currentLobby).not.toBeNull();
    });

    it("should work from initial state", () => {
      const action = lobbyActions.joined();
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.lobbies).toEqual([]);
      expect(newState.currentLobby).toBeNull();
    });
  });

  describe("LOBBY_STATE action", () => {
    it("should set currentLobby from payload", () => {
      const payload = {
        lobby_id: "lobby-1",
        leader_nick: "Leader1",
        leader_id: "leader-1",
        player_nicks: ["P1", "P2"],
        player_ids: ["p1", "p2"],
        player_avatars: ["a1", "a2"],
        player_count: 2,
        status: "open" as const,
      };
      const action = lobbyActions.state(payload, "p1");
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.currentLobby).not.toBeNull();
      expect(newState.currentLobby?.lobbyId).toBe("lobby-1");
      expect(newState.currentLobby?.leaderNick).toBe("Leader1");
      expect(newState.currentLobby?.leaderId).toBe("leader-1");
      expect(newState.currentLobby?.playerNicks).toEqual(["P1", "P2"]);
      expect(newState.currentLobby?.playerIds).toEqual(["p1", "p2"]);
      expect(newState.currentLobby?.playerAvatars).toEqual(["a1", "a2"]);
      expect(newState.currentLobby?.playerCount).toBe(2);
      expect(newState.currentLobby?.status).toBe("open");
    });

    it("should map fields correctly", () => {
      const payload = {
        lobby_id: "lobby-abc",
        leader_nick: "TestLeader",
        leader_id: "leader-id",
        player_nicks: ["PlayerA", "PlayerB"],
        player_ids: ["id-a", "id-b"],
        player_avatars: ["av-a", "av-b"],
        player_count: 2,
        status: "in_game" as const,
      };
      const action = lobbyActions.state(payload, "id-a");
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.currentLobby).toMatchObject({
        lobbyId: "lobby-abc",
        leaderNick: "TestLeader",
        leaderId: "leader-id",
        playerNicks: ["PlayerA", "PlayerB"],
        playerIds: ["id-a", "id-b"],
        playerAvatars: ["av-a", "av-b"],
        playerCount: 2,
        status: "in_game",
      });
    });

    it("should clear lobbies list", () => {
      const state: LobbySliceState = {
        lobbies: [
          {
            id: "lobby-1",
            leaderNick: "Leader1",
            playerNicks: ["P1"],
            playerIds: ["p1"],
            playerAvatars: [],
            playerCount: 1,
            status: "open",
          },
        ],
        currentLobby: null,
      };
      const payload = {
        lobby_id: "lobby-2",
        leader_nick: "Leader2",
        player_count: 1,
        status: "open" as const,
        player_nicks: ["P2"],
        player_ids: ["p2"],
      };
      const action = lobbyActions.state(payload, "p2");
      const newState = lobbyReducer(state, action);

      expect(newState.lobbies).toEqual([]);
    });

    it("should clear currentLobby when player not in participant list", () => {
      const state: LobbySliceState = {
        lobbies: [],
        currentLobby: {
          lobbyId: "lobby-1",
          leaderNick: "Leader1",
          playerNicks: ["P1", "P2"],
          playerIds: ["p1", "p2"],
          playerAvatars: [],
          playerCount: 2,
          status: "open",
        },
      };
      const payload = {
        lobby_id: "lobby-1",
        leader_nick: "Leader1",
        player_count: 1,
        status: "open" as const,
        player_nicks: ["P2"],
        player_ids: ["p2"], // p1 is not here anymore
      };
      const action = lobbyActions.state(payload, "p1");
      const newState = lobbyReducer(state, action);

      expect(newState.currentLobby).toBeNull();
    });

    it("should keep currentLobby when player is in participant list", () => {
      const payload = {
        lobby_id: "lobby-1",
        leader_nick: "Leader1",
        player_count: 2,
        status: "open" as const,
        player_nicks: ["P1", "P2"],
        player_ids: ["p1", "p2"],
      };
      const action = lobbyActions.state(payload, "p1");
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.currentLobby).not.toBeNull();
      expect(newState.currentLobby?.lobbyId).toBe("lobby-1");
    });

    it("should not check player removal when player_ids is empty", () => {
      const state: LobbySliceState = {
        lobbies: [],
        currentLobby: {
          lobbyId: "lobby-1",
          leaderNick: "Leader1",
          playerNicks: ["P1"],
          playerIds: ["p1"],
          playerAvatars: [],
          playerCount: 1,
          status: "open",
        },
      };
      const payload = {
        lobby_id: "lobby-1",
        leader_nick: "Leader1",
        player_count: 1,
        status: "open" as const,
        // player_ids is undefined/empty - skip check
      };
      const action = lobbyActions.state(payload, "p1");
      const newState = lobbyReducer(state, action);

      expect(newState.currentLobby).not.toBeNull();
    });

    it("should not check player removal when currentPlayerId is null", () => {
      const state: LobbySliceState = {
        lobbies: [],
        currentLobby: {
          lobbyId: "lobby-1",
          leaderNick: "Leader1",
          playerNicks: ["P1", "P2"],
          playerIds: ["p1", "p2"],
          playerAvatars: [],
          playerCount: 2,
          status: "open",
        },
      };
      const payload = {
        lobby_id: "lobby-1",
        leader_nick: "Leader1",
        player_count: 1,
        status: "open" as const,
        player_ids: ["p2"], // p1 not here
      };
      const action = lobbyActions.state(payload, null); // No current player
      const newState = lobbyReducer(state, action);

      // Should set new state even though p1 isn't in list
      expect(newState.currentLobby).not.toBeNull();
      expect(newState.currentLobby?.playerIds).toEqual(["p2"]);
    });

    it("should handle missing optional fields with defaults", () => {
      const payload = {
        lobby_id: "lobby-1",
        leader_nick: "Leader1",
        player_count: 1,
        status: "open" as const,
        // Missing: leader_id, player_nicks, player_ids, player_avatars
      };
      const action = lobbyActions.state(payload, null);
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.currentLobby?.leaderId).toBeUndefined();
      expect(newState.currentLobby?.playerNicks).toEqual([]);
      expect(newState.currentLobby?.playerIds).toEqual([]);
      expect(newState.currentLobby?.playerAvatars).toEqual([]);
    });

    it.each<
      [
        string,
        string | null,
        string[] | undefined,
        boolean, // shouldClearLobby
      ]
    >([
      ["player in list", "p1", ["p1", "p2"], false],
      ["player not in list", "p1", ["p2", "p3"], true],
      ["null currentPlayerId", null, ["p1", "p2"], false],
      ["empty player_ids", "p1", [], false], // Empty array skips check
      ["undefined player_ids", "p1", undefined, false], // Undefined skips check
    ])("should handle %s", (_desc: string, currentPlayerId: string | null, playerIds: string[] | undefined, shouldClearLobby: boolean) => {
      const state: LobbySliceState = {
        lobbies: [],
        currentLobby: {
          lobbyId: "lobby-1",
          leaderNick: "Leader1",
          playerNicks: ["P1", "P2"],
          playerIds: ["p1", "p2"],
          playerAvatars: [],
          playerCount: 2,
          status: "open",
        },
      };
      const payload = {
        lobby_id: "lobby-1",
        leader_nick: "Leader1",
        player_count: playerIds?.length ?? 0,
        status: "open" as const,
        player_ids: playerIds,
      };
      const action = lobbyActions.state(payload, currentPlayerId);
      const newState = lobbyReducer(state, action);

      if (shouldClearLobby) {
        expect(newState.currentLobby).toBeNull();
      } else {
        expect(newState.currentLobby).not.toBeNull();
      }
    });
  });

  describe("LEAVE_LOBBY action", () => {
    it("should clear currentLobby", () => {
      const state: LobbySliceState = {
        lobbies: [],
        currentLobby: {
          lobbyId: "lobby-1",
          leaderNick: "Leader1",
          playerNicks: ["P1"],
          playerIds: ["p1"],
          playerAvatars: [],
          playerCount: 1,
          status: "open",
        },
      };
      const action = lobbyActions.leave();
      const newState = lobbyReducer(state, action);

      expect(newState.currentLobby).toBeNull();
    });

    it("should not affect lobbies list", () => {
      const state: LobbySliceState = {
        lobbies: [
          {
            id: "lobby-1",
            leaderNick: "Leader1",
            playerNicks: ["P1"],
            playerIds: ["p1"],
            playerAvatars: [],
            playerCount: 1,
            status: "open",
          },
        ],
        currentLobby: {
          lobbyId: "lobby-2",
          leaderNick: "Leader2",
          playerNicks: ["P2"],
          playerIds: ["p2"],
          playerAvatars: [],
          playerCount: 1,
          status: "open",
        },
      };
      const action = lobbyActions.leave();
      const newState = lobbyReducer(state, action);

      expect(newState.lobbies).toHaveLength(1);
      expect(newState.lobbies[0].id).toBe("lobby-1");
    });

    it("should work when currentLobby is already null", () => {
      const action = lobbyActions.leave();
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.currentLobby).toBeNull();
      expect(newState.lobbies).toEqual([]);
    });
  });

  describe("RESET action", () => {
    it("should reset to initial state", () => {
      const state: LobbySliceState = {
        lobbies: [
          {
            id: "lobby-1",
            leaderNick: "Leader1",
            playerNicks: ["P1"],
            playerIds: ["p1"],
            playerAvatars: [],
            playerCount: 1,
            status: "open",
          },
        ],
        currentLobby: {
          lobbyId: "lobby-1",
          leaderNick: "Leader1",
          playerNicks: ["P1"],
          playerIds: ["p1"],
          playerAvatars: [],
          playerCount: 1,
          status: "open",
        },
      };
      const action = lobbyActions.reset();
      const newState = lobbyReducer(state, action);

      expect(newState.lobbies).toEqual([]);
      expect(newState.currentLobby).toBeNull();
    });

    it("should reset from initial state", () => {
      const action = lobbyActions.reset();
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState).toEqual(initialLobbyState);
    });
  });

  describe("Edge Cases", () => {
    it("should handle empty lobby list with existing currentLobby", () => {
      const state: LobbySliceState = {
        lobbies: [],
        currentLobby: {
          lobbyId: "lobby-1",
          leaderNick: "Leader1",
          playerNicks: ["P1"],
          playerIds: ["p1"],
          playerAvatars: [],
          playerCount: 1,
          status: "open",
        },
      };
      const payload = { lobbies: [] };
      const action = lobbyActions.listResult(payload);
      const newState = lobbyReducer(state, action);

      // Lobby disappeared from list, so currentLobby should be cleared
      expect(newState.currentLobby).toBeNull();
      expect(newState.lobbies).toEqual([]);
    });

    it("should handle lobbies with only player_ids (no player_nicks)", () => {
      const payload = {
        lobbies: [
          {
            id: "lobby-1",
            leader_nick: "Leader1",
            player_count: 2,
            status: "open" as const,
            player_ids: ["id1", "id2"],
          },
        ],
      };
      const action = lobbyActions.listResult(payload);
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.lobbies).toHaveLength(1);
      expect(newState.lobbies[0].playerIds).toEqual(["id1", "id2"]);
      expect(newState.lobbies[0].playerNicks).toEqual([]);
    });

    it("should handle lobbies with only player_nicks (no player_ids)", () => {
      const payload = {
        lobbies: [
          {
            id: "lobby-1",
            leader_nick: "Leader1",
            player_count: 2,
            status: "open" as const,
            player_nicks: ["Nick1", "Nick2"],
          },
        ],
      };
      const action = lobbyActions.listResult(payload);
      const newState = lobbyReducer(initialLobbyState, action);

      expect(newState.lobbies).toHaveLength(1);
      expect(newState.lobbies[0].playerNicks).toEqual(["Nick1", "Nick2"]);
      expect(newState.lobbies[0].playerIds).toEqual([]);
    });

    it("should handle lobby transition: in list → joined → state → left", () => {
      let state = initialLobbyState;

      // Lobby list received
      state = lobbyReducer(
        state,
        lobbyActions.listResult({
          lobbies: [
            {
              id: "lobby-1",
              leader_nick: "Leader1",
              player_count: 1,
              status: "open" as const,
              player_nicks: ["Leader1"],
            },
          ],
        })
      );
      expect(state.lobbies).toHaveLength(1);
      expect(state.currentLobby).toBeNull();

      // Joined lobby - list cleared
      state = lobbyReducer(state, lobbyActions.joined());
      expect(state.lobbies).toEqual([]);
      expect(state.currentLobby).toBeNull();

      // Lobby state received
      state = lobbyReducer(
        state,
        lobbyActions.state(
          {
            lobby_id: "lobby-1",
            leader_nick: "Leader1",
            player_count: 2,
            status: "open" as const,
            player_nicks: ["Leader1", "Player2"],
            player_ids: ["l1", "p2"],
          },
          "p2"
        )
      );
      expect(state.currentLobby?.lobbyId).toBe("lobby-1");
      expect(state.currentLobby?.playerCount).toBe(2);
      expect(state.lobbies).toEqual([]);

      // Leave lobby
      state = lobbyReducer(state, lobbyActions.leave());
      expect(state.currentLobby).toBeNull();
    });

    it("should handle player being removed from lobby", () => {
      let state: LobbySliceState = {
        lobbies: [],
        currentLobby: {
          lobbyId: "lobby-1",
          leaderNick: "Leader1",
          playerNicks: ["Leader1", "Player2"],
          playerIds: ["l1", "p2"],
          playerAvatars: [],
          playerCount: 2,
          status: "open",
        },
      };

      // State update shows player was removed
      state = lobbyReducer(
        state,
        lobbyActions.state(
          {
            lobby_id: "lobby-1",
            leader_nick: "Leader1",
            player_count: 1,
            status: "open" as const,
            player_nicks: ["Leader1"],
            player_ids: ["l1"], // p2 not here anymore
          },
          "p2"
        )
      );

      expect(state.currentLobby).toBeNull();
    });

    it("should handle lobby list update while in lobby", () => {
      const state: LobbySliceState = {
        lobbies: [],
        currentLobby: {
          lobbyId: "lobby-1",
          leaderNick: "Leader1",
          playerNicks: ["P1", "P2"],
          playerIds: ["p1", "p2"],
          playerAvatars: [],
          playerCount: 2,
          status: "open",
        },
      };

      // New lobby list that includes current lobby
      const payload = {
        lobbies: [
          {
            id: "lobby-1",
            leader_nick: "Leader1",
            player_count: 2,
            status: "open" as const,
            player_nicks: ["P1", "P2"],
          },
          {
            id: "lobby-2",
            leader_nick: "Leader2",
            player_count: 3,
            status: "open" as const,
            player_nicks: ["P3", "P4", "P5"],
          },
        ],
      };
      const action = lobbyActions.listResult(payload);
      const newState = lobbyReducer(state, action);

      // Current lobby preserved, lobbies list updated
      expect(newState.currentLobby?.lobbyId).toBe("lobby-1");
      expect(newState.lobbies).toHaveLength(2);
    });

    it("should maintain state reference integrity", () => {
      const state: LobbySliceState = {
        lobbies: [],
        currentLobby: {
          lobbyId: "lobby-1",
          leaderNick: "Leader1",
          playerNicks: ["P1"],
          playerIds: ["p1"],
          playerAvatars: [],
          playerCount: 1,
          status: "open",
        },
      };

      const action = lobbyActions.leave();
      const newState = lobbyReducer(state, action);

      // New state should be a different object
      expect(newState).not.toBe(state);
      // But lobbies array should still be the same reference if not changed
      expect(newState.lobbies).toBe(state.lobbies);
    });

    it("should not mutate original state", () => {
      const originalState: LobbySliceState = {
        lobbies: [
          {
            id: "lobby-1",
            leaderNick: "Leader1",
            playerNicks: ["P1"],
            playerIds: ["p1"],
            playerAvatars: [],
            playerCount: 1,
            status: "open",
          },
        ],
        currentLobby: null,
      };

      const stateCopy = JSON.parse(JSON.stringify(originalState));
      const action = lobbyActions.joined();
      lobbyReducer(originalState, action);

      // Original state should remain unchanged
      expect(originalState).toEqual(stateCopy);
    });

    it("should handle rapid state changes", () => {
      let state = initialLobbyState;

      // List result
      state = lobbyReducer(
        state,
        lobbyActions.listResult({
          lobbies: [
            { id: "l1", leader_nick: "L1", player_count: 1, status: "open" as const, player_nicks: ["L1"] },
          ],
        })
      );
      expect(state.lobbies).toHaveLength(1);

      // Join
      state = lobbyReducer(state, lobbyActions.joined());
      expect(state.lobbies).toEqual([]);

      // State
      state = lobbyReducer(
        state,
        lobbyActions.state(
          {
            lobby_id: "l1",
            leader_nick: "L1",
            player_count: 1,
            status: "open" as const,
          },
          null
        )
      );
      expect(state.currentLobby).not.toBeNull();

      // Leave
      state = lobbyReducer(state, lobbyActions.leave());
      expect(state.currentLobby).toBeNull();

      // Reset
      state = lobbyReducer(state, lobbyActions.reset());
      expect(state).toEqual(initialLobbyState);
    });
  });

  describe("Unknown actions", () => {
    it("should return current state for unknown action type", () => {
      const state: LobbySliceState = {
        lobbies: [
          {
            id: "lobby-1",
            leaderNick: "Leader1",
            playerNicks: ["P1"],
            playerIds: ["p1"],
            playerAvatars: [],
            playerCount: 1,
            status: "open",
          },
        ],
        currentLobby: null,
      };
      const action = { type: "UNKNOWN_ACTION" } as unknown as LobbyAction;
      const newState = lobbyReducer(state, action);

      expect(newState).toEqual(state);
    });

    it("should handle undefined action gracefully", () => {
      const state: LobbySliceState = {
        lobbies: [],
        currentLobby: {
          lobbyId: "lobby-1",
          leaderNick: "Leader1",
          playerNicks: ["P1"],
          playerIds: ["p1"],
          playerAvatars: [],
          playerCount: 1,
          status: "open",
        },
      };
      const action = { type: "RANDOM_TYPE" } as unknown as LobbyAction;
      const newState = lobbyReducer(state, action);

      expect(newState).toEqual(state);
    });
  });

  describe("Action Creators", () => {
    it("should create LOBBY_LIST_RESULT action", () => {
      const payload = {
        lobbies: [
          { id: "lobby-1", leader_nick: "Leader1", player_count: 1, status: "open" as const },
        ],
      };
      const action = lobbyActions.listResult(payload);

      expect(action).toEqual({
        type: "LOBBY_LIST_RESULT",
        payload,
      });
    });

    it("should create LOBBY_LIST_RESULT with undefined payload", () => {
      const action = lobbyActions.listResult(undefined);

      expect(action).toEqual({
        type: "LOBBY_LIST_RESULT",
        payload: undefined,
      });
    });

    it("should create LOBBY_JOINED action", () => {
      const action = lobbyActions.joined();

      expect(action).toEqual({ type: "LOBBY_JOINED" });
    });

    it("should create LOBBY_STATE action", () => {
      const payload = {
        lobby_id: "lobby-1",
        leader_nick: "Leader1",
        player_count: 2,
        status: "open" as const,
        player_nicks: ["P1", "P2"],
        player_ids: ["p1", "p2"],
      };
      const action = lobbyActions.state(payload, "p1");

      expect(action).toEqual({
        type: "LOBBY_STATE",
        payload,
        currentPlayerId: "p1",
      });
    });

    it("should create LOBBY_STATE with null currentPlayerId", () => {
      const payload = {
        lobby_id: "lobby-1",
        leader_nick: "Leader1",
        player_count: 1,
        status: "open" as const,
      };
      const action = lobbyActions.state(payload, null);

      expect(action).toEqual({
        type: "LOBBY_STATE",
        payload,
        currentPlayerId: null,
      });
    });

    it("should create LEAVE_LOBBY action", () => {
      const action = lobbyActions.leave();

      expect(action).toEqual({ type: "LEAVE_LOBBY" });
    });

    it("should create RESET action", () => {
      const action = lobbyActions.reset();

      expect(action).toEqual({ type: "RESET" });
    });
  });
});
