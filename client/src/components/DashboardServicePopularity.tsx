import { Card } from "react-bootstrap";
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import type { ServicePopularity } from "../services/AnalyticsService";

export default function DashboardServicePopularity({ items }: { items: ServicePopularity[] }) {
    // Longest bar first reads oddly top-to-bottom for a horizontal chart —
    // reverse so it renders largest-at-top.
    const data = [...items].reverse();

    return (
        <Card className="dashboard-card">
            <Card.Body>
                <Card.Title>💼 Service Popularity</Card.Title>
                {items.length === 0 ? (
                    <div className="dashboard-empty">No bookings in this range.</div>
                ) : (
                    <ResponsiveContainer width="100%" height={Math.max(180, data.length * 44)}>
                        <BarChart data={data} layout="vertical" margin={{ top: 4, right: 24, left: 0, bottom: 0 }}>
                            <CartesianGrid strokeDasharray="3 3" stroke="var(--chart-grid)" horizontal={false} />
                            <XAxis
                                type="number"
                                allowDecimals={false}
                                tick={{ fontSize: 12, fill: "var(--chart-muted)" }}
                                axisLine={{ stroke: "var(--chart-axis)" }}
                                tickLine={false}
                            />
                            <YAxis
                                type="category"
                                dataKey="serviceName"
                                width={120}
                                tick={{ fontSize: 12, fill: "var(--chart-text-secondary)" }}
                                axisLine={false}
                                tickLine={false}
                            />
                            <Tooltip
                                formatter={value => [String(value), "Bookings"]}
                                contentStyle={{ background: "var(--chart-surface)", border: "1px solid var(--chart-grid)", borderRadius: 8, fontSize: 13 }}
                            />
                            <Bar dataKey="bookingCount" fill="var(--series-1)" radius={[0, 4, 4, 0]} maxBarSize={22} />
                        </BarChart>
                    </ResponsiveContainer>
                )}
            </Card.Body>
        </Card>
    );
}
