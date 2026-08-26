import { useState, useEffect, useCallback, useMemo } from "react";
import { Alert, Button, Card, Col, Container, Form, Modal, Row, Spinner, ToggleButton, ToggleButtonGroup } from "react-bootstrap";
import { useAuth } from "../auth/AuthContext";
import {
    getMyAppointments, acceptReschedule, cancelBooking, updateBookingDescription,
    type BookingDetail, type BookingStatus,
} from "../services/BookingService";
import { extractTime } from "../services/ServiceSlotService";
import RescheduleBookingModal from "../modals/RescheduleBookingModal";
import acceptedIcon from "../assets/4315445.png";
import rejectedIcon from "../assets/cancel--v3.jpg";
import cancelledIcon from "../assets/5578867.png";
import pendingIcon from "../assets/2931154-200.png";
import rescheduledIcon from "../assets/3777044-200.png";
import pastIcon from "../assets/2496458-200.png";

const STATUS_META: Record<BookingStatus, { icon: string; label: string; color: string }> = {
    PENDING: { icon: pendingIcon, label: "Pending", color: "#b8860b" },
    ACCEPTED: { icon: acceptedIcon, label: "Accepted", color: "#2e7d32" },
    RESCHEDULED: { icon: rescheduledIcon, label: "Rescheduled", color: "#1565c0" },
    CANCELLED: { icon: cancelledIcon, label: "Cancelled", color: "#c62828" },
    REJECTED: { icon: rejectedIcon, label: "Rejected", color: "#c62828" },
    PAST: { icon: pastIcon, label: "Past", color: "#6c757d" },
};

const STATUS_OPTIONS: BookingStatus[] = ["PENDING", "ACCEPTED", "RESCHEDULED", "CANCELLED", "REJECTED", "PAST"];

const isActionable = (s: BookingStatus) => s === "PENDING" || s === "ACCEPTED" || s === "RESCHEDULED";

const PencilIcon = () => (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
        <path d="M12.146.146a.5.5 0 0 1 .708 0l3 3a.5.5 0 0 1 0 .708l-10 10a.5.5 0 0 1-.168.11l-5 2a.5.5 0 0 1-.65-.65l2-5a.5.5 0 0 1 .11-.168zM11.207 2.5 13.5 4.793 14.793 3.5 12.5 1.207zm1.586 3L10.5 3.207 4 9.707V10h.5a.5.5 0 0 1 .5.5v.5h.5a.5.5 0 0 1 .5.5v.5h.293zm-9.761 5.175-.106.106-1.528 3.821 3.821-1.528.106-.106A.5.5 0 0 1 5 12.5V12h-.5a.5.5 0 0 1-.5-.5V11h-.5a.5.5 0 0 1-.468-.325" />
    </svg>
);

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

