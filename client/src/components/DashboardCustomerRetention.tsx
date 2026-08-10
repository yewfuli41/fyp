import { Card } from "react-bootstrap";
import { Cell, Legend, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";
import type { CustomerRetention } from "../services/AnalyticsService";

const COLORS = { New: "var(--series-1)", Returning: "var(--series-3)" };

export default function DashboardCustomerRetention({ retention }: { retention: CustomerRetention | null }) {
    if (!retention) return null;

    const data = [
        { name: "New", value: retention.newCustomers },
        { name: "Returning", value: retention.returningCustomers },
    ];

    return (
        <Card className="dashboard-card">
            <Card.Body>
                <Card.Title>👥 New vs Returning Customers</Card.Title>
                {retention.totalCustomers === 0 ? (
                    <div className="dashboard-empty">No customer bookings in this range.</div>
                ) : (
                    <>
                        <ResponsiveContainer width="100%" height={220}>
                            <PieChart>
                                <Pie
                                    data={data}
                                    dataKey="value"
                                    nameKey="name"
                                    innerRadius={55}
                                    outerRadius={85}
                                    paddingAngle={2}
                                >
                                    {data.map(d => <Cell key={d.name} fill={COLORS[d.name as keyof typeof COLORS]} />)}
                                </Pie>
                                <Tooltip
                                    formatter={(value, name) => [String(value), String(name)]}
                                    contentStyle={{ background: "var(--chart-surface)", border: "1px solid var(--chart-grid)", borderRadius: 8, fontSize: 13 }}
                                />
                                <Legend iconType="circle" wrapperStyle={{ fontSize: 13 }} />
                            </PieChart>
                        </ResponsiveContainer>
                        <div className="text-center text-muted small mt-2">
                            {retention.totalCustomers} unique customer{retention.totalCustomers === 1 ? "" : "s"} in this range
                        </div>
                    </>
                )}
            </Card.Body>
        </Card>
    );
}
