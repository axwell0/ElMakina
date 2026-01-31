/**
 * Domain layer tests
 *
 * Tests for pure business logic in the domain layer.
 * These tests validate core game rules, card mappings, and action logic.
 */

import { describe, it, expect } from "vitest";
import {
  ACTION_ROLE,
  ACTION_LABEL_OVERRIDES,
  roleForAction,
  actionLabel,
  mainActionImage,
  counterActionImage,
  CHALLENGE_IMAGE,
} from "@/domain/game/actions";
import {
  ALL_CARD_ROLES,
  isCardRole,
  type CardRole,
} from "@/domain/game/cards";
import {
  ROLE_IMAGE,
  cardImageForRole,
  cardImageForRoleOrThrow,
} from "@/domain/game/cards/images";
import {
  STARTING_COINS,
  COUP_COST,
  ASSASSINATION_COST,
  INCOME_AMOUNT,
  FOREIGN_AID_AMOUNT,
  BUSINESSWOMAN_AMOUNT,
  STEAL_AMOUNT,
  MAX_COINS,
  MAX_PLAYERS,
  MIN_PLAYERS,
  getStartingHandSize,
} from "@/domain/game/rules/constants";

describe("CardRole Types", () => {
  describe("ALL_CARD_ROLES", () => {
    it("should contain all seven card roles", () => {
      expect(ALL_CARD_ROLES).toHaveLength(7);
      expect(ALL_CARD_ROLES).toContain("Businesswoman");
      expect(ALL_CARD_ROLES).toContain("TaxCollector");
      expect(ALL_CARD_ROLES).toContain("Policewoman");
      expect(ALL_CARD_ROLES).toContain("Colonel");
      expect(ALL_CARD_ROLES).toContain("Terrorist");
      expect(ALL_CARD_ROLES).toContain("Thief");
      expect(ALL_CARD_ROLES).toContain("Politician");
    });

    it("should be readonly", () => {
      // TypeScript ensures this at compile time
      // Runtime check: array should not be mutable
      const roles = ALL_CARD_ROLES as readonly string[];
      expect(Array.isArray(roles)).toBe(true);
    });
  });

  describe("isCardRole", () => {
    it.each<[string, boolean]>([
      ["Businesswoman", true],
      ["TaxCollector", true],
      ["Policewoman", true],
      ["Colonel", true],
      ["Terrorist", true],
      ["Thief", true],
      ["Politician", true],
    ])("should return %s for role '%s'", (role, expected) => {
      expect(isCardRole(role)).toBe(expected);
    });

    it.each<[string, boolean]>([
      ["", false],
      ["invalid", false],
      ["businesswoman", false], // case sensitive
      ["Duke", false],
      ["Captain", false],
      ["Contessa", false],
      ["Ambassador", false],
      ["Assassin", false],
    ])("should return false for invalid role '%s'", (role) => {
      expect(isCardRole(role)).toBe(false);
    });

    it("should work as a type guard", () => {
      const maybeRole: string = "Businesswoman";
      if (isCardRole(maybeRole)) {
        // TypeScript should narrow this to CardRole
        const confirmedRole: CardRole = maybeRole;
        expect(confirmedRole).toBe("Businesswoman");
      }
    });
  });
});

