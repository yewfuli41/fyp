import { useState, useEffect, useCallback } from "react";
import { Alert, Button, Form, Modal, Spinner } from "react-bootstrap";
import {
    getAvailableSlots, getPublicServices,
    type AvailableSlot, type StaffUnavailability,
} from "../services/PublicService";
import { rescheduleBooking, type BookingDetail } from "../services/BookingService";
import { extractTime } from "../services/ServiceSlotService";
import { displayDate, todayISO } from "../utils/serviceSlotHelpers";
import { findBookableOptionIds, isCurrentOption } from "../utils/serviceAvailability";
import BookingDatePicker from "../components/BookingDatePicker";
import "../styles/RescheduleBookingModal.css";

interface Props {
    show: boolean;
    // Fires on explicit dismissal only (X button / Escape) — a successful
    // reschedule calls onDone instead and leaves closing/advancing up to the
    // caller (e.g. a queue of several bookings to reschedule in a row).
    onHide: () => void;
    booking: BookingDetail | null;
    token: string | null;
    // Fires once the reschedule has actually been committed. Optional because
    // callers that pass onStage commit later, on their own terms, and drive
    // whatever comes next themselves.
    onDone?: () => void;
    // When set, picking a slot stages the choice instead of writing it: the
    // modal calls this and stops, leaving the caller to apply it later. Used
    // by leave approval, where nothing may be written until the owner has
    // settled every affected booking and the approval itself commits.
    onStage?: (newSlotOptionId: string) => void;
    // Keeps one staff member's slots out of the picker across a date range.
    // Set while approving that staff's leave, so the owner can't move a
    // customer onto another of their slots on a day they're taking off — the
    // leave is still pending, so nothing else would stop it.
    unavailable?: StaffUnavailability;
    // When set (a staff member, not the owner, is rescheduling), only that
    // staff's own slots are offered — the backend enforces this regardless.
    lockedStaffId?: string;
    // Lets the customer pick a different service option than the one they
    // originally booked, via a dropdown defaulting to it. Owner/staff
    // reschedules (calendar management, leave-affected bookings) never get
    // this — they can only move the booking to a different time for the
    // exact option the customer booked, never swap what they're paying for.
    allowOptionChange?: boolean;
    // Extra context shown above the date picker — e.g. "1 of 2 bookings
    // affected by this leave" when working through a queue of them.
    note?: string;
}

interface OptionChoice {
    serviceOptionId: string;
    serviceName: string;
    optionName: string;
}

interface RescheduleChoice extends AvailableSlot {
    slotOptionId: string;
}

