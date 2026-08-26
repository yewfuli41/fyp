import { Card } from "react-bootstrap";
import { Cell, Legend, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";
import type { BookingSummary } from "../services/AnalyticsService";

const TYPE_COLOR: Record<string, string> = {
    Online: "var(--series-1)",
    "Walk-in": "var(--series-2)",
};

export default function DashboardBookingTypeSplit({ summary }: { summary: BookingSummary | null }) {
    if (!summary) return null;

    const online = summary.byType.find(t => t.bookingType === "ONLINE")?.count ?? 0;
    const walkIn = summary.byType.find(t => t.bookingType === "WALK_IN")?.count ?? 0;
    const typeData = [{ name: "Online", value: online }, { name: "Walk-in", value: walkIn }];

    return (
        <Card className="dashboard-card">
            <Card.Body>
                <Card.Title>Online vs Walk-in</Card.Title>
                <div className="dashboard-stat-label mb-1">(range)</div>
                {online + walkIn === 0 ? (
                    <div className="dashboard-empty">No bookings in this range.</div>
                ) : (
                    <ResponsiveContainer width="100%" height={140}>
                        <PieChart>
                            <Pie data={typeData} dataKey="value" nameKey="name" innerRadius={32} outerRadius={50} paddingAngle={2}>
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
            </Card.Body>
        </Card>
    );
}
