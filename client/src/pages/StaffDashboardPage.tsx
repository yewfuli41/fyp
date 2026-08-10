import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Badge, Button, Card, Col, Container, Form, Row, Spinner } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { getBusinessBookings, type BookingDetail } from "../services/BookingService";
import { getBusinessServices, type Service } from "../services/ServiceService";
import { myLeaveApplications } from "../services/LeaveService";
import { extractTime } from "../services/ServiceSlotService";
import { todayISO } from "../utils/serviceSlotHelpers";
import { IconCalendar, IconClock } from "../components/icons";
import "../styles/BusinessDashboard.css";
import "../styles/BookingRequestsPanel.css";

const formatDate = (iso: string) => {
    const d = new Date(`${iso}T00:00:00`);
    return isNaN(d.getTime()) ? iso : d.toLocaleDateString("en-GB", { day: "numeric", month: "short", year: "numeric" });
};

const to12h = (rfc: string) => {
    const [h, m] = extractTime(rfc).split(":").map(Number);
    const period = h < 12 ? "am" : "pm";
    const hour = h % 12 === 0 ? 12 : h % 12;
    return `${hour}.${String(m).padStart(2, "0")} ${period}`;
};

const initials = (name: string) =>
    name.split(/\s+/).filter(Boolean).slice(0, 2).map(w => w[0]?.toUpperCase() ?? "").join("") || "?";

// A primary nav-shortcut button with a notification badge pinned to its
// corner — used for the two staff-only shortcuts (My Leave / My Booking),
// kept visually distinct from the read-only stat tiles below.
function ShortcutButton({ label, icon, badge, onClick }: { label: string; icon: ReactNode; badge: number; onClick: () => void }) {
    return (
        <span className="position-relative">
            <Button variant="primary" onClick={onClick} className="d-inline-flex align-items-center gap-2">
                {icon}{label}
            </Button>
            {badge > 0 && (
                <Badge bg="danger" pill className="position-absolute top-0 start-100 translate-middle" style={{ fontSize: "0.7rem" }}>
                    {badge}
                </Badge>
            )}
        </span>
    );
}

function StatTile({ value, label }: { value: number; label: string }) {
    return (
        <div className="dashboard-stat-tile h-100">
            <div className="dashboard-stat-value">{value}</div>
            <div className="dashboard-stat-label">{label}</div>
        </div>
    );
}

function AppointmentRow({ booking, showDate }: { booking: BookingDetail; showDate: boolean }) {
    return (
        <div className="d-flex align-items-center gap-3 py-3 border-bottom">
            <div className="req-avatar" style={{ width: 40, height: 40, fontSize: "0.8rem" }}>
                {initials(booking.customerName)}
            </div>
            <div className="flex-grow-1 min-w-0">
                <div className="fw-semibold text-truncate">{booking.customerName}</div>
                <div className="text-muted small text-truncate">{booking.optionName}</div>
            </div>
            <div className="text-end flex-shrink-0">
                {showDate && (
                    <div className="d-flex align-items-center justify-content-end gap-2 mb-1 text-muted small">
                        <IconCalendar size={14} />
                        <span>{formatDate(booking.date)}</span>
                    </div>
                )}
                <div className="d-flex align-items-center justify-content-end gap-2 fw-semibold small">
                    <IconClock size={14} />
                    <span>{to12h(booking.startTime)} - {to12h(booking.endTime)}</span>
                </div>
            </div>
        </div>
    );
}