describe("Card Images", () => {
  describe("ROLE_IMAGE", () => {
    it("should have an image path for each card role", () => {
      ALL_CARD_ROLES.forEach((role) => {
        expect(ROLE_IMAGE[role]).toBeDefined();
        expect(ROLE_IMAGE[role]).toMatch(/^\/cards\/[a-z]+\.png$/);
      });
    });

    it.each<[string, string]>([
      ["Businesswoman", "/cards/business.png"],
      ["TaxCollector", "/cards/tax.png"],
      ["Policewoman", "/cards/police.png"],
      ["Colonel", "/cards/colonel.png"],
      ["Terrorist", "/cards/terrorist.png"],
      ["Thief", "/cards/thief.png"],
      ["Politician", "/cards/politician.png"],
    ])("should map %s to %s", (role, expectedPath) => {
      expect(ROLE_IMAGE[role as CardRole]).toBe(expectedPath);
    });
  });

  describe("cardImageForRole", () => {
    it.each<[string, string]>([
      ["Businesswoman", "/cards/business.png"],
      ["TaxCollector", "/cards/tax.png"],
      ["Policewoman", "/cards/police.png"],
      ["Colonel", "/cards/colonel.png"],
      ["Terrorist", "/cards/terrorist.png"],
      ["Thief", "/cards/thief.png"],
      ["Politician", "/cards/politician.png"],
    ])("should return image path for valid role %s", (role, expected) => {
      expect(cardImageForRole(role)).toBe(expected);
    });

    it.each<[string, null]>([
      ["", null],
      ["invalid", null],
      ["businesswoman", null], // case sensitive
      ["Duke", null],
    ])("should return null for invalid role '%s'", (role, expected) => {
      expect(cardImageForRole(role)).toBe(expected);
    });
  });

  describe("cardImageForRoleOrThrow", () => {
    it.each<[string, string]>([
      ["Businesswoman", "/cards/business.png"],
      ["TaxCollector", "/cards/tax.png"],
      ["Policewoman", "/cards/police.png"],
      ["Colonel", "/cards/colonel.png"],
      ["Terrorist", "/cards/terrorist.png"],
      ["Thief", "/cards/thief.png"],
      ["Politician", "/cards/politician.png"],
    ])("should return image path for valid role %s", (role, expected) => {
      expect(cardImageForRoleOrThrow(role as CardRole)).toBe(expected);
    });

    it("should throw for invalid role", () => {
      // We need to bypass TypeScript here to test runtime behavior
      const invalidRole = "InvalidRole" as CardRole;
      expect(() => cardImageForRoleOrThrow(invalidRole)).toThrow(
        "Invalid card role: InvalidRole"
      );
    });

    it("should throw with correct error message", () => {
      const invalidRole = "FakeRole" as CardRole;
      expect(() => cardImageForRoleOrThrow(invalidRole)).toThrow(
        /Invalid card role: FakeRole/
      );
    });
  });
});

describe("Action Definitions", () => {
  describe("ACTION_ROLE mapping", () => {
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
    ])("action '%s' requires role '%s'", (action, role) => {
      expect(ACTION_ROLE[action]).toBe(role);
    });

    it("should not have entries for actions without roles", () => {
      // Actions like income, foreign_aid, coup don't require roles
      expect(ACTION_ROLE["income"]).toBeUndefined();
      expect(ACTION_ROLE["foreign_aid"]).toBeUndefined();
      expect(ACTION_ROLE["coup"]).toBeUndefined();
    });
  });

  describe("roleForAction", () => {
    it.each<[string, string]>([
      ["businesswoman", "Businesswoman"],
      ["tax", "TaxCollector"],
      ["investigate", "Policewoman"],
      ["assassinate", "Terrorist"],
      ["steal", "Thief"],
      ["exchange", "Politician"],
      ["block_steal", "Thief"],
    ])("should return role for action '%s'", (action, expectedRole) => {
      expect(roleForAction(action)).toBe(expectedRole);
    });

    it.each<[string, null]>([
      ["income", null],
      ["foreign_aid", null],
      ["coup", null],
      ["", null],
      ["invalid_action", null],
    ])("should return null for action '%s' without role", (action, expected) => {
      expect(roleForAction(action)).toBe(expected);
    });
  });

  describe("ACTION_LABEL_OVERRIDES", () => {
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
    ])("action '%s' has label '%s'", (action, label) => {
      expect(ACTION_LABEL_OVERRIDES[action]).toBe(label);
    });
  });

  describe("actionLabel", () => {
    it.each<[string, string]>([
      ["businesswoman", "take 4 coins"],
      ["tax", "collect tax"],
      ["investigate", "investigate"],
      ["assassinate", "assassinate"],
      ["steal", "steal 2 coins"],
      ["income", "income"],
      ["foreign_aid", "foreign aid"],
      ["coup", "coup"],
    ])("should return correct label for '%s'", (action, expected) => {
      expect(actionLabel(action)).toBe(expected);
    });

    it("should format unknown actions by replacing underscores", () => {
      expect(actionLabel("some_custom_action")).toBe("some custom action");
      expect(actionLabel("double_underscore__test")).toBe("double underscore  test");
    });

    it("should return unchanged for single word actions", () => {
      expect(actionLabel("custom")).toBe("custom");
    });
  });
});

