import { Card } from "react-bootstrap";
import { CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import type { BookingTrendPoint } from "../services/AnalyticsService";

const formatDate = (iso: string) => {
    const d = new Date(`${iso}T00:00:00`);
    return d.toLocaleDateString("en-GB", { day: "2-digit", month: "short" });
};

export default function DashboardBookingTrend({ points }: { points: BookingTrendPoint[] }) {
    return (
        <Card className="dashboard-card">
            <Card.Body>
                <Card.Title>📈 Booking Trend</Card.Title>
                {points.length === 0 ? (
                    <div className="dashboard-empty">No bookings in this range.</div>
                ) : (
                    <ResponsiveContainer width="100%" height={260}>
                        <LineChart data={points} margin={{ top: 12, right: 12, left: 0, bottom: 0 }}>
                            <CartesianGrid strokeDasharray="3 3" stroke="var(--chart-grid)" vertical={false} />
                            <XAxis
                                dataKey="date"
                                tickFormatter={formatDate}
                                tick={{ fontSize: 12, fill: "var(--chart-muted)" }}
                                axisLine={{ stroke: "var(--chart-axis)" }}
                                tickLine={false}
                            />
                            <YAxis
                                allowDecimals={false}
                                tick={{ fontSize: 12, fill: "var(--chart-muted)" }}
                                axisLine={false}
                                tickLine={false}
                                width={28}
                            />
                            <Tooltip
                                labelFormatter={label => formatDate(String(label))}
                                formatter={value => [String(value), "Bookings"]}
                                contentStyle={{ background: "var(--chart-surface)", border: "1px solid var(--chart-grid)", borderRadius: 8, fontSize: 13 }}
                            />
                            <Line
                                type="monotone"
                                dataKey="count"
                                stroke="var(--series-1)"
                                strokeWidth={2}
                                dot={{ r: 3, fill: "var(--series-1)" }}
                                activeDot={{ r: 5 }}
                            />
                        </LineChart>
                    </ResponsiveContainer>
                )}
            </Card.Body>
        </Card>
    );
}
