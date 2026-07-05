import { useMemo } from "react";
import { Alert } from "react-bootstrap";
import { extractTime, type ServiceSlot, type ServiceSlotInput } from "../services/ServiceSlotService";
import { NONE_COL } from "../utils/serviceSlotHelpers";

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
    onAddSlot: (prefill: Partial<ServiceSlotInput>) => void;
    onManageSlot: (slot: ServiceSlot) => void;
}

// Renders the CSS-grid calendar (staff headers, time labels, clickable
// background cells, and slot blocks) — including the lane-assignment layout
// math for overlapping slots. Filtering, fetching, and API calls live in CalendarPage.
export default function CalendarGrid({
    staffColumns, lines, intervals, rangeEnd, slots, onAddSlot, onManageSlot,
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
                    </div>
                ))}

                {/* time labels (sticky to left while scrolling) */}
                {lines.slice(0, intervals).map((t, i) => (
                    <div key={t} className="calendar-time-label" style={{ gridRow: i + 2 }}>
                        {t}
                    </div>
                ))}

                {/* background cells (clickable to add) — one per staff (spanning all its lanes) per time row */}
                {columnLayout.map(col =>
                    lines.slice(0, intervals).map((t, i) => (
                        <div
                            key={`${col.staffId}-${t}`}
                            className="calendar-bg-cell"
                            onClick={() => onAddSlot({ staffId: col.staffId === NONE_COL ? "" : col.staffId, startTime: t, endTime: lines[i + 1] ?? rangeEnd })}
                            style={{
                                gridColumn: `${col.gridColStart} / ${col.gridColStart + col.laneCount}`, gridRow: i + 2,
                            }}
                        />
                    ))
                )}

                {/* slot blocks — overlapping slots for the same staff sit in separate lanes, side by side */}
                {slots.map(slot => {
                    const colId = slot.staff ? slot.staff.staffId : NONE_COL;
                    const col = columnLayout.find(c => c.staffId === colId);
                    if (!col) return null;
                    const laneIdx = col.lane.get(slot.serviceSlotId) ?? 0;
                    const start = extractTime(slot.startTime);
                    const end = extractTime(slot.endTime);
                    let startLine = lines.indexOf(start);
                    let endLine = lines.indexOf(end);
                    if (startLine < 0) startLine = 0;
                    if (endLine < 0) endLine = lines.length - 1;
                    const pkg = slot.serviceSlotPackages[0];
                    const label = pkg
                        ? `${pkg.servicePackage.service.serviceName} — ${pkg.servicePackage.servicePackageName}`
                        : "Slot";
                    return (
                        <div
                            key={slot.serviceSlotId}
                            className="calendar-slot-block"
                            onClick={() => onManageSlot(slot)}
                            style={{
                                gridColumn: col.gridColStart + laneIdx,
                                gridRow: `${startLine + 2} / ${endLine + 2}`,
                            }}
                        >
                            <div className="calendar-slot-time">{start}–{end}</div>
                            <div>{label}</div>
                            {slot.serviceSlotPackages.length > 1 && (
                                <div className="calendar-slot-more">+{slot.serviceSlotPackages.length - 1} more</div>
                            )}
                        </div>
                    );
                })}
            </div>
        </div>
    );
}
