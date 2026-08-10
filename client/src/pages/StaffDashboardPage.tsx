import { useEffect, useMemo, useState } from "react";
import { Badge, Button, Col, Container, Form, Row, Spinner } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { getBusinessBookings, type BookingDetail } from "../services/BookingService";
import { getBusinessServices, type Service } from "../services/ServiceService";
import { myLeaveApplications } from "../services/LeaveService";
import { extractTime } from "../services/ServiceSlotService";
import { todayISO } from "../utils/serviceSlotHelpers";
import { IconCalendar, IconClock } from "../components/icons";

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

// Small notification-dot badge for the corner of a button — same visual
// treatment AppNavbar's NavCountBadge uses for nav links, just inline here
// since these buttons live on the page instead of the nav bar.
function CornerBadge({ count }: { count: number }) {
    if (count === 0) return null;
    return (
        <Badge
            bg="danger"
            pill
            className="position-absolute top-0 start-100 translate-middle"
            style={{ fontSize: "0.7rem" }}
            title={`${count} pending`}
        >
            {count}
        </Badge>
    );
}

function AppointmentRow({ booking, showDate }: { booking: BookingDetail; showDate: boolean }) {
    return (
        <div className="d-flex justify-content-between align-items-center py-3 border-bottom">
            <div>
                <div className="fw-semibold">{booking.customerName}</div>
                <div className="text-muted small">{booking.optionName}</div>
            </div>
            <div className="text-end">
                {showDate && (
                    <div className="d-flex align-items-center justify-content-end gap-2 mb-1">
                        <IconCalendar size={16} />
                        <span>{formatDate(booking.date)}</span>
                    </div>
                )}
                <div className="d-flex align-items-center justify-content-end gap-2">
                    <IconClock size={16} />
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
// as badges on the two shortcut buttons rather than a separate notification
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

    const selectedServiceName = services.find(s => s.serviceId === selectedServiceId)?.serviceName;
    const upcomingAppointments = useMemo(() => confirmed
        .filter(b => b.date > today)
        .filter(b => !selectedDate || b.date === selectedDate)
        .filter(b => !selectedServiceName || b.serviceName === selectedServiceName)
        .sort((a, b) => a.date === b.date ? a.startTime.localeCompare(b.startTime) : a.date.localeCompare(b.date)),
        [confirmed, today, selectedDate, selectedServiceName],
    );

    if (loading) {
        return (
            <Container className="d-flex justify-content-center align-items-center" style={{ minHeight: "50vh" }}>
                <Spinner animation="border" variant="primary" />
            </Container>
        );
    }

    return (
        <Container className="py-4 text-start" style={{ maxWidth: 900 }}>
            <h1 className="fs-2 fw-bold mb-4">Staff Dashboard</h1>

            <div className="d-flex gap-3 mb-4">
                <span className="position-relative">
                    <Button variant="primary" onClick={() => navigate("/leave")}>My Leave</Button>
                    <CornerBadge count={decidedLeaveCount} />
                </span>
                <span className="position-relative">
                    <Button variant="primary" onClick={() => navigate("/my-calendar")}>My Booking</Button>
                    <CornerBadge count={pendingBookingCount} />
                </span>
            </div>

            <h2 className="fs-4 fw-bold mb-2">Today Appointments</h2>
            {todayAppointments.length === 0 ? (
                <p className="text-muted">No appointments today.</p>
            ) : (
                <div className="mb-4">
                    {todayAppointments.map(b => <AppointmentRow key={b.bookingId} booking={b} showDate={false} />)}
                </div>
            )}

            <Row className="align-items-end mb-2 g-3">
                <Col xs="auto">
                    <h2 className="fs-4 fw-bold mb-0">Upcoming Appointments</h2>
                </Col>
                <Col xs={12} md="auto" className="ms-md-auto">
                    <Form.Label className="fw-semibold small mb-1">Select Date</Form.Label>
                    <Form.Control type="date" value={selectedDate} onChange={e => setSelectedDate(e.target.value)} />
                </Col>
                <Col xs={12} md="auto">
                    <Form.Label className="fw-semibold small mb-1">Select Service</Form.Label>
                    <Form.Select value={selectedServiceId} onChange={e => setSelectedServiceId(e.target.value)}>
                        <option value="">— Select Service —</option>
                        {services.map(s => <option key={s.serviceId} value={s.serviceId}>{s.serviceName}</option>)}
                    </Form.Select>
                </Col>
            </Row>
            {upcomingAppointments.length === 0 ? (
                <p className="text-muted">No upcoming appointments{selectedDate || selectedServiceId ? " match your filters." : "."}</p>
            ) : (
                <div>
                    {upcomingAppointments.map(b => <AppointmentRow key={b.bookingId} booking={b} showDate />)}
                </div>
            )}
        </Container>
    );
}