// First landing page for staff mode (mirrors the owner's BusinessDashboardPage
// at the same "/" route) — a quick-glance summary instead of dropping staff
// straight into the full calendar. Pending-action counts (leave awaiting the
// owner's decision, bookings awaiting the staff's own accept/reject) surface
// as badges on the two shortcut tiles rather than a separate notification
// bell, so there's one consistent place to look instead of two.
export default function StaffDashboardPage() {
    const { token } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");
    const navigate = useNavigate();

    const [bookings, setBookings] = useState<BookingDetail[]>([]);
    const [decidedLeaveCount, setDecidedLeaveCount] = useState(0);
    const [services, setServices] = useState<Service[]>([]);
    const [loading, setLoading] = useState(true);

    const [selectedDate, setSelectedDate] = useState("");
    const [selectedServiceId, setSelectedServiceId] = useState("");

    useEffect(() => {
        if (!activeToken) return;
        const load = async () => {
            try {
                const [bookingsRes, leavesRes, servicesRes] = await Promise.all([
                    getBusinessBookings(activeToken),
                    myLeaveApplications(activeToken),
                    getBusinessServices(activeToken),
                ]);
                setBookings(bookingsRes.data?.businessBookings ?? []);
                // Notify on a decision having been made (decidedAt set), not on
                // still-pending applications — the staff member already knows
                // those are pending since they submitted them; what's new is
                // the owner's outcome. Skip ones whose leave period is already
                // over — nothing left to act on or care about there.
                setDecidedLeaveCount((leavesRes.data?.myLeaveApplications ?? [])
                    .filter(l => l.decidedAt && l.endDate >= todayISO()).length);
                setServices(servicesRes.data?.displayServices ?? []);
            } finally {
                setLoading(false);
            }
        };
        load();
    }, [activeToken]);

    const pendingBookingCount = useMemo(() => bookings.filter(b => b.status === "PENDING").length, [bookings]);
    const confirmed = useMemo(() => bookings.filter(b => b.status === "ACCEPTED"), [bookings]);

    const today = todayISO();
    const todayAppointments = useMemo(
        () => confirmed.filter(b => b.date === today).sort((a, b) => a.startTime.localeCompare(b.startTime)),
        [confirmed, today],
    );
    const upcomingAll = useMemo(() => confirmed.filter(b => b.date > today), [confirmed, today]);

    const selectedServiceName = services.find(s => s.serviceId === selectedServiceId)?.serviceName;
    const upcomingAppointments = useMemo(() => upcomingAll
        .filter(b => !selectedDate || b.date === selectedDate)
        .filter(b => !selectedServiceName || b.serviceName === selectedServiceName)
        .sort((a, b) => a.date === b.date ? a.startTime.localeCompare(b.startTime) : a.date.localeCompare(b.date)),
        [upcomingAll, selectedDate, selectedServiceName],
    );

    if (loading) {
        return (
            <Container className="d-flex justify-content-center align-items-center" style={{ minHeight: "50vh" }}>
                <Spinner animation="border" variant="primary" />
            </Container>
        );
    }

    return (
        <Container className="dashboard-page py-4" style={{ maxWidth: 960 }}>
            <h1 className="fs-2 fw-bold mb-4">Staff Dashboard</h1>

            <Row className="g-3 mb-4 align-items-stretch">
                <Col xs={6} md={3}>
                    <StatTile value={todayAppointments.length} label="Today" />
                </Col>
                <Col xs={6} md={3}>
                    <StatTile value={upcomingAll.length} label="Upcoming" />
                </Col>
                <Col xs={12} md={6} className="d-flex align-items-center justify-content-md-end gap-2 flex-wrap">
                    <ShortcutButton label="My Leave" icon={<IconCalendar size={16} />} badge={decidedLeaveCount} onClick={() => navigate("/leave")} />
                    <ShortcutButton label="My Booking" icon={<IconClock size={16} />} badge={pendingBookingCount} onClick={() => navigate("/my-calendar")} />
                </Col>
            </Row>

            <Card className="dashboard-card mb-4">
                <Card.Body>
                    <div className="dashboard-subheading mb-3">Today's Appointments</div>
                    {todayAppointments.length === 0 ? (
                        <div className="dashboard-empty">No appointments today.</div>
                    ) : (
                        <div>
                            {todayAppointments.map(b => <AppointmentRow key={b.bookingId} booking={b} showDate={false} />)}
                        </div>
                    )}
                </Card.Body>
            </Card>

            <Card className="dashboard-card">
                <Card.Body>
                    <Row className="align-items-end mb-3 g-3">
                        <Col xs="auto">
                            <div className="dashboard-subheading mb-0">Upcoming Appointments</div>
                        </Col>
                        <Col xs={12} md="auto" className="ms-md-auto">
                            <Form.Label className="fw-semibold small mb-1">Select Date</Form.Label>
                            <Form.Control size="sm" type="date" value={selectedDate} onChange={e => setSelectedDate(e.target.value)} />
                        </Col>
                        <Col xs={12} md="auto">
                            <Form.Label className="fw-semibold small mb-1">Select Service</Form.Label>
                            <Form.Select size="sm" value={selectedServiceId} onChange={e => setSelectedServiceId(e.target.value)}>
                                <option value="">— Select Service —</option>
                                {services.map(s => <option key={s.serviceId} value={s.serviceId}>{s.serviceName}</option>)}
                            </Form.Select>
                        </Col>
                    </Row>
                    {upcomingAppointments.length === 0 ? (
                        <div className="dashboard-empty">
                            No upcoming appointments{selectedDate || selectedServiceId ? " match your filters." : "."}
                        </div>
                    ) : (
                        <div>
                            {upcomingAppointments.map(b => <AppointmentRow key={b.bookingId} booking={b} showDate />)}
                        </div>
                    )}
                </Card.Body>
            </Card>
        </Container>
    );
}
