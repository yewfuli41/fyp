import { useEffect, useMemo, type Dispatch, type SetStateAction } from "react";
import { Alert, Button, Form, Modal } from "react-bootstrap";
import type { Service } from "../services/ServiceService";
import type { Staff } from "../services/StaffService";
import { extractTime, type ServiceSlot, type ServiceSlotInput } from "../services/ServiceSlotService";
import type { BookingDetail } from "../services/BookingService";
import { computeTimeOptions, defaultOptionId, isOptionOfferedOn, todayISO, type WorkingHour } from "../utils/serviceSlotHelpers";

interface ManageServiceSlotModalProps {
    show: boolean;
    onHide: () => void;
    managing: ServiceSlot | null;
    manageError: string;

    // Reassign staff (only surfaced once the slot has a booking)
    reassignTo: string;
    setReassignTo: Dispatch<SetStateAction<string>>;
    availableStaff: { staffId: string; name: string }[];
    onReassign: () => void;
    manageBusy: boolean;

    // Single occurrence edit only. Recurring weekday edits are intentionally not exposed here.
    services: Service[];
    staff: Staff[];
    workingHours: WorkingHour[];
    lines: string[];
    editForm: ServiceSlotInput;
    setEditForm: Dispatch<SetStateAction<ServiceSlotInput>>;
    editFormServiceId: string;
    setEditFormServiceId: Dispatch<SetStateAction<string>>;
    editFormError: string;
    editFieldErrors: Record<string, string>;
    setEditFieldErrors: Dispatch<SetStateAction<Record<string, string>>>;
    onUpdate: () => void;
    isEditSaving: boolean;
    lockedStaffId?: string;

    // Delete
    deleteFuture: boolean;
    setDeleteFuture: Dispatch<SetStateAction<boolean>>;
    onDelete: () => void;

    // Reassigning to someone else on the team is an owner-only action —
    // false hides the whole reassign section for a staff member.
    allowReassign?: boolean;

    // The live booking occupying this slot (pending/accepted/rescheduled), if
    // any — enables Cancel/Reschedule regardless of whether it's been accepted yet.
    booking?: BookingDetail | null;
    onCancelBooking?: () => void;
    onRescheduleBooking?: () => void;
    // Rejecting is only offered while the booking is the business's own
    // reschedule proposal (awaiting the customer's acceptance) — the business
    // can withdraw it, it just can't accept it itself.
    onRejectBooking?: () => void;
}

