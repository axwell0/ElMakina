/**
 * Utility function tests
 *
 * Tests for pure utility functions across lib modules.
 * These tests validate class name merging, layout calculations,
 * card utilities, audio utilities, and replay utilities.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cn } from "@/lib/utils";
import { getPlayerPositions, BOARD_LAYOUT } from "@/lib/layout";
import { cardImageForRole } from "@/lib/cards";
import { playSfx } from "@/lib/audio";
import { getReplayUrl, fetchReplay } from "@/lib/replay";
import { roleForAction, actionLabel, mainActionImage, counterActionImage } from "@/lib/actions";

describe("cn() - Class Name Merging", () => {
  it("should merge simple class names", () => {
    expect(cn("class1", "class2")).toBe("class1 class2");
  });

  it("should handle single class name", () => {
    expect(cn("single")).toBe("single");
  });

  it("should filter out falsy values", () => {
    expect(cn("class1", false && "class2", "class3", null, undefined, "", "class4")).toBe("class1 class3 class4");
  });

  it("should handle conditional classes", () => {
    const isActive = true;
    const isDisabled = false;
    expect(cn("base", isActive && "active", isDisabled && "disabled")).toBe("base active");
  });

  it("should merge Tailwind classes correctly", () => {
    expect(cn("px-2 py-1", "px-4")).toBe("py-1 px-4");
  });

  it("should handle empty input", () => {
    expect(cn()).toBe("");
  });

  it("should handle all falsy inputs", () => {
    expect(cn(false, null, undefined, "")).toBe("");
  });

  it("should handle object syntax", () => {
    expect(cn({ active: true, disabled: false })).toBe("active");
  });

  it("should handle array syntax", () => {
    expect(cn(["class1", "class2"])).toBe("class1 class2");
  });

  it("should handle nested arrays", () => {
    expect(cn(["class1", ["class2", "class3"]])).toBe("class1 class2 class3");
  });

  it("should deduplicate conflicting Tailwind classes", () => {
    expect(cn("text-red-500", "text-blue-500")).toBe("text-blue-500");
  });

  it("should handle complex real-world example", () => {
    const isPrimary = true;
    const isLarge = false;
    const result = cn(
      "inline-flex items-center justify-center rounded-md text-sm font-medium",
      "bg-slate-900 text-white hover:bg-slate-700",
      isPrimary && "bg-blue-600 hover:bg-blue-700",
      isLarge ? "h-12 px-6" : "h-10 px-4",
      "disabled:opacity-50 disabled:pointer-events-none"
    );
    expect(result).toContain("inline-flex");
    expect(result).toContain("bg-blue-600");
    expect(result).toContain("h-10");
    expect(result).not.toContain("bg-slate-900"); // Should be overridden by bg-blue-600
  });
});

describe("BOARD_LAYOUT Constants", () => {
  it("should have correct center coordinates", () => {
    expect(BOARD_LAYOUT.X_CENTER).toBe(50);
    expect(BOARD_LAYOUT.Y_CENTER).toBe(50);
  });

  it("should have correct radius values", () => {
    expect(BOARD_LAYOUT.RADIUS_X).toBe(44);
    expect(BOARD_LAYOUT.RADIUS_Y).toBe(42);
  });

  it("should have correct card offset", () => {
    expect(BOARD_LAYOUT.CARD_OFFSET).toBe(12);
  });
});

describe("getPlayerPositions() - Layout Calculations", () => {
  it("should return empty array for zero players", () => {
    expect(getPlayerPositions(0)).toEqual([]);
  });

  it("should return empty array for negative player count", () => {
    expect(getPlayerPositions(-1)).toEqual([]);
  });

  it("should calculate correct position for single player", () => {
    const positions = getPlayerPositions(1);
    expect(positions).toHaveLength(1);
    expect(positions[0]).toMatchObject({
      x: expect.any(Number),
      y: expect.any(Number),
      cardX: expect.any(Number),
      cardY: expect.any(Number),
      angleDeg: expect.any(Number),
    });
  });

  it("should calculate positions for 2 players", () => {
    const positions = getPlayerPositions(2);
    expect(positions).toHaveLength(2);
    
    // First player should be at top
    expect(positions[0].y).toBeLessThan(BOARD_LAYOUT.Y_CENTER);
    
    // Second player should be at bottom
    expect(positions[1].y).toBeGreaterThan(BOARD_LAYOUT.Y_CENTER);
  });

  it("should calculate positions for 4 players", () => {
    const positions = getPlayerPositions(4);
    expect(positions).toHaveLength(4);
    
    // Check all positions have valid coordinates
    positions.forEach(pos => {
      expect(pos.x).toBeGreaterThanOrEqual(0);
      expect(pos.x).toBeLessThanOrEqual(100);
      expect(pos.y).toBeGreaterThanOrEqual(0);
      expect(pos.y).toBeLessThanOrEqual(100);
    });
  });

  it("should have card positions closer to center than player positions", () => {
    const positions = getPlayerPositions(3);
    
    positions.forEach(pos => {
      const playerDistToCenter = Math.sqrt(
        Math.pow(pos.x - BOARD_LAYOUT.X_CENTER, 2) +
        Math.pow(pos.y - BOARD_LAYOUT.Y_CENTER, 2)
      );
      const cardDistToCenter = Math.sqrt(
        Math.pow(pos.cardX - BOARD_LAYOUT.X_CENTER, 2) +
        Math.pow(pos.cardY - BOARD_LAYOUT.Y_CENTER, 2)
      );
      
      expect(cardDistToCenter).toBeLessThan(playerDistToCenter);
    });
  });

  it("should distribute players evenly around the ellipse", () => {
    const positions = getPlayerPositions(4);
    
    // Calculate angles between consecutive players
    const angles = positions.map(p => Math.atan2(
      p.y - BOARD_LAYOUT.Y_CENTER,
      p.x - BOARD_LAYOUT.X_CENTER
    ));
    
    // Check that angles are roughly evenly spaced (within tolerance)
    const angleDiffs = [];
    for (let i = 0; i < angles.length; i++) {
      const next = (i + 1) % angles.length;
      let diff = angles[next] - angles[i];
      if (diff < 0) diff += 2 * Math.PI;
      angleDiffs.push(diff);
    }
    
    const expectedDiff = (2 * Math.PI) / 4;
    angleDiffs.forEach(diff => {
      expect(Math.abs(diff - expectedDiff)).toBeLessThan(0.01);
    });
  });

  it("should return positions with 4 decimal precision", () => {
    const positions = getPlayerPositions(3);
    
    positions.forEach(pos => {
      const xDecimals = (pos.x.toString().split('.')[1] || '').length;
      const yDecimals = (pos.y.toString().split('.')[1] || '').length;
      expect(xDecimals).toBeLessThanOrEqual(4);
      expect(yDecimals).toBeLessThanOrEqual(4);
    });
  });

  it("should handle maximum reasonable player count", () => {
    const positions = getPlayerPositions(8);
    expect(positions).toHaveLength(8);
  });
});

describe("cardImageForRole() - Card Utilities", () => {
  it.each<[string, string]>([
    ["Businesswoman", "/cards/business.png"],
    ["TaxCollector", "/cards/tax.png"],
    ["Policewoman", "/cards/police.png"],
    ["Colonel", "/cards/colonel.png"],
    ["Terrorist", "/cards/terrorist.png"],
    ["Thief", "/cards/thief.png"],
    ["Politician", "/cards/politician.png"],
  ])("should return correct image path for %s", (role, expected) => {
    expect(cardImageForRole(role)).toBe(expected);
  });

  it.each<[string, null]>([
    ["", null],
    ["invalid", null],
    ["businesswoman", null], // case sensitive
    ["Duke", null],
    ["Captain", null],
  ])("should return null for invalid role '%s'", (role, expected) => {
    expect(cardImageForRole(role)).toBe(expected);
  });

  it("should handle whitespace-only strings", () => {
    expect(cardImageForRole("   ")).toBeNull();
    expect(cardImageForRole("\t\n")).toBeNull();
  });

  it("should handle strings with special characters", () => {
    expect(cardImageForRole("Businesswoman!")).toBeNull();
    expect(cardImageForRole("<script>")).toBeNull();
  });
});

describe("playSfx() - Audio Utilities", () => {
  let mockAudio: HTMLAudioElement;
  let consoleWarnSpy: ReturnType<typeof vi.spyOn>;
  let consoleErrorSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    mockAudio = {
      currentTime: 0,
      play: vi.fn().mockResolvedValue(undefined),
      src: "/audio/test.mp3",
    } as unknown as HTMLAudioElement;
    consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should not play when audio is null", () => {
    playSfx(null, false);
    expect(mockAudio.play).not.toHaveBeenCalled();
  });

  it("should not play when muted is true", () => {
    playSfx(mockAudio, true);
    expect(mockAudio.play).not.toHaveBeenCalled();
  });

  it("should reset currentTime before playing", () => {
    mockAudio.currentTime = 5;
    playSfx(mockAudio, false);
    expect(mockAudio.currentTime).toBe(0);
  });

  it("should call play() when not muted", () => {
    playSfx(mockAudio, false);
    expect(mockAudio.play).toHaveBeenCalledTimes(1);
  });

  it("should handle NotSupportedError with warning", async () => {
    const error = new Error("Not supported") as Error & { name: string };
    error.name = "NotSupportedError";
    mockAudio.play = vi.fn().mockRejectedValue(error);
    
    playSfx(mockAudio, false);
    await new Promise(resolve => setTimeout(resolve, 10));
    
    expect(consoleWarnSpy).toHaveBeenCalledWith(
      expect.stringContaining("could not be loaded")
    );
  });

  it("should handle NotAllowedError with warning", async () => {
    const error = new Error("Not allowed") as Error & { name: string };
    error.name = "NotAllowedError";
    mockAudio.play = vi.fn().mockRejectedValue(error);
    
    playSfx(mockAudio, false);
    await new Promise(resolve => setTimeout(resolve, 10));
    
    expect(consoleWarnSpy).toHaveBeenCalledWith(
      expect.stringContaining("blocked by browser")
    );
  });

  it("should handle generic errors with error log", async () => {
    const error = new Error("Generic error");
    mockAudio.play = vi.fn().mockRejectedValue(error);
    
    playSfx(mockAudio, false);
    await new Promise(resolve => setTimeout(resolve, 10));
    
    expect(consoleErrorSpy).toHaveBeenCalledWith(
      "SFX play failed:",
      error
    );
  });
});

describe("getReplayUrl() - Replay URL Generation", () => {
  const originalEnv = process.env;

  beforeEach(() => {
    process.env = { ...originalEnv };
    // Mock window.location for browser environment
    Object.defineProperty(global, "window", {
      value: {
        location: {
          hostname: "localhost",
          port: "3000",
          origin: "http://localhost:3000",
        },
      },
      writable: true,
    });
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  it("should generate URL with match ID only", () => {
    const url = getReplayUrl("match-123");
    expect(url).toContain("/replay/match-123");
  });

  it("should generate URL with viewer ID", () => {
    const url = getReplayUrl("match-123", "viewer-456");
    expect(url).toContain("/replay/match-123");
    expect(url).toContain("viewer_id=viewer-456");
  });

  it("should not include viewer_id param when viewerId is undefined", () => {
    const url = getReplayUrl("match-123");
    expect(url).not.toContain("viewer_id");
  });

  it("should handle special characters in match ID", () => {
    const url = getReplayUrl("match-abc-123_test");
    expect(url).toContain("match-abc-123_test");
  });
});

describe("fetchReplay() - Replay Fetching", () => {
  const originalFetch = global.fetch;

  beforeEach(() => {
    global.fetch = vi.fn();
  });

  afterEach(() => {
    global.fetch = originalFetch;
  });

  it("should fetch replay with correct headers", async () => {
    const mockResponse = {
      ok: true,
      json: vi.fn().mockResolvedValue({ match: { ID: "test" } }),
    };
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse);

    await fetchReplay("match-123", "viewer-456");

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/replay/match-123"),
      expect.objectContaining({
        headers: { Accept: "application/json" },
        cache: "no-store",
      })
    );
  });

  it("should throw error on non-ok response", async () => {
    const mockResponse = {
      ok: false,
      status: 404,
      text: vi.fn().mockResolvedValue("Not found"),
    };
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse);

    await expect(fetchReplay("match-123", "viewer-456")).rejects.toThrow("Not found");
  });

  it("should use status code in error message when no text", async () => {
    const mockResponse = {
      ok: false,
      status: 500,
      text: vi.fn().mockResolvedValue(""),
    };
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse);

    await expect(fetchReplay("match-123", "viewer-456")).rejects.toThrow("500");
  });

  it("should return parsed JSON on success", async () => {
    const mockData = {
      match: { ID: "match-123" },
      participants: [],
      events: [],
      snapshots: [],
      viewer_player_id: "viewer-456",
      viewer_index: 0,
    };
    const mockResponse = {
      ok: true,
      json: vi.fn().mockResolvedValue(mockData),
    };
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse);

    const result = await fetchReplay("match-123", "viewer-456");
    expect(result).toEqual(mockData);
  });
});

describe("roleForAction() - Action Role Mapping", () => {
  it.each<[string, string]>([
    ["businesswoman", "Businesswoman"],
    ["tax", "TaxCollector"],
    ["tax_business_woman", "TaxCollector"],
    ["investigate", "Policewoman"],
    ["block_investigate", "Policewoman"],
    ["accuse", "Colonel"],
    ["assassinate", "Terrorist"],
    ["block_terrorist", "Colonel"],
    ["steal", "Thief"],
    ["block_steal", "Thief"],
    ["exchange", "Politician"],
    ["block_foreign_aid", "TaxCollector"],
  ])("should map action '%s' to role '%s'", (action, role) => {
    expect(roleForAction(action)).toBe(role);
  });

  it.each<[string, null]>([
    ["income", null],
    ["foreign_aid", null],
    ["coup", null],
    ["", null],
    ["invalid_action", null],
    ["unknown", null],
  ])("should return null for action '%s' without role", (action, expected) => {
    expect(roleForAction(action)).toBe(expected);
  });
});

describe("actionLabel() - Action Label Generation", () => {
  it.each<[string, string]>([
    ["businesswoman", "take 4 coins"],
    ["tax", "collect tax"],
    ["tax_business_woman", "tax businesswoman"],
    ["investigate", "investigate"],
    ["accuse", "accuse"],
    ["assassinate", "assassinate"],
    ["steal", "steal 2 coins"],
    ["exchange", "exchange"],
    ["income", "income"],
    ["foreign_aid", "foreign aid"],
    ["coup", "coup"],
    ["block_foreign_aid", "block foreign aid"],
    ["block_investigate", "block investigate"],
    ["block_terrorist", "block assassinate"],
    ["block_steal", "block steal"],
  ])("should return '%s' for action '%s'", (action, label) => {
    expect(actionLabel(action)).toBe(label);
  });

  it("should format unknown actions by replacing underscores with spaces", () => {
    expect(actionLabel("custom_action")).toBe("custom action");
    expect(actionLabel("double__underscore")).toBe("double  underscore");
    expect(actionLabel("a_b_c_d")).toBe("a b c d");
  });

  it("should return action as-is when no override and no underscores", () => {
    expect(actionLabel("custom")).toBe("custom");
    expect(actionLabel("action")).toBe("action");
  });

  it("should handle empty string", () => {
    expect(actionLabel("")).toBe("");
  });
});

describe("mainActionImage() - Action Image Helpers", () => {
  it.each<[string, string]>([
    ["businesswoman", "/cards/business.png"],
    ["tax", "/cards/tax.png"],
    ["investigate", "/cards/police.png"],
    ["assassinate", "/cards/terrorist.png"],
    ["steal", "/cards/thief.png"],
    ["exchange", "/cards/politician.png"],
  ])("should return image for main action '%s'", (action, expected) => {
    expect(mainActionImage(action)).toBe(expected);
  });

  it.each<[string, null]>([
    ["income", null],
    ["foreign_aid", null],
    ["coup", null],
    ["invalid", null],
    ["", null],
  ])("should return null for action '%s' without associated role", (action, expected) => {
    expect(mainActionImage(action)).toBe(expected);
  });

  it("should return image for counter actions when used as main", () => {
    expect(mainActionImage("block_steal")).toBe("/cards/thief.png");
  });
});

describe("counterActionImage() - Counter Action Image Helpers", () => {
  it.each<[string, string]>([
    ["block_steal", "/cards/thief.png"],
    ["block_foreign_aid", "/cards/tax.png"],
    ["block_investigate", "/cards/police.png"],
    ["block_terrorist", "/cards/colonel.png"],
  ])("should return image for counter action '%s'", (action, expected) => {
    expect(counterActionImage(action)).toBe(expected);
  });

  it("should return image for main actions (same as counter)", () => {
    // These actions work for both main and counter
    expect(counterActionImage("steal")).toBe("/cards/thief.png");
    expect(counterActionImage("investigate")).toBe("/cards/police.png");
  });

  it.each<[string, null]>([
    ["income", null],
    ["foreign_aid", null],
    ["coup", null],
    ["invalid", null],
  ])("should return null for action '%s' without associated role", (action, expected) => {
    expect(counterActionImage(action)).toBe(expected);
  });
});

describe("Integration Tests - Cross-module Functionality", () => {
  it("should correctly chain action -> role -> image", () => {
    const action = "assassinate";
    const role = roleForAction(action);
    expect(role).toBe("Terrorist");
    
    const image = cardImageForRole(role!);
    expect(image).toBe("/cards/terrorist.png");
    
    expect(mainActionImage(action)).toBe(image);
  });

  it("should handle complete flow for all card-requiring actions", () => {
    const actionsRequiringCards = [
      "businesswoman",
      "tax",
      "investigate",
      "accuse",
      "assassinate",
      "steal",
      "exchange",
    ];
    
    actionsRequiringCards.forEach(action => {
      const role = roleForAction(action);
      expect(role).not.toBeNull();
      
      const image = cardImageForRole(role!);
      expect(image).not.toBeNull();
      expect(image).toMatch(/^\/cards\/[a-z]+\.png$/);
      
      expect(mainActionImage(action)).toBe(image);
    });
  });

  it("should have consistent labels for all counter actions", () => {
    const counterActions = [
      "block_steal",
      "block_foreign_aid",
      "block_investigate",
      "block_terrorist",
    ];
    
    counterActions.forEach(action => {
      const label = actionLabel(action);
      expect(label).toContain("block");
      expect(label.length).toBeGreaterThan(0);
    });
  });
});