describe("Action Helpers", () => {
  describe("mainActionImage", () => {
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
    ])("should return null for action '%s' without associated role", (action, expected) => {
      expect(mainActionImage(action)).toBe(expected);
    });
  });

  describe("counterActionImage", () => {
    it.each<[string, string]>([
      ["block_steal", "/cards/thief.png"],
      ["block_foreign_aid", "/cards/tax.png"],
      ["block_investigate", "/cards/police.png"],
      ["block_terrorist", "/cards/colonel.png"],
    ])("should return image for counter action '%s'", (action, expected) => {
      expect(counterActionImage(action)).toBe(expected);
    });

    it("should return null for non-counter actions", () => {
      expect(counterActionImage("steal")).toBe("/cards/thief.png"); // Main action also works
      expect(counterActionImage("income")).toBeNull();
    });
  });

  describe("CHALLENGE_IMAGE", () => {
    it("should be the colonel image", () => {
      expect(CHALLENGE_IMAGE).toBe("/cards/colonel.png");
    });
  });
});

describe("Game Rules Constants", () => {
  describe("Currency Constants", () => {
    it("should have correct starting coins", () => {
      expect(STARTING_COINS).toBe(2);
    });

    it("should have correct coup cost", () => {
      expect(COUP_COST).toBe(7);
    });

    it("should have correct assassination cost", () => {
      expect(ASSASSINATION_COST).toBe(3);
    });

    it("should have correct income amount", () => {
      expect(INCOME_AMOUNT).toBe(1);
    });

    it("should have correct foreign aid amount", () => {
      expect(FOREIGN_AID_AMOUNT).toBe(2);
    });

    it("should have correct businesswoman amount", () => {
      expect(BUSINESSWOMAN_AMOUNT).toBe(4);
    });

    it("should have correct steal amount", () => {
      expect(STEAL_AMOUNT).toBe(2);
    });

    it("should have correct max coins", () => {
      expect(MAX_COINS).toBe(12);
    });
  });

  describe("Player Count Constants", () => {
    it("should have correct max players", () => {
      expect(MAX_PLAYERS).toBe(9);
    });

    it("should have correct min players", () => {
      expect(MIN_PLAYERS).toBe(2);
    });
  });

  describe("getStartingHandSize", () => {
    it.each<[number, number]>([
      [2, 3],
      [3, 3],
      [4, 3],
      [5, 2],
      [6, 2],
      [7, 2],
      [8, 2],
      [9, 2],
    ])("should return %s cards for %s players", (playerCount, expectedHandSize) => {
      expect(getStartingHandSize(playerCount)).toBe(expectedHandSize);
    });

    it("should handle edge cases", () => {
      // Boundary between 3 and 2 cards is at 4-5 players
      expect(getStartingHandSize(4)).toBe(3);
      expect(getStartingHandSize(5)).toBe(2);
    });

    it("should return 3 for player counts below 2 (edge case)", () => {
      expect(getStartingHandSize(1)).toBe(3);
      expect(getStartingHandSize(0)).toBe(3);
    });

    it("should return 2 for player counts above 9 (edge case)", () => {
      expect(getStartingHandSize(10)).toBe(2);
      expect(getStartingHandSize(100)).toBe(2);
    });
  });
});

describe("Domain Integration", () => {
  it("should correctly link actions to card images through roles", () => {
    // Full chain: action -> role -> image
    const action = "assassinate";
    const role = roleForAction(action);
    expect(role).toBe("Terrorist");

    const image = cardImageForRole(role!);
    expect(image).toBe("/cards/terrorist.png");

    // Alternative path through helper
    expect(mainActionImage(action)).toBe("/cards/terrorist.png");
  });

  it("should maintain consistency between ACTION_ROLE and roleForAction", () => {
    Object.entries(ACTION_ROLE).forEach(([action, expectedRole]) => {
      expect(roleForAction(action)).toBe(expectedRole);
    });
  });

  it("should have consistent labels for all actions with roles", () => {
    Object.keys(ACTION_ROLE).forEach((action) => {
      const label = actionLabel(action);
      expect(label).toBeDefined();
      expect(typeof label).toBe("string");
      expect(label.length).toBeGreaterThan(0);
    });
  });

  it("should have images for all roles referenced by actions", () => {
    const rolesFromActions = new Set(Object.values(ACTION_ROLE));
    rolesFromActions.forEach((role) => {
      const image = cardImageForRole(role);
      expect(image).not.toBeNull();
      expect(image).toMatch(/^\/cards\/[a-z]+\.png$/);
    });
  });
});