export default function AppointmentsPage() {
    const { token } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [appointments, setAppointments] = useState<BookingDetail[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [busyId, setBusyId] = useState<string | null>(null);

    // Filters
    const [dateFilter, setDateFilter] = useState("");
    const [statusFilters, setStatusFilters] = useState<BookingStatus[]>([]);
    const [search, setSearch] = useState("");


    const [cancelTarget, setCancelTarget] = useState<BookingDetail | null>(null);
    const [rescheduling, setRescheduling] = useState<BookingDetail | null>(null);

    // Inline note editing — only one card's note can be in edit mode at a time.
    const [editingNoteId, setEditingNoteId] = useState<string | null>(null);
    const [noteDraft, setNoteDraft] = useState("");
    const [noteSaving, setNoteSaving] = useState(false);

    const fetchAppointments = useCallback(async () => {
        if (!activeToken) return;
        try {
            const res = await getMyAppointments(activeToken);
            if (res.errors?.length) setError(res.errors[0].message);
            else setAppointments(res.data?.myAppointments ?? []);
        } catch {
            setError("Failed to load appointments.");
        } finally {
            setLoading(false);
        }
    }, [activeToken]);

    useEffect(() => { fetchAppointments(); }, [fetchAppointments]);

    const runAction = async (
        booking: BookingDetail,
        fn: (token: string, id: string) => Promise<{ errors?: { message: string }[] }>,
    ) => {
        if (!activeToken) return;
        setBusyId(booking.bookingId);
        setError("");
        try {
            const result = await fn(activeToken, booking.bookingId);
            if (result.errors?.length) { setError(result.errors[0].message); return; }
            setCancelTarget(null);
            fetchAppointments();
        } catch {
            setError("Something went wrong. Please try again.");
        } finally {
            setBusyId(null);
        }
    };

    const startEditingNote = (booking: BookingDetail) => {
        setEditingNoteId(booking.bookingId);
        setNoteDraft(booking.description ?? "");
        setError("");
    };

    const saveNote = async (bookingId: string) => {
        if (!activeToken) return;
        setNoteSaving(true);
        setError("");
        try {
            const result = await updateBookingDescription(activeToken, bookingId, noteDraft);
            if (result.errors?.length) { setError(result.errors[0].message); return; }
            setEditingNoteId(null);
            fetchAppointments();
        } catch {
            setError("Something went wrong. Please try again.");
        } finally {
            setNoteSaving(false);
        }
    };

    const filtered = useMemo(() => {
        const q = search.trim().toLowerCase();
        return appointments
            .filter(a => {
                if (dateFilter && a.date !== dateFilter) return false;
                if (statusFilters.length > 0 && !statusFilters.includes(a.status)) return false;
                if (q && !(
                    a.businessName.toLowerCase().includes(q) ||
                    a.serviceName.toLowerCase().includes(q) ||
                    a.optionName.toLowerCase().includes(q) ||
                    (a.staffName ?? "").toLowerCase().includes(q)
                )) return false;
                return true;
            })
            .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
    }, [appointments, dateFilter, statusFilters, search]);

    if (loading) {
        return (
            <Container className="d-flex justify-content-center align-items-center" style={{ minHeight: "50vh" }}>
                <Spinner animation="border" variant="primary" />
            </Container>
        );
    }

    return (
        <Container className="py-4 text-start" style={{ maxWidth: 900 }}>
            <h1 className="fs-2 fw-bold mb-4">Appointments Management</h1>

            <Row className="g-3 mb-4">
                <Col xs={12} md={3}>
                    <Form.Label className="fw-semibold">Select Date</Form.Label>
                    <Form.Control type="date" value={dateFilter} onChange={e => setDateFilter(e.target.value)} />
                </Col>
                <Col xs={12} md={5}>
                    <Form.Label className="fw-semibold d-block">Select Status</Form.Label>
                    <ToggleButtonGroup
                        type="checkbox"
                        value={statusFilters}
                        onChange={(vals: string[]) => setStatusFilters(vals as BookingStatus[])}
                        className="flex-wrap gap-2"
                    >
                        {STATUS_OPTIONS.map(s => (
                            <ToggleButton key={s} id={`status-${s}`} value={s} variant="outline-primary" size="sm" className="rounded-pill">
                                {STATUS_META[s].label}
                            </ToggleButton>
                        ))}
                    </ToggleButtonGroup>
                    <div className="text-muted small mt-1">All shown when none selected.</div>
                </Col>
                <Col xs={12} md={4}>
                    <Form.Label className="fw-semibold">Search Appointments</Form.Label>
                    <Form.Control
                        type="text"
                        placeholder="Search by business, service, staff…"
                        value={search}
                        onChange={e => setSearch(e.target.value)}
                    />
                </Col>
            </Row>

            {error && <Alert variant="danger">{error}</Alert>}

            {filtered.length === 0 ? (
                <p className="text-muted">
                    {appointments.length === 0 ? "You have no appointments yet." : "No appointments match your filters."}
                </p>
            ) : (
                <div className="d-flex flex-column gap-3">
                    {filtered.map((a, i) => {
                        const meta = STATUS_META[a.status];
                        const busy = busyId === a.bookingId;
                        return (
                            <Card key={a.bookingId} className="shadow-sm">
                                <Card.Body>
                                    <div className="d-flex justify-content-between align-items-start flex-wrap gap-2 mb-3">
                                        <h2 className="h5 fw-bold mb-0">{i + 1}. {a.businessName}</h2>
                                        {isActionable(a.status) && (
                                            <div className="d-flex gap-2">
                                                {a.status === "RESCHEDULED" && (
                                                    <Button size="sm" variant="success" disabled={busy}
                                                        onClick={() => runAction(a, acceptReschedule)}>
                                                        Accept
                                                    </Button>
                                                )}
                                                <Button size="sm" variant="primary" disabled={busy}
                                                    onClick={() => setRescheduling(a)}>
                                                    Reschedule
                                                </Button>
                                                <Button size="sm" variant="danger" disabled={busy}
                                                    onClick={() => setCancelTarget(a)}>
                                                    Cancel
                                                </Button>
                                            </div>
                                        )}
                                    </div>

                                    <div className="ps-1">
                                        <p className="mb-2"><strong>Service Option:</strong> {a.serviceName} — {a.optionName}</p>
                                        <p className="mb-2"><strong>Date:</strong> {formatDate(a.date)}</p>
                                        <p className="mb-2"><strong>Time Slot:</strong> {to12h(a.startTime)} - {to12h(a.endTime)}</p>
                                        <p className="mb-2"><strong>Staff:</strong> {a.staffName ?? "Owner-managed"}</p>

                                        {editingNoteId === a.bookingId ? (
                                            <div className="mb-3">
                                                <Form.Control
                                                    as="textarea"
                                                    rows={2}
                                                    autoFocus
                                                    value={noteDraft}
                                                    onChange={e => setNoteDraft(e.target.value)}
                                                    placeholder="Anything the business should know before your appointment"
                                                />
                                                <div className="d-flex gap-2 mt-2">
                                                    <Button size="sm" variant="primary" disabled={noteSaving}
                                                        onClick={() => saveNote(a.bookingId)}>
                                                        {noteSaving ? "Saving…" : "Save"}
                                                    </Button>
                                                    <Button size="sm" variant="outline-secondary" disabled={noteSaving}
                                                        onClick={() => setEditingNoteId(null)}>
                                                        Cancel
                                                    </Button>
                                                </div>
                                            </div>
                                        ) : (
                                            <p className="mb-3 d-flex align-items-start gap-2">
                                                <span>
                                                    <strong>Notes:</strong>{" "}
                                                    {a.description || <span className="text-muted fst-italic">None</span>}
                                                </span>
                                                {isActionable(a.status) && (
                                                    <Button
                                                        variant="link"
                                                        className="p-0 text-secondary"
                                                        aria-label="Edit notes"
                                                        onClick={() => startEditingNote(a)}
                                                    >
                                                        <PencilIcon />
                                                    </Button>
                                                )}
                                            </p>
                                        )}

                                        <div className="d-flex align-items-center gap-2">
                                            <img src={meta.icon} alt={meta.label} style={{ width: 34, height: 34, objectFit: "contain" }} />
                                            <span className="fw-bold" style={{ color: meta.color }}>{meta.label}</span>
                                        </div>
                                    </div>
                                </Card.Body>
                            </Card>
                        );
                    })}
                </div>
            )}

            {/* ── Cancel confirmation ──────────────────────────────────────────── */}
            <Modal show={!!cancelTarget} onHide={() => setCancelTarget(null)}>
                <Modal.Header closeButton><Modal.Title>Cancel appointment</Modal.Title></Modal.Header>
                <Modal.Body>
                    Cancel your appointment with <strong>{cancelTarget?.businessName}</strong>
                    {cancelTarget ? ` on ${formatDate(cancelTarget.date)}` : ""}? This can't be undone.
                    {error && <Alert variant="danger" className="py-2 mb-0 mt-3">{error}</Alert>}
                </Modal.Body>
                <Modal.Footer>
                    <Button variant="outline-secondary" onClick={() => setCancelTarget(null)}>Keep it</Button>
                    <Button
                        variant="danger"
                        disabled={busyId === cancelTarget?.bookingId}
                        onClick={() => cancelTarget && runAction(cancelTarget, cancelBooking)}
                    >
                        Yes, cancel
                    </Button>
                </Modal.Footer>
            </Modal>

            <RescheduleBookingModal
                show={!!rescheduling}
                onHide={() => setRescheduling(null)}
                booking={rescheduling}
                token={activeToken}
                allowOptionChange
                onDone={() => { setRescheduling(null); fetchAppointments(); }}
            />
        </Container>
    );
}
