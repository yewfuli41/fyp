// One of these accent tokens (defined in booking-theme.css) is assigned to
// each business's avatar tile, deterministically by id — so the same
// business always shows the same colour everywhere it appears, and a
// repeat visitor can recognise it by tile colour alone.
const AVATAR_COLORS = [
    "var(--bk-primary)",
    "var(--bk-violet)",
    "var(--bk-teal)",
    "var(--bk-coral)",
    "var(--bk-rose)",
    "var(--bk-amber)",
];

export function businessAvatarColor(businessId: string): string {
    let hash = 0;
    for (let i = 0; i < businessId.length; i++) {
        hash = (hash * 31 + businessId.charCodeAt(i)) | 0;
    }
    return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length];
}
