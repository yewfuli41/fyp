import { useMemo } from "react";
import { Alert } from "react-bootstrap";
import { extractTime, type ServiceSlot, type ServiceSlotInput } from "../services/ServiceSlotService";
import { NONE_COL } from "../utils/serviceSlotHelpers";
import type { BookingDetail } from "../services/BookingService";

// One status class per legend entry in CalendarLegend — a slot with no active
// booking (not in bookingBySlot) is "available".
const slotStatusClass = (booking: BookingDetail | undefined): string => {
    switch (booking?.status) {
        case "ACCEPTED": return "calendar-slot-accepted";
        case "RESCHEDULED": return "calendar-slot-reschedule-pending";
        case "PENDING": return "calendar-slot-pending";
        default: return "calendar-slot-available";
    }
};

const LANE_MIN_WIDTH = 90; // px per overlapping-slot lane
const ROW_HEIGHT = 48; // px per 30-min interval

interface ColumnLayout {
    staffId: string;
    name: string;
    lane: Map<string, number>;
    laneCount: number;
    gridColStart: number;
}

// Slots for the same staff can now overlap (backend no longer rejects it), so
// within one staff column each slot gets its own "lane" (like Google/Outlook
// calendar's side-by-side layout) via greedy interval scheduling: sorted by
// start time, each slot takes the first lane whose previous slot has already
// ended, otherwise it opens a new lane.
const assignLanes = (colSlots: ServiceSlot[]): { lane: Map<string, number>; laneCount: number } => {
    const sorted = [...colSlots].sort(
        (a, b) => extractTime(a.startTime).localeCompare(extractTime(b.startTime))
    );
    const laneEnds: string[] = [];
    const lane = new Map<string, number>();
    for (const slot of sorted) {
        const start = extractTime(slot.startTime);
        const end = extractTime(slot.endTime);
        let laneIdx = laneEnds.findIndex(e => e <= start);
        if (laneIdx === -1) {
            laneIdx = laneEnds.length;
            laneEnds.push(end);
        } else {
            laneEnds[laneIdx] = end;
        }
        lane.set(slot.serviceSlotId, laneIdx);
    }
    return { lane, laneCount: Math.max(laneEnds.length, 1) };
};

interface CalendarGridProps {
    staffColumns: { staffId: string; name: string }[];
    lines: string[];
    intervals: number;
    rangeEnd: string;
    slots: ServiceSlot[];
    // Each column's open ("working hours") intervals on the currently
    // selected day — cells outside all of a column's intervals are grayed out.
    openHoursByColumn: Record<string, { start: string; end: string }[]>;
    // Whether that column's staff has an approved leave covering the
    // selected day — takes over the whole column with a distinct shaded
    // look and blocks adding new slots there, regardless of working hours.
    onLeaveByColumn: Record<string, boolean>;
    onAddSlot: (prefill: Partial<ServiceSlotInput>) => void;
    onManageSlot: (slot: ServiceSlot) => void;
    // Active (pending/accepted/rescheduled) booking per serviceSlotId, if any
    // — drives each slot block's status colour (see CalendarLegend).
    bookingBySlot: Record<string, BookingDetail>;
}

const isCellOpen = (intervals: { start: string; end: string }[], start: string, end: string): boolean =>
    intervals.some(iv => start >= iv.start && end <= iv.end);

const toMinutes = (hhmm: string): number => {
    const [h, m] = hhmm.split(":").map(Number);
    return h * 60 + m;
};

