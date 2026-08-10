import { Card, Table } from "react-bootstrap";
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import type { StaffUtilization } from "../services/AnalyticsService";

export default function DashboardStaffUtilization({ items }: { items: StaffUtilization[] }) {
    const data = items.map(s => ({ ...s, hoursBookedRounded: Math.round(s.hoursBooked * 10) / 10 }));

    // Axis ticks in fixed 0.5h steps, up to the next half-hour above the
    // tallest bar (so the last tick always covers the max value).
    const maxHours = Math.max(0, ...data.map(d => d.hoursBookedRounded));
    const axisMax = Math.max(0.5, Math.ceil(maxHours * 2) / 2);
    const hourTicks: number[] = [];
    for (let h = 0; h <= axisMax + 1e-9; h += 0.5) hourTicks.push(Math.round(h * 10) / 10);

    return (
        <Card className="dashboard-card">
            <Card.Body>
                <Card.Title>👨‍💼 Staff Utilization</Card.Title>
                {data.length === 0 ? (
                    <div className="dashboard-empty">No slots in this range.</div>
                ) : (
                    <>
                        <ResponsiveContainer width="100%" height={Math.max(180, data.length * 44)}>
                            <BarChart data={data} layout="vertical" margin={{ top: 4, right: 24, left: 0, bottom: 0 }}>
                                <CartesianGrid strokeDasharray="3 3" stroke="var(--chart-grid)" horizontal={false} />
                                <XAxis
                                    type="number"
                                    unit=" h"
                                    domain={[0, axisMax]}
                                    ticks={hourTicks}
                                    tick={{ fontSize: 12, fill: "var(--chart-muted)" }}
                                    axisLine={{ stroke: "var(--chart-axis)" }}
                                    tickLine={false}
                                />
                                <YAxis
                                    type="category"
                                    dataKey="staffName"
                                    width={100}
                                    tick={{ fontSize: 12, fill: "var(--chart-text-secondary)" }}
                                    axisLine={false}
                                    tickLine={false}
                                />
                                <Tooltip
                                    formatter={(value, _name, item) => {
                                        // eslint-disable-next-line @typescript-eslint/no-explicit-any
                                        const payload = (item as any)?.payload as { bookings: number } | undefined;
                                        const detail = payload ? ` (${payload.bookings} bookings)` : "";
                                        return [`${value} h${detail}`, "Hours booked"];
                                    }}
                                    contentStyle={{ background: "var(--chart-surface)", border: "1px solid var(--chart-grid)", borderRadius: 8, fontSize: 13 }}
                                />
                                <Bar dataKey="hoursBookedRounded" fill="var(--series-3)" radius={[0, 4, 4, 0]} maxBarSize={22} />
                            </BarChart>
                        </ResponsiveContainer>

                        <Table size="sm" borderless className="mb-0 mt-2">
                            <thead>
                                <tr className="text-muted small">
                                    <th>Staff</th>
                                    <th className="text-end">Bookings</th>
                                    <th className="text-end">Hours booked</th>
                                </tr>
                            </thead>
                            <tbody>
                                {data.map(s => (
                                    <tr key={s.staffId ?? s.staffName}>
                                        <td>{s.staffName}</td>
                                        <td className="text-end">{s.bookings}</td>
                                        <td className="text-end">{s.hoursBooked.toFixed(1)} h</td>
                                    </tr>
                                ))}
                            </tbody>
                        </Table>
                    </>
                )}
            </Card.Body>
        </Card>
    );
}
