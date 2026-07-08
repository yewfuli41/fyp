import { useState, useEffect } from "react";
import { Alert, Button, Form, Modal, Spinner } from "react-bootstrap";
import { extractTime } from "../services/ServiceSlotService";
import { getAvailableSlots, type AvailableSlot } from "../services/PublicService";
import { createBooking } from "../services/BookingService";
import { applyGraphQLErrors } from "../utils/graphqlErrors";

interface Staff {
    staffId: string;
    name: string;
}

interface Props {
    show: boolean;
    onHide: () => void;
    businessId: string;
    serviceOptionId: string;
    serviceOptionName: string;
    staff: Staff[];
    token: string | null;
}

export default function BookingModal({ show, onHide, businessId, serviceOptionId, serviceOptionName, staff, token }: Props) {
    const today = new Date().toISOString().slice(0, 10);

    const [date, setDate] = useState(today);
    const [staffId, setStaffId] = useState("");
    const [slots, setSlots] = useState<AvailableSlot[]>([]);
    const [selectedSlotOptionId, setSelectedSlotOptionId] = useState("");
    const [loadingSlots, setLoadingSlots] = useState(false);
    const [formError, setFormError] = useState("");
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [success, setSuccess] = useState(false);

    useEffect(() => {
        if (!show) return;
        setSlots([]);
        setSelectedSlotOptionId("");
        setFormError("");
        setSuccess(false);
        fetchSlots(date, staffId);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [show]);

    const fetchSlots = async (d: string, sid: string) => {
        setLoadingSlots(true);
        setSlots([]);
        setSelectedSlotOptionId("");
        try {
            const res = await getAvailableSlots(businessId, serviceOptionId, d, sid || undefined);
            setSlots(res.data?.availableSlots ?? []);
        } catch {
            setFormError("Failed to load available slots.");
        } finally {
            setLoadingSlots(false);
        }
    };

    const handleDateChange = (d: string) => {
        setDate(d);
        fetchSlots(d, staffId);
    };

    const handleStaffChange = (sid: string) => {
        setStaffId(sid);
        fetchSlots(date, sid);
    };

    const handleBook = async () => {
        if (!token) { setFormError("Please log in to make a booking."); return; }
        if (!selectedSlotOptionId) { setFormError("Please select a time slot."); return; }
        setIsSubmitting(true);
        setFormError("");
        try {
            const res = await createBooking(token, selectedSlotOptionId);
            if (applyGraphQLErrors(res, { setFormError, fallbackMessage: "Booking failed." })) return;
            setSuccess(true);
        } catch {
            setFormError("Something went wrong. Please try again.");
        } finally {
            setIsSubmitting(false);
        }
    };

    const handleClose = () => {
        setDate(today);
        setStaffId("");
        setSlots([]);
        setSelectedSlotOptionId("");
        setFormError("");
        setSuccess(false);
        onHide();
    };

    return (
        <Modal show={show} onHide={handleClose} centered>
            <Modal.Header closeButton>
                <Modal.Title>Book — {serviceOptionName}</Modal.Title>
            </Modal.Header>
            <Modal.Body>
                {success ? (
                    <Alert variant="success">
                        Booking submitted! Your booking is pending confirmation.
                    </Alert>
                ) : (
                    <>
                        {formError && <Alert variant="danger">{formError}</Alert>}

                        <Form.Group className="mb-3">
                            <Form.Label><strong>1. Select Date</strong></Form.Label>
                            <Form.Control
                                type="date"
                                value={date}
                                min={today}
                                onChange={e => handleDateChange(e.target.value)}
                            />
                        </Form.Group>

                        {staff.length > 0 && (
                            <Form.Group className="mb-3">
                                <Form.Label><strong>2. Select Staff</strong> <span className="text-muted fw-normal">(optional)</span></Form.Label>
                                <Form.Select value={staffId} onChange={e => handleStaffChange(e.target.value)}>
                                    <option value="">All staff</option>
                                    {staff.map(s => (
                                        <option key={s.staffId} value={s.staffId}>{s.name}</option>
                                    ))}
                                </Form.Select>
                            </Form.Group>
                        )}

                        <div className="mb-2"><strong>{staff.length > 0 ? "3." : "2."} Select Time Slot</strong></div>
                        {loadingSlots ? (
                            <div className="text-center py-3"><Spinner size="sm" /> Loading slots…</div>
                        ) : slots.length === 0 ? (
                            <p className="text-muted">No slots available for this date.</p>
                        ) : (
                            <div className="d-flex flex-column gap-2">
                                {slots.map(slot => {
                                    const slotOptionId = slot.serviceSlotOptions[0]?.slotOptionId ?? "";
                                    const staffName = slot.staff?.name;
                                    const isSelected = selectedSlotOptionId === slotOptionId;
                                    return (
                                        <Button
                                            key={slot.serviceSlotId}
                                            variant={isSelected ? "primary" : "outline-secondary"}
                                            onClick={() => setSelectedSlotOptionId(slotOptionId)}
                                            className="text-start"
                                        >
                                            <div>{extractTime(slot.startTime)} – {extractTime(slot.endTime)}</div>
                                            {staffName && <small className="text-muted">{staffName}</small>}
                                        </Button>
                                    );
                                })}
                            </div>
                        )}
                    </>
                )}
            </Modal.Body>
            <Modal.Footer>
                <Button variant="outline-secondary" onClick={handleClose}>
                    {success ? "Close" : "Cancel"}
                </Button>
                {!success && (
                    <Button variant="primary" onClick={handleBook} disabled={isSubmitting || !selectedSlotOptionId}>
                        {isSubmitting ? "Booking…" : "Book"}
                    </Button>
                )}
            </Modal.Footer>
        </Modal>
    );
}
