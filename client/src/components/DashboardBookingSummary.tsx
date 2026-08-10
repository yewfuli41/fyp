import { useState } from "react";
import { Card, Col, Form, Row } from "react-bootstrap";
import { Cell, Legend, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";
import type { BookingSummary } from "../services/AnalyticsService";

const STATUS_COLOR: Record<string, string> = {
    ACCEPTED: "var(--series-3)",
    PENDING: "var(--series-4)",
    RESCHEDULED: "var(--series-1)",
    CANCELLED: "var(--series-8)",
    REJECTED: "var(--series-2)",
    PAST: "var(--chart-muted)",
};

const TYPE_COLOR: Record<string, string> = {
    Online: "var(--series-1)",
    "Walk-in": "var(--series-2)",
};

const cap = (s: string) => s.charAt(0) + s.slice(1).toLowerCase();

export default function DashboardBookingSummary({ summary }: { summary: BookingSummary | null }) {
    const [statusFilter, setStatusFilter] = useState("");

    if (!summary) return null;

    const online = summary.byType.find(t => t.bookingType === "ONLINE")?.count ?? 0;
    const walkIn = summary.byType.find(t => t.bookingType === "WALK_IN")?.count ?? 0;
    const typeData = [{ name: "Online", value: online }, { name: "Walk-in", value: walkIn }];

    // "Total bookings" narrows to whichever status is picked below — no extra
    // fetch needed, byStatus already has every status's count for this range.
    const filteredCount = statusFilter
        ? summary.byStatus.find(sc => sc.status === statusFilter)?.count ?? 0
        : summary.totalBookings;

    return (
        <Card className="dashboard-card">
            <Card.Body>
                <Card.Title>📊 Booking Summary</Card.Title>
                <Row className="g-4 mt-1">
                    <Col xs={12} lg={7}>
                        <Row className="g-3">
                            <Col xs={6} md={4}>
                                <div className="dashboard-stat-tile">
                                    <div className="dashboard-stat-value">{summary.todayAcceptedCount}</div>
                                    <div className="dashboard-stat-label">Accepted Bookings</div>
                                    <div className="dashboard-stat-label">(Today)</div>
                                </div>
                            </Col>
                            <Col xs={6} md={4}>
                                <div className="dashboard-stat-tile">
                                    <div className="dashboard-stat-value">{summary.pendingCount}</div>
                                    <div className="dashboard-stat-label">Pending Bookings</div>
                                </div>
                            </Col>
                            <Col xs={12} md={4}>
                                <div className="dashboard-stat-tile">
                                    <div className="dashboard-stat-value">{filteredCount}</div>
                                    <div className="dashboard-stat-label">
                                        {statusFilter ? `${cap(statusFilter)}` : "Total"}
                                    </div>
                                    <Form.Select
                                        size="sm"
                                        className="mt-2"
                                        value={statusFilter}
                                        onChange={e => setStatusFilter(e.target.value)}
                                    >
                                        <option value="">All statuses</option>
                                        {summary.byStatus.map(sc => (
                                            <option key={sc.status} value={sc.status}>{cap(sc.status)} · {sc.count}</option>
                                        ))}
                                    </Form.Select>
                                </div>
                            </Col>
                        </Row>

                        {summary.byStatus.length > 0 && (
                            <div className="d-flex flex-wrap gap-2 mt-3">
                                {summary.byStatus.map(sc => (
                                    <span key={sc.status} className="dashboard-status-pill">
                                        <span
                                            className="dashboard-status-dot"
                                            style={{ background: STATUS_COLOR[sc.status] ?? "var(--chart-muted)" }}
                                        />
                                        {cap(sc.status)} · {sc.count}
                                    </span>
                                ))}
                            </div>
                        )}
                    </Col>

                    <Col xs={12} lg={5}>
                        <div className="dashboard-stat-label mb-1">Online vs walk-in (range)</div>
                        {online + walkIn === 0 ? (
                            <div className="dashboard-empty">No bookings in this range.</div>
                        ) : (
                            <ResponsiveContainer width="100%" height={190}>
                                <PieChart>
                                    <Pie data={typeData} dataKey="value" nameKey="name" innerRadius={45} outerRadius={70} paddingAngle={2}>
                                        {typeData.map(d => <Cell key={d.name} fill={TYPE_COLOR[d.name]} />)}
                                    </Pie>
                                    <Tooltip
                                        formatter={(value, name) => [String(value), String(name)]}
                                        contentStyle={{ background: "var(--chart-surface)", border: "1px solid var(--chart-grid)", borderRadius: 8, fontSize: 13 }}
                                    />
                                    <Legend
                                        iconType="circle"
                                        wrapperStyle={{ fontSize: 12 }}
                                        formatter={(value: string) => {
                                            const item = typeData.find(d => d.name === value);
                                            return `${value} (${item?.value ?? 0})`;
                                        }}
                                    />
                                </PieChart>
                            </ResponsiveContainer>
                        )}
                    </Col>
                </Row>
            </Card.Body>
        </Card>
    );
}
