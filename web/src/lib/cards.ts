export type CardRole =
    | "Businesswoman"
    | "TaxCollector"
    | "Policewoman"
    | "Colonel"
    | "Terrorist"
    | "Thief"
    | "Politician";

const ROLE_IMAGE: Record<CardRole, string> = {
    Businesswoman: "/cards/business.png",
    TaxCollector: "/cards/tax.png",
    Policewoman: "/cards/police.png",
    Colonel: "/cards/colonel.png",
    Terrorist: "/cards/terrorist.png",
    Thief: "/cards/thief.png",
    Politician: "/cards/politician.png",
};

export function cardImageForRole(role: string): string | null {
    if (role in ROLE_IMAGE) {
        return ROLE_IMAGE[role as CardRole];
    }
    return null;
}
