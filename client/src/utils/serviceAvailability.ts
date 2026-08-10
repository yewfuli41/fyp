import { getAvailableDates, type PublicServiceOption } from "../services/PublicService";
import { getOptionStatus } from "./serviceFormHelpers";
import { toISO } from "./serviceSlotHelpers";

// Options past their effectiveUntil date ("Previous" in the owner's Manage
// Services view) are no longer bookable — customers browsing shouldn't see
// them at all.
export const isCurrentOption = (opt: PublicServiceOption) =>
    getOptionStatus(opt.effectiveFrom, opt.effectiveUntil) !== "expired";

// How far ahead to look for a bookable slot before giving up on an option —
// generous enough to catch options whose owner only just opened up next
// month's slots, without checking indefinitely far into the future.
export const AVAILABILITY_WINDOW_DAYS = 180;

// Checks every given option (all belonging to the same business) in
// parallel and returns the ids of the ones that have at least one bookable
// slot within the window — an option can be "current" (not expired) but
// fully booked out or not yet slotted at all, in which case there's nothing
// to book.
export const findBookableOptionIds = async (
    businessId: string,
    options: PublicServiceOption[],
): Promise<Set<string>> => {
    if (options.length === 0) return new Set();
    const today = toISO(new Date());
    const until = toISO(new Date(Date.now() + AVAILABILITY_WINDOW_DAYS * 24 * 60 * 60 * 1000));
    const entries = await Promise.all(options.map(async opt => {
        try {
            const res = await getAvailableDates(businessId, today, until, undefined, undefined, opt.serviceOptionId);
            return [opt.serviceOptionId, (res.data?.availableDates.length ?? 0) > 0] as const;
        } catch {
            return [opt.serviceOptionId, false] as const;
        }
    }));
    return new Set(entries.filter(([, bookable]) => bookable).map(([id]) => id));
};
