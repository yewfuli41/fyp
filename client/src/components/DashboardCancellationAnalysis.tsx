import { Alert, Card, Col, Row, Spinner } from "react-bootstrap";
import {
    Bar, BarChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis,
} from "recharts";
import type { CancellationAnalysis, ServiceFilterOption } from "../services/AnalyticsService";
import type { Staff } from "../services/StaffService";
import MultiSelectDropdown from "./MultiSelectDropdown";

interface Props {
    analysis: CancellationAnalysis | null;
    isLoading: boolean;
    // Distinct from "still loading" — without this, a failed fetch (analysis
    // stays null but isLoading has already gone false) had no way to escape
    // the spinner branch below and just spun forever.
    error: string;
    staffList: Staff[];
    // Includes soft-deleted services, so past cancellations/rejections under
    // a since-deleted service stay filterable.
    serviceList: ServiceFilterOption[];
    staffIds: string[];
    // Selected service *names*, not ids — a name can span multiple
    // serviceIds (see ServiceFilterOption), so the caller expands the
    // selected names back into the full id list before querying.
    serviceNames: string[];
    onStaffChange: (staffIds: string[]) => void;
    onServiceNamesChange: (serviceNames: string[]) => void;
}

export default function DashboardCancellationAnalysis({
    analysis, isLoading, error, staffList, serviceList, staffIds, serviceNames, onStaffChange, onServiceNamesChange,
}: Props) {
    const byService = analysis ? [...analysis.byService].reverse() : [];
    // decided-by split, one bar per staff (or "Owner-managed") — who's
    // behind the cancelled/rejected bookings, not just how many.
    const byStaff = analysis?.byStaff ?? [];

    // Headline totals for the same customer-vs-staff split the By staff
    // chart breaks down — byStaff covers every cancellation/rejection in the
    // range, so summing its two columns gives each side's total.
    const customerTotal = byStaff.reduce((sum, s) => sum + s.customerCount, 0);
    const staffTotal = byStaff.reduce((sum, s) => sum + s.staffCount, 0);

    return (
        <Card className="dashboard-card">
            <Card.Body>
                <div className="d-flex flex-wrap justify-content-between align-items-start gap-2 mb-1">
                    <Card.Title className="mb-0">❌ Cancellation Analysis</Card.Title>
                    <div className="d-flex flex-wrap gap-2">
                        <MultiSelectDropdown
                            label="Staff"
                            selected={staffIds}
                            onChange={onStaffChange}
                            options={staffList.map(s => ({ value: s.staffId, label: s.name }))}
                        />
                        <MultiSelectDropdown
                            label="Service"
                            selected={serviceNames}
                            onChange={onServiceNamesChange}
                            options={serviceList.map(s => ({
                                value: s.serviceName,
                                label: s.deleted ? `${s.serviceName} (deleted)` : s.serviceName,
                            }))}
                        />
                    </div>
                </div>

                {isLoading ? (
                    <div className="text-center py-5"><Spinner size="sm" /></div>
                ) : error ? (
                    <Alert variant="danger" className="py-2 mb-0">{error}</Alert>
                ) : !analysis ? (
                    <div className="dashboard-empty">No data for this range.</div>
                ) : (
                    <>
                        <Row className="g-3 mb-3 mt-1">
                            <Col xs={6}>
                                <div className="dashboard-stat-tile">
                                    <div className="dashboard-stat-value">{customerTotal}</div>
                                    <div className="dashboard-stat-label">Cancelled by customer</div>
                                </div>
                            </Col>
                            <Col xs={6}>
                                <div className="dashboard-stat-tile">
                                    <div className="dashboard-stat-value">{staffTotal}</div>
                                    <div className="dashboard-stat-label">Cancelled/rejected by staff</div>
                                </div>
                            </Col>
                        </Row>

                        <div className="dashboard-subheading">By staff</div>
                        {byStaff.length === 0 ? (
                            <div className="dashboard-empty">No cancellations or rejections in this range.</div>
                        ) : (
                            <ResponsiveContainer width="100%" height={Math.max(140, byStaff.length * 40)}>
                                <BarChart data={byStaff} layout="vertical" margin={{ top: 4, right: 24, left: 0, bottom: 0 }}>
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
                                        dataKey="staffName"
                                        width={120}
                                        tick={{ fontSize: 12, fill: "var(--chart-text-secondary)" }}
                                        axisLine={false}
                                        tickLine={false}
                                    />
                                    <Tooltip
                                        contentStyle={{ background: "var(--chart-surface)", border: "1px solid var(--chart-grid)", borderRadius: 8, fontSize: 13 }}
                                    />
                                    <Legend wrapperStyle={{ fontSize: 12 }} />
                                    <Bar
                                        dataKey="customerCount" name="Cancelled by customer" stackId="decided"
                                        fill="var(--series-1)" maxBarSize={20}
                                    />
                                    <Bar
                                        dataKey="staffCount" name="Cancelled/rejected by staff" stackId="decided"
                                        fill="var(--series-8)" radius={[0, 4, 4, 0]} maxBarSize={20}
                                    />
                                </BarChart>
                            </ResponsiveContainer>
                        )}

                        <div className="dashboard-subheading mt-4">By service</div>
                        {byService.length === 0 ? (
                            <div className="dashboard-empty">No cancellations or rejections in this range.</div>
                        ) : (
                            <ResponsiveContainer width="100%" height={Math.max(140, byService.length * 40)}>
                                <BarChart data={byService} layout="vertical" margin={{ top: 4, right: 24, left: 0, bottom: 0 }}>
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
                                        formatter={value => [String(value), "Cancelled + rejected"]}
                                        contentStyle={{ background: "var(--chart-surface)", border: "1px solid var(--chart-grid)", borderRadius: 8, fontSize: 13 }}
                                    />
                                    <Bar dataKey="count" fill="var(--series-8)" radius={[0, 4, 4, 0]} maxBarSize={20} />
                                </BarChart>
                            </ResponsiveContainer>
                        )}
                    </>
                )}
            </Card.Body>
        </Card>
    );
}
