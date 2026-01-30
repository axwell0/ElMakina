import type {CardRole} from "./cards";
import {cardImageForRole} from "./cards";

const ACTION_ROLE: Record<string, CardRole> = {
    businesswoman: "Businesswoman",
    tax: "TaxCollector",
    tax_business_woman: "TaxCollector",
    investigate: "Policewoman",
    block_investigate: "Policewoman",
    accuse: "Colonel",
    assassinate: "Terrorist",
    block_terrorist: "Colonel",
    steal: "Thief",
    block_steal: "Thief",
    exchange: "Politician",
    block_foreign_aid: "TaxCollector",
};

const ACTION_LABEL_OVERRIDES: Record<string, string> = {
    businesswoman: "take 4 coins",
    tax: "collect tax",
    tax_business_woman: "tax businesswoman",
    investigate: "investigate",
    accuse: "accuse",
    assassinate: "assassinate",
    steal: "steal 2 coins",
    exchange: "exchange",
    income: "income",
    foreign_aid: "foreign aid",
    coup: "coup",
    block_foreign_aid: "block foreign aid",
    block_investigate: "block investigate",
    block_terrorist: "block assassinate",
    block_steal: "block steal",
};

export function roleForAction(actionId: string): CardRole | null {
    return ACTION_ROLE[actionId] ?? null;
}

export function actionLabel(actionId: string): string {
    return ACTION_LABEL_OVERRIDES[actionId] ?? actionId.replace(/_/g, " ");
}

export const CHALLENGE_IMAGE = "/cards/colonel.png";

export function mainActionImage(actionId: string): string | null {
    const role = roleForAction(actionId);
    return role ? cardImageForRole(role) : null;
}

export function counterActionImage(actionId: string): string | null {
    const role = roleForAction(actionId);
    return role ? cardImageForRole(role) : null;
}