export default function ManageServiceSlotModal({
    show, onHide, managing, manageError,
    reassignTo, setReassignTo, availableStaff, onReassign, manageBusy,
    services, staff, workingHours, lines,
    editForm, setEditForm, editFormServiceId, setEditFormServiceId, editFormError,
    editFieldErrors, setEditFieldErrors, onUpdate, isEditSaving, lockedStaffId,
    deleteFuture, setDeleteFuture, onDelete, allowReassign = true,
    booking, onCancelBooking, onRescheduleBooking, onRejectBooking,
}: ManageServiceSlotModalProps) {
    // Check the option is actually effective on the slot's own date — not
    // just today (this is single-occurrence editing, so there's always one
    // concrete date). The backend re-resolves per-date regardless, but this
    // keeps the checkbox list from offering something that won't apply.
    const editCheckDate = editForm.date || todayISO();

    const editSelectedService = services.find(s => s.serviceId === editFormServiceId);
    const editActiveOptions = editSelectedService?.serviceOptions.filter(o => isOptionOfferedOn(o, editCheckDate)) ?? [];
    // A service with no option offered on the target date can't be selected
    // here — but keep the slot's current service in the list even if it no
    // longer qualifies, so the existing selection doesn't just disappear out
    // from under the owner.
    const editSelectableServices = services.filter(s =>
        s.serviceOptions.some(o => isOptionOfferedOn(o, editCheckDate)) || s.serviceId === editFormServiceId);

    const editTimeOptions = useMemo(
        () => computeTimeOptions(editForm, staff, workingHours, lines),
        [editForm.staffId, editForm.daysOfWeek, editForm.date, staff, workingHours, lines],
    );

    useEffect(() => {
        if (!managing || managing.hasBooking) return;
        const startOptions = editTimeOptions.slice(0, -1);
        if (startOptions.length === 0) return;
        setEditForm(f => {
            const start = startOptions.includes(f.startTime) ? f.startTime : startOptions[0];
            const endOpts = editTimeOptions.filter(t => t > start);
            const end = endOpts.includes(f.endTime) ? f.endTime : (endOpts[0] ?? f.endTime);
            if (start === f.startTime && end === f.endTime) return f;
            return { ...f, startTime: start, endTime: end };
        });
    }, [editTimeOptions, managing, setEditForm]);

    return (
        <Modal show={show} onHide={onHide} size="lg" backdrop="static">
            <Modal.Header closeButton><Modal.Title>Manage Service Slot</Modal.Title></Modal.Header>
            <Modal.Body>
                {managing && (managing.hasBooking ? (
                    <>
                        <p className="mb-1"><strong>Date:</strong> {managing.date}</p>
                        <p className="mb-1"><strong>Time:</strong> {extractTime(managing.startTime)}–{extractTime(managing.endTime)}</p>
                        <p className="mb-1"><strong>Staff:</strong> {managing.staff?.name ?? "—"}</p>
                        <p className="mb-3"><strong>Options:</strong>{" "}
                            {managing.serviceSlotOptions.map(p => p.serviceOption.serviceOptionName).join(", ") || "—"}
                        </p>

                        {booking && (
                            <>
                                <hr />
                                <p className="mb-1"><strong>Booking:</strong> {booking.customerName} ({booking.optionName})</p>
                                <p className="mb-1"><strong>Status:</strong> {booking.status.toLowerCase()}</p>
                                <p className="mb-2">
                                    <strong>Note:</strong>{" "}
                                    {booking.description
                                        ? booking.description
                                        : <span className="text-muted fst-italic">None</span>}
                                </p>
                                <div className="d-flex gap-2 mb-2">
                                    <Button variant="outline-primary" size="sm" disabled={manageBusy} onClick={onRescheduleBooking}>
                                        Reschedule booking
                                    </Button>
                                    {booking.status === "RESCHEDULED" && (
                                        <Button variant="outline-danger" size="sm" disabled={manageBusy} onClick={onRejectBooking}>
                                            Reject reschedule
                                        </Button>
                                    )}
                                    <Button variant="danger" size="sm" disabled={manageBusy} onClick={onCancelBooking}>
                                        Cancel booking
                                    </Button>
                                </div>
                            </>
                        )}

                        {allowReassign ? (
                            <>
                                <Alert variant="info" className="py-2 small">
                                    This slot already has a booking — it cannot be deleted, and its date, time and
                                    options can no longer be changed. You can still reassign staff below.
                                </Alert>

                                <hr />
                                <Form.Group className="mb-2">
                                    <Form.Label className="fw-semibold">Reassign staff</Form.Label>
                                    <Form.Select value={reassignTo} onChange={e => setReassignTo(e.target.value)}>
                                        <option value="">Select…</option>
                                        <option value="__unassign__">Unassigned (owner-managed)</option>
                                        {availableStaff.map(s => <option key={s.staffId} value={s.staffId}>{s.name}</option>)}
                                    </Form.Select>
                                    {availableStaff.length === 0 && (
                                        <div className="text-muted small mt-1">No available staff for this time — you can still set it to owner-managed.</div>
                                    )}
                                </Form.Group>
                                {manageError && <Alert variant="danger" className="py-2">{manageError}</Alert>}
                                <Button variant="primary" size="sm" onClick={onReassign} disabled={!reassignTo || manageBusy}>
                                    Save reassignment
                                </Button>
                            </>
                        ) : (
                            <Alert variant="info" className="py-2 small mb-0">
                                This slot already has a booking — it cannot be deleted. Contact the business owner
                                to change or reassign it.
                            </Alert>
                        )}
                    </>
                ) : (
                    <>
                        <Form.Group className="mb-3">
                            <Form.Label>Service <span className="text-danger">*</span></Form.Label>
                            <Form.Select
                                value={editFormServiceId}
                                onChange={e => {
                                    const serviceId = e.target.value;
                                    setEditFormServiceId(serviceId);
                                    const pkgId = defaultOptionId(services, serviceId, editCheckDate);
                                    setEditForm(f => ({ ...f, serviceOptionIds: pkgId ? [pkgId] : [] }));
                                    setEditFieldErrors(prev => ({ ...prev, serviceOptionIds: "" }));
                                }}
                            >
                                <option value="">Select a service</option>
                                {editSelectableServices.map(s => <option key={s.serviceId} value={s.serviceId}>{s.serviceName}</option>)}
                            </Form.Select>
                            {!editSelectedService && editFieldErrors.serviceOptionIds && (
                                <div className="text-danger small mt-1">{editFieldErrors.serviceOptionIds}</div>
                            )}
                        </Form.Group>

                        {editSelectedService && (
                            <Form.Group className="mb-3">
                                <Form.Label>Options <span className="text-danger">*</span></Form.Label>
                                {editActiveOptions.length === 0 && (
                                    <div className="text-muted small">This service has no options.</div>
                                )}
                                {editActiveOptions.map(pkg => (
                                    <Form.Check
                                        key={pkg.serviceOptionId}
                                        type="checkbox"
                                        label={pkg.serviceOptionItems.length > 0
                                            ? `${pkg.serviceOptionName} - ${pkg.serviceOptionItems.map(i => i.serviceOptionItemName).join(", ")}`
                                            : pkg.serviceOptionName}
                                        checked={editForm.serviceOptionIds.includes(pkg.serviceOptionId)}
                                        onChange={e => setEditForm(f => ({
                                            ...f,
                                            serviceOptionIds: e.target.checked
                                                ? [...f.serviceOptionIds, pkg.serviceOptionId]
                                                : f.serviceOptionIds.filter(id => id !== pkg.serviceOptionId),
                                        }))}
                                    />
                                ))}
                                {editFieldErrors.serviceOptionIds && <div className="text-danger small">{editFieldErrors.serviceOptionIds}</div>}
                            </Form.Group>
                        )}

                        {lockedStaffId === undefined && (
                            <Form.Group className="mb-3">
                                <Form.Label>Staff</Form.Label>
                                <Form.Select
                                    value={editForm.staffId}
                                    onChange={e => setEditForm(f => ({ ...f, staffId: e.target.value }))}
                                    isInvalid={!!editFieldErrors.staffId}
                                >
                                    <option value="">Owner-managed</option>
                                    {staff.map(s => <option key={s.staffId} value={s.staffId}>{s.name}</option>)}
                                </Form.Select>
                                <Form.Control.Feedback type="invalid">{editFieldErrors.staffId}</Form.Control.Feedback>
                            </Form.Group>
                        )}

                        <Form.Group className="mb-3">
                            <Form.Label>Date <span className="text-danger">*</span></Form.Label>
                            <Form.Control
                                type="date"
                                value={editForm.date ?? ""}
                                min={todayISO()}
                                onChange={e => setEditForm(f => ({ ...f, date: e.target.value, daysOfWeek: [] }))}
                            />
                            {editFieldErrors.schedule && <div className="text-danger small mt-1">{editFieldErrors.schedule}</div>}
                        </Form.Group>

                        {editTimeOptions.length < 2 ? (
                            <Alert variant="warning" className="py-2 small mb-3">
                                {editForm.staffId
                                    ? "This staff has no working hours on the selected date."
                                    : "The business has no working hours on the selected date."}
                            </Alert>
                        ) : (
                            <div className="d-flex gap-3 mb-3">
                                <Form.Group className="flex-fill">
                                    <Form.Label>Start <span className="text-danger">*</span></Form.Label>
                                    <Form.Select
                                        value={editForm.startTime}
                                        onChange={e => {
                                            const newStart = e.target.value;
                                            setEditForm(f => {
                                                const endOpts = editTimeOptions.filter(t => t > newStart);
                                                const end = endOpts.includes(f.endTime) ? f.endTime : (endOpts[0] ?? f.endTime);
                                                return { ...f, startTime: newStart, endTime: end };
                                            });
                                        }}
                                        isInvalid={!!editFieldErrors.startTime}
                                    >
                                        {editTimeOptions.slice(0, -1).map(t => <option key={t} value={t}>{t}</option>)}
                                    </Form.Select>
                                    <Form.Control.Feedback type="invalid">{editFieldErrors.startTime}</Form.Control.Feedback>
                                </Form.Group>
                                <Form.Group className="flex-fill">
                                    <Form.Label>End <span className="text-danger">*</span></Form.Label>
                                    <Form.Select
                                        value={editForm.endTime}
                                        onChange={e => setEditForm(f => ({ ...f, endTime: e.target.value }))}
                                    >
                                        {editTimeOptions.filter(t => t > editForm.startTime).map(t => <option key={t} value={t}>{t}</option>)}
                                    </Form.Select>
                                </Form.Group>
                            </div>
                        )}

                        {editFormError && <Alert variant="danger" className="py-2">{editFormError}</Alert>}
                        <Button
                            variant="primary" size="sm" className="mb-4"
                            onClick={onUpdate}
                            disabled={isEditSaving || editTimeOptions.length < 2}
                        >
                            {isEditSaving ? "Saving..." : "Save changes"}
                        </Button>

                        <hr />
                        <Form.Check
                            type="checkbox"
                            className="mb-2"
                            label="Also delete all future slots of this series, from this date onward"
                            checked={deleteFuture}
                            onChange={e => setDeleteFuture(e.target.checked)}
                        />
                        <Button variant="danger" size="sm" onClick={onDelete} disabled={manageBusy}>
                            {manageBusy ? "Working..." : "Delete slot"}
                        </Button>
                    </>
                ))}
            </Modal.Body>
        </Modal>
    );
}