export default function RescheduleBookingModal({
    show, onHide, booking, token, onDone, onStage, unavailable, lockedStaffId, allowOptionChange, note,
}: Props) {
    const [date, setDate] = useState(todayISO());
    const [selectedOptionId, setSelectedOptionId] = useState("");
    const [optionChoices, setOptionChoices] = useState<OptionChoice[]>([]);
    const [slots, setSlots] = useState<RescheduleChoice[]>([]);
    const [loading, setLoading] = useState(false);
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState("");

    useEffect(() => {
        if (show && booking) {
            setDate(booking.date && booking.date >= todayISO() ? booking.date : todayISO());
            setSelectedOptionId(booking.serviceOptionId);
            setSlots([]);
            setError("");
        }
    }, [show, booking]);

    // Only fetched for the customer's "change service option" picker — an
    // owner/staff reschedule stays fixed to the booking's own option, so
    // there's nothing to populate a dropdown with. Filtered down to options
    // that are actually still bookable — same "current + has a real
    // upcoming slot" check the services-browsing page uses, so the customer
    // never picks something with nothing to reschedule onto.
    useEffect(() => {
        if (!show || !booking || !allowOptionChange) return;
        let cancelled = false;
        getPublicServices(booking.businessId).then(async res => {
            const entries = (res.data?.publicServices ?? []).flatMap(service =>
                service.serviceOptions.map(option => ({
                    serviceOptionId: option.serviceOptionId,
                    serviceName: service.serviceName,
                    optionName: option.serviceOptionName,
                    option,
                })),
            );
            const currentOptions = entries.filter(e => isCurrentOption(e.option)).map(e => e.option);
            const bookableIds = await findBookableOptionIds(booking.businessId, currentOptions);
            if (cancelled) return;
            const choices = entries
                .filter(e => bookableIds.has(e.serviceOptionId))
                .map(({ serviceOptionId, serviceName, optionName }) => ({ serviceOptionId, serviceName, optionName }));
            setOptionChoices(choices);
            // The booking's own option might no longer be bookable (fully
            // booked out, or no longer current) — fall back to the first
            // one that is, so the dropdown's value always matches a real
            // listed choice.
            setSelectedOptionId(prev =>
                choices.some(c => c.serviceOptionId === prev) ? prev : (choices[0]?.serviceOptionId ?? prev));
        });
        return () => { cancelled = true; };
    }, [show, booking, allowOptionChange]);

    const loadSlots = useCallback(async () => {
        if (!booking || !selectedOptionId) return;
        setLoading(true);
        setError("");
        try {
            const res = await getAvailableSlots(booking.businessId, selectedOptionId, date, lockedStaffId, undefined, unavailable);
            const choices = (res.data?.availableSlots ?? []).flatMap(slot => {
                const slotOptionId = slot.serviceSlotOptions[0]?.slotOptionId;
                return slotOptionId ? [{ ...slot, slotOptionId }] : [];
            }).sort((a, b) => a.startTime.localeCompare(b.startTime) || a.endTime.localeCompare(b.endTime));
            setSlots(choices);
        } catch {
            setError("Failed to load available slots.");
            setSlots([]);
        } finally {
            setLoading(false);
        }
        // Same reason as BookingDatePicker: depend on the unavailability's
        // fields, not the inline object's identity.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [booking, date, lockedStaffId, selectedOptionId,
        unavailable?.staffId, unavailable?.from, unavailable?.until]);

    useEffect(() => {
        if (show && booking && selectedOptionId) loadSlots();
    }, [show, booking, selectedOptionId, loadSlots]);

    const handlePick = async (newSlotOptionId: string) => {
        // Staged: hand the choice back unwritten and let the caller decide
        // when (and whether) it reaches the database.
        if (onStage) {
            onStage(newSlotOptionId);
            return;
        }
        if (!token || !booking) return;
        setBusy(true);
        setError("");
        try {
            const result = await rescheduleBooking(token, booking.bookingId, newSlotOptionId);
            if (result.errors?.length) {
                setError(result.errors[0].message);
                return;
            }
            // Success only ever calls onDone — the caller decides whether
            // that means closing (simple case) or moving on to the next
            // booking in a queue without the modal flickering shut first.
            onDone?.();
        } catch {
            setError("Something went wrong. Please try again.");
        } finally {
            setBusy(false);
        }
    };

    return (
        <Modal show={show} onHide={onHide} backdrop="static">
            <Modal.Header closeButton><Modal.Title>Reschedule booking</Modal.Title></Modal.Header>
            <Modal.Body>
                {note && <Alert variant="info" className="py-2">{note}</Alert>}

                {/* The slot being moved. Shown for every reschedule, but it
                    matters most in the leave queue, where the owner works
                    through several bookings in a row and the service name
                    alone doesn't say which appointment is on screen. */}
                {booking && (
                    <div className="rb-current mb-3">
                        <div className="rb-current-label">Currently booked</div>
                        <div className="rb-current-when">
                            {displayDate(new Date(`${booking.date}T00:00:00`))}
                            {" · "}
                            {extractTime(booking.startTime)}–{extractTime(booking.endTime)}
                        </div>
                        <div className="rb-current-meta">
                            {booking.serviceName} — {booking.optionName}
                            {" · "}{booking.customerName}
                            {booking.staffName ? ` · ${booking.staffName}` : " · Owner-managed"}
                        </div>
                    </div>
                )}

                {booking && allowOptionChange && optionChoices.length > 0 ? (
                    <Form.Group className="mb-3">
                        <Form.Label>Service option</Form.Label>
                        <Form.Select
                            value={selectedOptionId}
                            onChange={e => setSelectedOptionId(e.target.value)}
                        >
                            {optionChoices.map(o => (
                                <option key={o.serviceOptionId} value={o.serviceOptionId}>
                                    {o.serviceName} — {o.optionName}
                                </option>
                            ))}
                        </Form.Select>
                    </Form.Group>
                ) : null}

                {booking && (
                    <Form.Group className="mb-3">
                        <Form.Label>Pick a new date</Form.Label>
                        <BookingDatePicker
                            businessId={booking.businessId}
                            // Falls back to the booking's own option for the
                            // first render, before the effect below has copied
                            // it into state — without it the picker would
                            // briefly ask for "any option in the business" and
                            // colour dates by slots this booking can't move to.
                            serviceOptionId={selectedOptionId || booking.serviceOptionId}
                            staffId={lockedStaffId}
                            unavailable={unavailable}
                            value={date}
                            onChange={setDate}
                        />
                    </Form.Group>
                )}

                {error && <Alert variant="danger">{error}</Alert>}

                {loading ? (
                    <div className="text-center py-3"><Spinner size="sm" /></div>
                ) : slots.length === 0 ? (
                    <p className="text-muted mb-0">No available time slots on this date.</p>
                ) : (
                    <div className="d-flex flex-column gap-2">
                        {slots.map(slot => (
                            <Button
                                key={slot.slotOptionId}
                                variant="outline-primary"
                                className="d-flex justify-content-between align-items-center"
                                disabled={busy}
                                onClick={() => handlePick(slot.slotOptionId)}
                            >
                                <span>{extractTime(slot.startTime)}–{extractTime(slot.endTime)}</span>
                                <span className="text-muted small">{slot.staff?.name ?? "Owner-managed"}</span>
                            </Button>
                        ))}
                    </div>
                )}
            </Modal.Body>
        </Modal>
    );
}