// Renders the CSS-grid calendar (staff headers, time labels, clickable
// background cells, and slot blocks) — including the lane-assignment layout
// math for overlapping slots. Filtering, fetching, and API calls live in CalendarPage.
export default function CalendarGrid({
    staffColumns, lines, intervals, rangeEnd, slots, openHoursByColumn, onLeaveByColumn,
    onAddSlot, onManageSlot, bookingBySlot,
}: CalendarGridProps) {
    // Per-column lane assignment (for overlapping slots) plus each column's
    // starting grid-column index and how many lane-columns it spans.
    const columnLayout: ColumnLayout[] = useMemo(() => {
        let gridColStart = 2; // column 1 is the time-label column
        return staffColumns.map(col => {
            const colSlots = slots.filter(slot => (slot.staff ? slot.staff.staffId : NONE_COL) === col.staffId);
            const { lane, laneCount } = assignLanes(colSlots);
            const layout = { ...col, lane, laneCount, gridColStart };
            gridColStart += laneCount;
            return layout;
        });
    }, [staffColumns, slots]);
    const totalLanes = columnLayout.reduce((sum, col) => sum + col.laneCount, 0);
    // Slot times aren't guaranteed to land on a 30-min grid line (e.g. a
    // walk-in recorded at whatever time "now" happens to be) — position slot
    // blocks by exact minute offset rather than snapping to `lines` indices.
    const rangeStartMinutes = toMinutes(lines[0] ?? "00:00");
    const totalHeight = intervals * ROW_HEIGHT;

    if (columnLayout.length === 0) {
        return <Alert variant="info">No staff to display. Add staff to start scheduling slots.</Alert>;
    }

    return (
        <div className="calendar-grid-wrapper">
            <div
                className="calendar-grid"
                style={{
                    gridTemplateColumns: `70px repeat(${totalLanes}, minmax(${LANE_MIN_WIDTH}px, 1fr))`,
                    gridTemplateRows: `42px repeat(${intervals}, ${ROW_HEIGHT}px)`,
                    minWidth: 70 + totalLanes * LANE_MIN_WIDTH,
                }}
            >
                {/* top-left corner (sticky on both axes) */}
                <div className="calendar-corner" />

                {/* staff header row (sticky to top while scrolling) — spans all of that staff's lanes */}
                {columnLayout.map(col => (
                    <div
                        key={col.staffId}
                        className="calendar-header-cell"
                        style={{ gridColumn: `${col.gridColStart} / ${col.gridColStart + col.laneCount}` }}
                    >
                        {col.name}
                        {onLeaveByColumn[col.staffId] && (
                            <span className="calendar-header-leave-badge">On Leave</span>
                        )}
                    </div>
                ))}

                {/* time labels (sticky to left while scrolling) */}
                {lines.slice(0, intervals).map((t, i) => (
                    <div key={t} className="calendar-time-label" style={{ gridRow: i + 2 }}>
                        {t}
                    </div>
                ))}

                {/* background cells (clickable to add) — one per staff (spanning all its lanes) per time row */}
                {columnLayout.map(col => {
                    const onLeave = onLeaveByColumn[col.staffId] ?? false;
                    return lines.slice(0, intervals).map((t, i) => {
                        const cellEnd = lines[i + 1] ?? rangeEnd;
                        const open = isCellOpen(openHoursByColumn[col.staffId] ?? [], t, cellEnd);
                        return (
                            <div
                                key={`${col.staffId}-${t}`}
                                className={`calendar-bg-cell${onLeave ? " calendar-bg-cell-leave" : open ? "" : " calendar-bg-cell-closed"}`}
                                onClick={onLeave ? undefined : () => onAddSlot({ staffId: col.staffId === NONE_COL ? "" : col.staffId, startTime: t, endTime: cellEnd })}
                                style={{
                                    gridColumn: `${col.gridColStart} / ${col.gridColStart + col.laneCount}`, gridRow: i + 2,
                                    cursor: onLeave ? "not-allowed" : undefined,
                                }}
                            />
                        );
                    });
                })}

                {/* One positioned wrapper per column, spanning its whole time area — slot
                    blocks inside are absolutely positioned (top/height by exact minute,
                    left/width by lane) relative to THIS wrapper, not the page. CSS Grid's
                    "grid area as containing block for abspos items" isn't reliably applied
                    here, so positioning is scoped the ordinary way: a `position: relative`
                    ancestor per column. */}
                {columnLayout.map(col => {
                    const colSlots = slots.filter(slot => (slot.staff ? slot.staff.staffId : NONE_COL) === col.staffId);
                    return (
                        <div
                            key={`slots-${col.staffId}`}
                            className="calendar-col-slots"
                            style={{ gridColumn: `${col.gridColStart} / ${col.gridColStart + col.laneCount}`, gridRow: "2 / -1" }}
                        >
                            {colSlots.map(slot => {
                                const laneIdx = col.lane.get(slot.serviceSlotId) ?? 0;
                                const start = extractTime(slot.startTime);
                                const end = extractTime(slot.endTime);
                                const top = Math.max(0, (toMinutes(start) - rangeStartMinutes) / 30 * ROW_HEIGHT);
                                const height = Math.min(
                                    Math.max(4, (toMinutes(end) - toMinutes(start)) / 30 * ROW_HEIGHT),
                                    totalHeight - top,
                                );
                                const pkg = slot.serviceSlotOptions[0];
                                const label = pkg
                                    ? `${pkg.serviceOption.service.serviceName}`
                                    : "Slot";
                                return (
                                    <div
                                        key={slot.serviceSlotId}
                                        className={`calendar-slot-block ${slotStatusClass(bookingBySlot[slot.serviceSlotId])}`}
                                        onClick={() => onManageSlot(slot)}
                                        style={{
                                            position: "absolute",
                                            top,
                                            height,
                                            left: `${(laneIdx / col.laneCount) * 100}%`,
                                            width: `${(1 / col.laneCount) * 100}%`,
                                        }}
                                    >
                                        <svg
                                            className="calendar-slot-edit-icon"
                                            viewBox="0 0 16 16"
                                            fill="currentColor"
                                            aria-hidden="true"
                                        >
                                            <path d="M12.146.146a.5.5 0 0 1 .708 0l3 3a.5.5 0 0 1 0 .708l-10 10a.5.5 0 0 1-.168.11l-5 2a.5.5 0 0 1-.65-.65l2-5a.5.5 0 0 1 .11-.168zM11.207 2.5 13.5 4.793 14.793 3.5 12.5 1.207zM12.793 5.5 10.5 3.207 4 9.707V10h.5a.5.5 0 0 1 .5.5v.5h.5a.5.5 0 0 1 .5.5v.5h.293zm-9.761 5.175-.106.106-1.528 3.821 3.821-1.528.106-.106A.5.5 0 0 1 5 12.5V12h-.5a.5.5 0 0 1-.5-.5V11h-.5a.5.5 0 0 1-.468-.325" />
                                        </svg>
                                        <div className="calendar-slot-time">{start}–{end}</div>
                                        <div>{label}</div>
                                        {slot.serviceSlotOptions.length > 1 && (
                                            <div className="calendar-slot-more">+{slot.serviceSlotOptions.length - 1} more</div>
                                        )}
                                    </div>
                                );
                            })}
                        </div>
                    );
                })}
            </div>
        </div>
    );
}
