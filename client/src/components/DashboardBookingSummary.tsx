import { Card, Col, Row } from "react-bootstrap";
import type { BookingSummary, CancellationAnalysis } from "../services/AnalyticsService";

export default function DashboardBookingSummary(
    { summary, cancellation }: { summary: BookingSummary | null; cancellation: CancellationAnalysis | null },
) {
    if (!summary) return null;

    return (
        <Card className="dashboard-card">
            <Card.Body>
                <Card.Title>📊 Booking Summary</Card.Title>
                <Row className="g-3 mt-1">
                    <Col xs={6} md={3}>
                        <div className="dashboard-stat-tile">
                            <div className="dashboard-stat-value">{summary.todayAcceptedCount}</div>
                            <div className="dashboard-stat-label">Accepted Bookings</div>
                            <div className="dashboard-stat-label">(Today)</div>
                        </div>
                    </Col>
                    <Col xs={6} md={3}>
                        <div className="dashboard-stat-tile">
                            <div className="dashboard-stat-value">{summary.pendingCount}</div>
                            <div className="dashboard-stat-label">Pending</div>
                            <div className="dashboard-stat-label">Bookings</div>
                        </div>
                    </Col>
                    <Col xs={6} md={3}>
                        <div className="dashboard-stat-tile">
                            <div className="dashboard-stat-value">
                                {((cancellation?.cancelledOrRejectedRate ?? 0) * 100).toFixed(1)}%
                            </div>
                            <div className="dashboard-stat-label">Cancelled & rejected rate</div>
                        </div>
                    </Col>
                    <Col xs={6} md={3}>
                        <div className="dashboard-stat-tile">
                            <div className="dashboard-stat-value">{summary.totalBookings}</div>
                            <div className="dashboard-stat-label">Total</div>
                        </div>
                    </Col>
                </Row>
            </Card.Body>
        </Card>
    );
}
