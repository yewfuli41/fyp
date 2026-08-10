import { useState } from "react";
import { Badge, Button, Col, Row } from "react-bootstrap";
import { type BookingDetail } from "../services/BookingService";
import { extractTime } from "../services/ServiceSlotService";
import "../styles/BookingRequestsPanel.css";

interface Props {
    pending: BookingDetail[];
    busyId: string | null;
    onAccept: (b: BookingDetail) => void;
    onReject: (b: BookingDetail) => void;
    onReschedule: (b: BookingDetail) => void;
}

// Bootstrap-icons paths (16×16), tinted via currentColor.
const Icon = ({ path, size = 16 }: { path: string; size?: number }) => (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
        <path d={path} />
    </svg>
);
const ICON = {
    chevronRight: "M4.646 1.646a.5.5 0 0 1 .708 0l6 6a.5.5 0 0 1 0 .708l-6 6a.5.5 0 0 1-.708-.708L10.293 8 4.646 2.354a.5.5 0 0 1 0-.708",
    chevronDown: "M1.646 4.646a.5.5 0 0 1 .708 0L8 10.293l5.646-5.647a.5.5 0 0 1 .708.708l-6 6a.5.5 0 0 1-.708 0l-6-6a.5.5 0 0 1 0-.708",
    calendar: "M3.5 0a.5.5 0 0 1 .5.5V1h8V.5a.5.5 0 0 1 1 0V1h1a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2H2a2 2 0 0 1-2-2V3a2 2 0 0 1 2-2h1V.5a.5.5 0 0 1 .5-.5M1 4v10a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V4z",
    clock: "M8 3.5a.5.5 0 0 0-1 0V9a.5.5 0 0 0 .252.434l3.5 2a.5.5 0 0 0 .496-.868L8 8.71zM8 16A8 8 0 1 0 8 0a8 8 0 0 0 0 16m7-8A7 7 0 1 1 1 8a7 7 0 0 1 14 0",
    person: "M8 8a3 3 0 1 0 0-6 3 3 0 0 0 0 6m2-3a2 2 0 1 1-4 0 2 2 0 0 1 4 0m4 8c0 1-1 1-1 1H3s-1 0-1-1 1-4 6-4 6 3 6 4m-1-.004c-.001-.246-.154-.986-.832-1.664C11.516 10.68 10.289 10 8 10c-2.29 0-3.516.68-4.168 1.332-.678.678-.83 1.418-.832 1.664z",
    check: "M13.854 3.646a.5.5 0 0 1 0 .708l-7 7a.5.5 0 0 1-.708 0l-3.5-3.5a.5.5 0 1 1 .708-.708L6.5 10.293l6.646-6.647a.5.5 0 0 1 .708 0",
    reschedule: "M8 3a5 5 0 1 1-4.546 2.914.5.5 0 0 0-.908-.417A6 6 0 1 0 8 2zM8 4.466V.534a.25.25 0 0 0-.41-.192L5.23 2.308a.25.25 0 0 0 0 .384l2.36 1.966A.25.25 0 0 0 8 4.466",
    x: "M4.646 4.646a.5.5 0 0 1 .708 0L8 7.293l2.646-2.647a.5.5 0 0 1 .708.708L8.707 8l2.647 2.646a.5.5 0 0 1-.708.708L8 8.707l-2.646 2.647a.5.5 0 0 1-.708-.708L7.293 8 4.646 5.354a.5.5 0 0 1 0-.708",
};

const initials = (name: string) =>
    name.split(/\s+/).filter(Boolean).slice(0, 2).map(w => w[0]?.toUpperCase() ?? "").join("") || "?";

const shortDate = (iso: string) => {
    const d = new Date(`${iso}T00:00:00`);
    return isNaN(d.getTime()) ? iso : d.toLocaleDateString("en-GB", { weekday: "short", day: "numeric", month: "short" });
};

export default function BookingRequestsPanel({ pending, busyId, onAccept, onReject, onReschedule }: Props) {
    const [open, setOpen] = useState(false);
    const [detailId, setDetailId] = useState<string | null>(null);

    if (pending.length === 0) return null;

    return (
        <div className="mb-4">
            <div className="d-flex align-items-center gap-2 mb-3">
                <h2 className="h5 fw-bold mb-0">
                    Pending requests <Badge bg="warning" text="dark">{pending.length}</Badge>
                </h2>
                <Button
                    variant="link"
                    className="text-decoration-none p-0 d-inline-flex align-items-center text-secondary"
                    aria-label={open ? "Collapse pending requests" : "Expand pending requests"}
                    aria-expanded={open}
                    onClick={() => setOpen(o => !o)}
                >
                    <Icon path={open ? ICON.chevronDown : ICON.chevronRight} size={20} />
                </Button>
            </div>

            {open && (
                <Row className="g-3">
                    {pending.map(b => {
                        const busy = busyId === b.bookingId;
                        return (
                            <Col key={b.bookingId} xs={12} lg={6}>
                                <div className="req-card">
                                    <div className="d-flex align-items-start justify-content-between gap-3 mb-3">
                                        <div className="d-flex align-items-center gap-3">
                                            <div className="req-avatar">{initials(b.customerName)}</div>
                                            <div className="fw-bold fs-5">{b.customerName}</div>
                                        </div>
                                        <button
                                            className="req-btn req-outline req-detail-btn"
                                            onClick={() => setDetailId(id => id === b.bookingId ? null : b.bookingId)}
                                        >
                                            {detailId === b.bookingId ? "Hide detail" : "Detail"}
                                        </button>
                                    </div>

                                    <div className="fw-semibold mb-2">{b.serviceName} — {b.optionName}</div>

                                    <div className="req-meta mb-3">
                                        <span><Icon path={ICON.calendar} /> {shortDate(b.date)}</span>
                                        <span><Icon path={ICON.clock} /> {extractTime(b.startTime)}</span>
                                        <span><Icon path={ICON.person} /> {b.staffName ?? "Owner-managed"}</span>
                                    </div>

                                    {b.status === "RESCHEDULED" && (
                                        <div className="mb-2">
                                            <Badge bg="info" text="dark">Awaiting customer's acceptance</Badge>
                                        </div>
                                    )}

                                    {detailId === b.bookingId && (
                                        <div className="req-detail mb-2">
                                            {b.description
                                                ? <p className="mb-0">{b.description}</p>
                                                : <p className="mb-0 text-muted fst-italic">No additional notes from the customer.</p>}
                                        </div>
                                    )}

                                    <div className="req-actions">
                                        {b.status === "PENDING" && (
                                            <button className="req-btn req-accept" disabled={busy} onClick={() => onAccept(b)}>
                                                <Icon path={ICON.check} /> Accept
                                            </button>
                                        )}
                                        <button className="req-btn req-outline" disabled={busy} onClick={() => onReschedule(b)}>
                                            <Icon path={ICON.reschedule} /> Reschedule
                                        </button>
                                        <button className="req-btn req-outline req-reject" disabled={busy} onClick={() => onReject(b)}>
                                            <Icon path={ICON.x} /> Reject
                                        </button>
                                    </div>
                                </div>
                            </Col>
                        );
                    })}
                </Row>
            )}
        </div>
    );
}
