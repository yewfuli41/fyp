import type { Service } from "../services/ServiceService";
import type { Staff, WorkingHour } from "../services/StaffService";
import { extractTime, type ServiceSlotInput } from "../services/ServiceSlotService";
import { DAYS_OF_WEEK } from "./time";

export type { WorkingHour };

export const DEFAULT_START = "09:00";
export const DEFAULT_END = "18:00";
export const NONE_COL = "__none__"; // calendar column for owner-managed (unassigned) slots

// 30-min time lines between start and end, inclusive of end.
export const timeLines = (start: string, end: string): string[] => {
    const [sh, sm] = start.split(":").map(Number);
    const [eh, em] = end.split(":").map(Number);
    const startMin = sh * 60 + sm;
    const endMin = eh * 60 + em;
    const lines: string[] = [];
    for (let m = startMin; m <= endMin; m += 30) {
        lines.push(`${String(Math.floor(m / 60)).padStart(2, "0")}:${String(m % 60).padStart(2, "0")}`);
    }
    return lines;
};

// weekday name (lowercase) of a YYYY-MM-DD date, timezone-safe.
export const weekdayName = (date: string): string => {
    const d = new Date(`${date}T00:00:00`);
    return DAYS_OF_WEEK[(d.getDay() + 6) % 7]; // getDay: 0=Sun → our array is Mon-first
};

export const toISO = (d: Date): string =>
    `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;

export const displayDate = (d: Date): string =>
    d.toLocaleDateString("en-GB", { weekday: "short", day: "2-digit", month: "short", year: "numeric" });

export const addDays = (d: Date, n: number): Date => {
    const next = new Date(d);
    next.setDate(next.getDate() + n);
    return next;
};

export const todayISO = (): string => toISO(new Date());

export const nowHHMM = (): string => {
    const now = new Date();
    return `${String(now.getHours()).padStart(2, "0")}:${String(now.getMinutes()).padStart(2, "0")}`;
};

export const cap = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

export const emptyForm = (date: string): ServiceSlotInput => ({
    staffId: "", date, daysOfWeek: [], startTime: DEFAULT_START, endTime: DEFAULT_END,
    serviceOptionIds: [],
});

// Selecting a service pre-checks its first option that's actually offered on
// the relevant date. Note: `removed` means "no version of this option's group
// is effective today" (see IsRemoved in serviceRepo.go) — it's relative to
// today, not to the date being scheduled for, so it must NOT gate this
// alongside isOptionOfferedOn(date): an upcoming option can be `removed` today
// while still being exactly what should be offered for a future slot date.
export const defaultOptionId = (services: Service[], serviceId: string, date: string, daysOfWeek: string[] = []): string | undefined =>
    services.find(s => s.serviceId === serviceId)?.serviceOptions
        .find(option => isOptionSelectableFor(option, date, daysOfWeek))?.serviceOptionId;

// Whether an option's effective window covers the given date — used wherever
// a specific date (not just "today") determines which options are offerable:
// creating/editing a slot, and a customer picking a booking date.
export const isOptionOfferedOn = (opt: { effectiveFrom?: string; effectiveUntil?: string }, date: string): boolean =>
    (!opt.effectiveFrom || opt.effectiveFrom <= date) && (!opt.effectiveUntil || opt.effectiveUntil >= date);

// Whether an option should be offered in the Add-slot form for the current
// schedule choice. A single date has one concrete occurrence, so this is
// exactly isOptionOfferedOn for that date. A weekday-recurring schedule has
// no single date — it spans many future occurrences — so an "upcoming"
// option (effectiveFrom still ahead) must stay selectable here too: the
// backend only creates occurrences once the option actually becomes
// effective, and stops creating them past its effectiveUntil. Filtering by
// "effective today" would hide upcoming/limited-time options from recurring
// schedules entirely, even though they're exactly what recurring schedules
// need to support.
export const isOptionSelectableFor = (
    opt: { effectiveFrom?: string; effectiveUntil?: string }, date: string, daysOfWeek: string[],
): boolean =>
    daysOfWeek.length > 0
        ? !opt.effectiveUntil || opt.effectiveUntil >= todayISO()
        : isOptionOfferedOn(opt, date);

// A column's open time intervals ("HH:MM"–"HH:MM") on a given weekday —
// used by the calendar grid to gray out cells outside working hours.
export const openIntervals = (workingHours: WorkingHour[], weekday: string): { start: string; end: string }[] =>
    workingHours
        .filter(wh => wh.day === weekday)
        .map(wh => ({ start: extractTime(wh.startTime), end: extractTime(wh.endTime) }));

// Allowed slot times = working-hours window shared by EVERY selected weekday
// for the chosen staff (or the business, for owner-managed) — the
// intersection, not the union. One start/end time is applied to all selected
// days at once, so a day with no working hours at all (or a narrower window
// than the others) must constrain the result, not be silently ignored: with
// a union, checking "Sunday" (no hours) alongside "Monday" (9-18) would still
// show Monday's times as if they applied to both days, and nothing would stop
// a slot from being created on Sunday too.
export const computeTimeOptions = (
    input: ServiceSlotInput, staffList: Staff[], workingHours: WorkingHour[], lines: string[],
): string[] => {
    const days = input.daysOfWeek.length > 0
        ? input.daysOfWeek
        : (input.date ? [weekdayName(input.date)] : []);
    if (days.length === 0) return lines;
    const source: WorkingHour[] = input.staffId
        ? (staffList.find(s => s.staffId === input.staffId)?.workingHours ?? [])
        : workingHours;

    let min = "00:00", max = "23:59";
    for (const day of days) {
        const dayHours = source.filter(wh => wh.day === day);
        if (dayHours.length === 0) return []; // this day isn't worked at all — no time can cover every selected day
        let dayMin = "23:59", dayMax = "00:00";
        for (const wh of dayHours) {
            const s = extractTime(wh.startTime), e = extractTime(wh.endTime);
            if (s < dayMin) dayMin = s;
            if (e > dayMax) dayMax = e;
        }
        if (dayMin > min) min = dayMin;
        if (dayMax < max) max = dayMax;
    }
    if (min >= max) return [];
    const options = timeLines(min, max);
    // A single date (not a recurring weekday) that's today can't offer times
    // that have already passed — the backend rejects those anyway.
    if (!input.date || input.date !== todayISO()) return options;
    const now = nowHHMM();
    return options.filter(t => t > now);
};
