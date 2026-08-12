import { Alert, Button, ButtonGroup, Card, Col, Row, Spinner } from "react-bootstrap";
import type { SlotUtilization, SlotUtilizationGroupBy } from "../services/AnalyticsService";

const GROUP_BY_OPTIONS: [SlotUtilizationGroupBy, string][] = [
    ["weekday", "Weekdays"],
    ["date", "Dates"],
    ["month", "Months"],
    ["year", "Years"],
];

const WEEKDAY_LABEL: Record<string, string> = {
    monday: "Mon", tuesday: "Tue", wednesday: "Wed", thursday: "Thu", friday: "Fri", saturday: "Sat", sunday: "Sun",
};

const MONTH_NAME = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

// Backend point labels are a lowercase weekday name, a zero-padded
// day-of-month ("15"), a zero-padded month ("07"), or a year ("2026") —
// reformat into something compact enough for a small cell.
const pointLabel = (label: string, groupBy: SlotUtilizationGroupBy): string => {
    if (groupBy === "weekday") return WEEKDAY_LABEL[label] ?? label;
    if (groupBy === "month") return MONTH_NAME[Number(label) - 1] ?? label;
    if (groupBy === "date") return String(Number(label)); // "05" -> "5"
    return label; // year
};

// The full, unambiguous form of a point, for the cell's hover tooltip —
// combines the section's subtitle back in, since the point label alone
// ("15") doesn't say which month.
const pointTooltipLabel = (subtitle: string, label: string, groupBy: SlotUtilizationGroupBy): string => {
    if (groupBy === "date") return `${subtitle}-${label}`;
    if (groupBy === "month") return `${MONTH_NAME[Number(label) - 1] ?? label} ${subtitle}`;
    return pointLabel(label, groupBy);
};

// "date" grouping's subtitle is "YYYY-MM" -> "YYYY Mon"; "month" grouping's
// is already just the bare year, shown as-is.
const sectionSubtitle = (subtitle: string, groupBy: SlotUtilizationGroupBy): string => {
    if (!subtitle || groupBy !== "date") return subtitle;
    const [y, m] = subtitle.split("-").map(Number);
    return `${y} ${MONTH_NAME[m - 1] ?? m}`;
};

// Sequential blue ramp, light→dark — one hue, magnitude only (see dataviz
// skill's color-formula: sequential = one hue, never a rainbow).
const SEQ_STEPS = ["var(--seq-100)", "var(--seq-250)", "var(--seq-400)", "var(--seq-550)", "var(--seq-700)"];

const cellColor = (rate: number | null): string => {
    if (rate === null) return "var(--chart-grid)"; // no slots that bucket — neutral, not zero-utilization
    const idx = Math.min(SEQ_STEPS.length - 1, Math.floor(rate * SEQ_STEPS.length));
    return SEQ_STEPS[idx];
};

// One legend swatch per SEQ_STEPS bucket, labelled by the % range it covers
// (buckets are equal-width, so this stays in sync if SEQ_STEPS changes size),
// plus the "no slots scheduled" neutral swatch.
const bucketSize = 100 / SEQ_STEPS.length;
const LEGEND_ITEMS = [
    ...SEQ_STEPS.map((color, i) => ({
        color,
        label: `${Math.round(i * bucketSize)}–${Math.round((i + 1) * bucketSize)}%`,
    })),
    { color: "var(--chart-grid)", label: "No slots" },
];

interface Props {
    utilization: SlotUtilization | null;
    isLoading: boolean;
    // Distinct from "still loading" — without this, a failed fetch
    // (utilization stays null but isLoading has already gone false) had no
    // way to escape the empty-state branch below and just showed nothing.
    error: string;
    groupBy: SlotUtilizationGroupBy;
    onGroupByChange: (groupBy: SlotUtilizationGroupBy) => void;
}

export default function DashboardSlotUtilization({ utilization, isLoading, error, groupBy, onGroupByChange }: Props) {
    const sections = utilization?.sections ?? [];

    return (
        <Card className="dashboard-card">
            <Card.Body>
                <div className="d-flex flex-wrap justify-content-between align-items-center gap-2 mb-1">
                    <Card.Title className="mb-0">🕒 Slot Utilization</Card.Title>
                    <ButtonGroup size="sm">
                        {GROUP_BY_OPTIONS.map(([key, label]) => (
                            <Button
                                key={key}
                                variant={groupBy === key ? "primary" : "outline-secondary"}
                                onClick={() => onGroupByChange(key)}
                            >
                                {label}
                            </Button>
                        ))}
                    </ButtonGroup>
                </div>

                {isLoading ? (
                    <div className="text-center py-5"><Spinner size="sm" /></div>
                ) : error ? (
                    <Alert variant="danger" className="py-2 mb-0">{error}</Alert>
                ) : !utilization || utilization.totalSlots === 0 ? (
                    <div className="dashboard-empty">No slots scheduled in this range.</div>
                ) : (
                    <>
                        <Row className="g-3 mb-3 mt-1">
                            <Col xs={4}>
                                <div className="dashboard-stat-tile">
                                    <div className="dashboard-stat-value">{(utilization.utilizationRate * 100).toFixed(0)}%</div>
                                    <div className="dashboard-stat-label">Utilization rate</div>
                                </div>
                            </Col>
                            <Col xs={4}>
                                <div className="dashboard-stat-tile">
                                    <div className="dashboard-stat-value">{utilization.bookedSlots}</div>
                                    <div className="dashboard-stat-label">Booked slots</div>
                                </div>
                            </Col>
                            <Col xs={4}>
                                <div className="dashboard-stat-tile">
                                    <div className="dashboard-stat-value">{utilization.totalSlots}</div>
                                    <div className="dashboard-stat-label">Total slots</div>
                                </div>
                            </Col>
                        </Row>

                        <div className="dashboard-heatmap-sections">
                            {sections.map(section => (
                                <div key={section.subtitle || "_"}>
                                    {section.subtitle && (
                                        <div className="dashboard-subheading">{sectionSubtitle(section.subtitle, groupBy)}</div>
                                    )}
                                    <div className="dashboard-heatmap-strip">
                                        {section.points.map(p => {
                                            const rate = p.totalSlots > 0 ? p.bookedSlots / p.totalSlots : null;
                                            return (
                                                <div key={p.label} className="dashboard-heatmap-strip-item">
                                                    <div
                                                        className="dashboard-heatmap-cell"
                                                        style={{ background: cellColor(rate) }}
                                                        title={`${pointTooltipLabel(section.subtitle, p.label, groupBy)} — ${p.bookedSlots}/${p.totalSlots} booked`}
                                                    />
                                                    <div className="dashboard-heatmap-strip-label">{pointLabel(p.label, groupBy)}</div>
                                                </div>
                                            );
                                        })}
                                    </div>
                                </div>
                            ))}
                        </div>

                        <div className="d-flex flex-wrap align-items-center gap-3 mt-3">
                            <span className="dashboard-stat-label mb-0">Slot Booking Rate</span>
                            <div className="d-flex align-items-center gap-2">
                                {LEGEND_ITEMS.map(item => (
                                    <span key={item.label} className="d-flex align-items-center gap-1">
                                        <span className="dashboard-heatmap-legend-swatch" style={{ background: item.color }} />
                                        <span className="text-muted small">{item.label}</span>
                                    </span>
                                ))}
                            </div>
                        </div>
                    </>
                )}
            </Card.Body>
        </Card>
    );
}
