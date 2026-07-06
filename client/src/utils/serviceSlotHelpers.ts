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

const nowHHMM = (): string => {
    const now = new Date();
    return `${String(now.getHours()).padStart(2, "0")}:${String(now.getMinutes()).padStart(2, "0")}`;
};

export const cap = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

export const emptyForm = (date: string): ServiceSlotInput => ({
    staffId: "", date, daysOfWeek: [], startTime: DEFAULT_START, endTime: DEFAULT_END,
    serviceOptionIds: [],
});

// A service's first package (lowest serviceOptionId) is always its
// auto-created default package (named after the service itself), so
// selecting a service can safely pre-check it as the initial package choice.
export const defaultOptionId = (services: Service[], serviceId: string): string | undefined =>
    services.find(s => s.serviceId === serviceId)?.serviceOptions[0]?.serviceOptionId;

// Allowed slot times = working-hours envelope of the chosen staff (or the
// business, for owner-managed) across the selected weekday(s) / date. Shared
// by the Add and Edit forms.
export const computeTimeOptions = (
    input: ServiceSlotInput, staffList: Staff[], workingHours: WorkingHour[], lines: string[],
): string[] => {
    const days = input.daysOfWeek.length > 0
        ? input.daysOfWeek
        : (input.date ? [weekdayName(input.date)] : []);
    const source: WorkingHour[] = input.staffId
        ? (staffList.find(s => s.staffId === input.staffId)?.workingHours ?? [])
        : workingHours;
    const relevant = source.filter(wh => days.length === 0 || days.includes(wh.day));
    if (relevant.length === 0) return days.length === 0 ? lines : [];
    let min = "23:59", max = "00:00";
    for (const wh of relevant) {
        const s = extractTime(wh.startTime), e = extractTime(wh.endTime);
        if (s < min) min = s;
        if (e > max) max = e;
    }
    const options = timeLines(min, max);
    // A single date (not a recurring weekday) that's today can't offer times
    // that have already passed — the backend rejects those anyway.
    if (!input.date || input.date !== todayISO()) return options;
    const now = nowHHMM();
    return options.filter(t => t > now);
};
