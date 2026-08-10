import { useEffect, useState } from "react";
import { Spinner } from "react-bootstrap";
import { getAvailableDates } from "../services/PublicService";
import { toISO, todayISO } from "../utils/serviceSlotHelpers";
import { IconChevronLeft, IconChevronRight } from "./icons";
import "../styles/BookingDatePicker.css";

interface Props {
    businessId: string;
    // Omit to check across every service in the business (e.g. reschedule,
    // which isn't tied to one service).
    serviceId?: string;
    // Narrows further to one specific option of that service — pass this
    // once the customer has already locked in an option so the calendar only
    // colours dates where THAT option (not just any option of the service)
    // has a bookable slot.
    serviceOptionId?: string;
    // Omit to check across every staff member (e.g. the owner rescheduling);
    // pass a specific staff id to see only that staff's own slots (e.g. a
    // staff member's own reschedule picker).
    staffId?: string;
    // Narrows instead to owner-managed (no staff assigned) slots — wins over
    // staffId if both are somehow passed.
    unassignedOnly?: boolean;
    value: string;
    onChange: (isoDate: string) => void;
}

const WEEKDAY_LABELS = ["Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"];
const MONTH_LABEL = (d: Date) => d.toLocaleDateString("en-GB", { month: "long", year: "numeric" });

// A visual, per-day green/red month calendar — a native <input type="date">
// can't be styled per-day, so this replaces it wherever picking a date
// should be informed by slot availability (booking, reschedule).
export default function BookingDatePicker({
    businessId, serviceId, serviceOptionId, staffId, unassignedOnly, value, onChange,
}: Props) {
    const today = todayISO();
    const selected = new Date(`${value || today}T00:00:00`);
    const [viewYear, setViewYear] = useState(selected.getFullYear());
    const [viewMonth, setViewMonth] = useState(selected.getMonth());
    const [availableDates, setAvailableDates] = useState<Set<string> | null>(null);

    useEffect(() => {
        const firstOfMonth = new Date(viewYear, viewMonth, 1);
        const lastOfMonth = new Date(viewYear, viewMonth + 1, 0);
        setAvailableDates(null);
        getAvailableDates(
            businessId, toISO(firstOfMonth), toISO(lastOfMonth), serviceId, staffId, serviceOptionId, unassignedOnly,
        )
            .then(res => setAvailableDates(new Set(res.data?.availableDates ?? [])))
            .catch(() => setAvailableDates(new Set()));
    }, [businessId, serviceId, serviceOptionId, staffId, unassignedOnly, viewYear, viewMonth]);

    const firstOfMonth = new Date(viewYear, viewMonth, 1);
    const daysInMonth = new Date(viewYear, viewMonth + 1, 0).getDate();
    const leadingBlanks = (firstOfMonth.getDay() + 6) % 7; // Mon-first

    const todayDate = new Date(`${today}T00:00:00`);
    const canGoPrev = viewYear > todayDate.getFullYear()
        || (viewYear === todayDate.getFullYear() && viewMonth > todayDate.getMonth());

    const changeMonth = (delta: number) => {
        const next = new Date(viewYear, viewMonth + delta, 1);
        setViewYear(next.getFullYear());
        setViewMonth(next.getMonth());
    };

    const cells: (string | null)[] = [
        ...Array(leadingBlanks).fill(null),
        ...Array.from({ length: daysInMonth }, (_, i) => toISO(new Date(viewYear, viewMonth, i + 1))),
    ];

    return (
        <div className="booking-date-picker">
            <div className="booking-date-picker-header">
                <button
                    type="button"
                    className="booking-date-picker-nav"
                    onClick={() => changeMonth(-1)}
                    disabled={!canGoPrev}
                    aria-label="Previous month"
                >
                    <IconChevronLeft size={14} />
                </button>
                <div className="fw-semibold">{MONTH_LABEL(firstOfMonth)}</div>
                <button
                    type="button"
                    className="booking-date-picker-nav"
                    onClick={() => changeMonth(1)}
                    aria-label="Next month"
                >
                    <IconChevronRight size={14} />
                </button>
            </div>

            <div className="booking-date-picker-grid booking-date-picker-weekdays">
                {WEEKDAY_LABELS.map(w => <div key={w}>{w}</div>)}
            </div>

            {availableDates === null ? (
                <div className="text-center py-4"><Spinner size="sm" /></div>
            ) : (
                <div className="booking-date-picker-grid">
                    {cells.map((iso, i) => {
                        if (!iso) return <div key={`blank-${i}`} />;
                        const isPast = iso < today;
                        const hasSlots = availableDates.has(iso);
                        const isSelected = iso === value;
                        const classes = ["booking-date-picker-day"];
                        if (isPast) classes.push("is-past");
                        else classes.push(hasSlots ? "has-slots" : "no-slots");
                        if (isSelected) classes.push("is-selected");
                        return (
                            <button
                                key={iso}
                                type="button"
                                className={classes.join(" ")}
                                disabled={isPast}
                                onClick={() => onChange(iso)}
                            >
                                {Number(iso.slice(-2))}
                            </button>
                        );
                    })}
                </div>
            )}

            <div className="booking-date-picker-legend">
                <span><i className="booking-date-picker-dot has-slots" /> Has availability</span>
                <span><i className="booking-date-picker-dot no-slots" /> Fully booked</span>
            </div>
        </div>
    );
}
