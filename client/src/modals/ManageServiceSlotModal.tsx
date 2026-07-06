import { useEffect, useMemo, type Dispatch, type SetStateAction } from "react";
import { Alert, Button, Form, Modal } from "react-bootstrap";
import type { Service } from "../services/ServiceService";
import type { Staff } from "../services/StaffService";
import { extractTime, type ServiceSlot, type ServiceSlotInput } from "../services/ServiceSlotService";
import { cap, computeTimeOptions, defaultOptionId, todayISO, type WorkingHour } from "../utils/serviceSlotHelpers";
import { DAYS_OF_WEEK } from "../utils/time";

interface ManageServiceSlotModalProps {
    show: boolean;
    onHide: () => void;
    managing: ServiceSlot | null;
    manageError: string;
    services: Service[];
    staff: Staff[];
    workingHours: WorkingHour[];
    lines: string[];

    // Reassign staff (only surfaced once the slot has a booking)
    reassignTo: string;
    setReassignTo: Dispatch<SetStateAction<string>>;
    availableStaff: { staffId: string; name: string }[];
    onReassign: () => void;
    manageBusy: boolean;

    // Full edit (date/days-of-week, time, staff, packages) — only while unbooked
    editForm: ServiceSlotInput;
    setEditForm: Dispatch<SetStateAction<ServiceSlotInput>>;
    editFormServiceId: string;
    setEditFormServiceId: Dispatch<SetStateAction<string>>;
    editFormError: string;
    editFieldErrors: Record<string, string>;
    setEditFieldErrors: Dispatch<SetStateAction<Record<string, string>>>;
    editApplyFuture: boolean;
    setEditApplyFuture: Dispatch<SetStateAction<boolean>>;
    onUpdate: () => void;
    isEditSaving: boolean;

    // Delete
    deleteFuture: boolean;
    setDeleteFuture: Dispatch<SetStateAction<boolean>>;
    onDelete: () => void;

    // When set (a staff member, not the owner, is managing the slot), the
    // Staff field is hidden in the edit form — the backend forces the slot
    // to stay on this staff ID regardless of what's in `editForm.staffId`.
    lockedStaffId?: string;
    // Reassigning to someone else on the team is an owner-only action —
    // false hides the whole reassign section for a staff member.
    allowReassign?: boolean;
}

export default function ManageServiceSlotModal({
    show, onHide, managing, manageError, services, staff, workingHours, lines,
    reassignTo, setReassignTo, availableStaff, onReassign, manageBusy,
    editForm, setEditForm, editFormServiceId, setEditFormServiceId, editFormError,
    editFieldErrors, setEditFieldErrors, editApplyFuture, setEditApplyFuture, onUpdate, isEditSaving,
    deleteFuture, setDeleteFuture, onDelete, lockedStaffId, allowReassign = true,
}: ManageServiceSlotModalProps) {
    const editSelectedService = services.find(s => s.serviceId === editFormServiceId);

    const editTimeOptions = useMemo(
        () => computeTimeOptions(editForm, staff, workingHours, lines),
        [editForm.staffId, editForm.daysOfWeek, editForm.date, staff, workingHours, lines],
    );

    // Keep the chosen start/end within the currently allowed range (see the
    // matching effect on the Add form for why the last option is excluded).
    useEffect(() => {
        if (!managing || managing.hasBooking) return;
        const startOptions = editTimeOptions.slice(0, -1);
        if (startOptions.length === 0) return;
        setEditForm(f => {
            let start = startOptions.includes(f.startTime) ? f.startTime : startOptions[0];
            const endOpts = editTimeOptions.filter(t => t > start);
            let end = endOpts.includes(f.endTime) ? f.endTime : endOpts[0];
            if (start === f.startTime && end === f.endTime) return f;
            return { ...f, startTime: start, endTime: end };
        });
    }, [editTimeOptions, managing, setEditForm]);

    return (
        <Modal show={show} onHide={onHide} size="lg" backdrop="static">
            <Modal.Header closeButton><Modal.Title>Manage Service Slot</Modal.Title></Modal.Header>
            <Modal.Body>
                {manageError && <Alert variant="danger">{manageError}</Alert>}
                {managing && (managing.hasBooking ? (
                    <>
                        <p className="mb-1"><strong>Date:</strong> {managing.date}</p>
                        <p className="mb-1"><strong>Time:</strong> {extractTime(managing.startTime)}–{extractTime(managing.endTime)}</p>
                        <p className="mb-1"><strong>Staff:</strong> {managing.staff?.name ?? "—"}</p>
                        <p className="mb-3"><strong>Options:</strong>{" "}
                            {managing.serviceSlotOptions.map(p => p.serviceOption.serviceOptionName).join(", ") || "—"}
                        </p>

                        {allowReassign ? (
                            <>
                                <Alert variant="info" className="py-2 small">
                                    This slot already has a booking — date, time, staff and options can no longer be
                                    changed. You can still reassign staff below.
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
                                <Button variant="primary" size="sm" onClick={onReassign} disabled={!reassignTo || manageBusy}>
                                    Save reassignment
                                </Button>
                            </>
                        ) : (
                            <Alert variant="info" className="py-2 small mb-0">
                                This slot already has a booking — contact the business owner to change or reassign it.
                            </Alert>
                        )}
                    </>
                ) : (
                    <>
                        {editFormError && <Alert variant="danger">{editFormError}</Alert>}

                        <Form.Group className="mb-3">
                            <Form.Label>Service <span className="text-danger">*</span></Form.Label>
                            <Form.Select
                                value={editFormServiceId}
                                onChange={e => {
                                    const serviceId = e.target.value;
                                    setEditFormServiceId(serviceId);
                                    const pkgId = defaultOptionId(services, serviceId);
                                    setEditForm(f => ({ ...f, serviceOptionIds: pkgId ? [pkgId] : [] }));
                                    setEditFieldErrors(prev => ({ ...prev, serviceOptionIds: "" }));
                                }}
                            >
                                <option value="">Select a service</option>
                                {services.map(s => <option key={s.serviceId} value={s.serviceId}>{s.serviceName}</option>)}
                            </Form.Select>
                            {!editSelectedService && editFieldErrors.serviceOptionIds && (
                                <div className="text-danger small mt-1">{editFieldErrors.serviceOptionIds}</div>
                            )}
                        </Form.Group>

                        {editSelectedService && (
                            <Form.Group className="mb-3">
                                <Form.Label>Options <span className="text-danger">*</span></Form.Label>
                                {editSelectedService.serviceOptions.length === 0 && (
                                    <div className="text-muted small">This service has no options.</div>
                                )}
                                {editSelectedService.serviceOptions.map(pkg => (
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
                            <Form.Label className="fw-semibold">Select Days of the Week or Date <span className="text-danger">*</span></Form.Label>
                            <div className="d-flex flex-wrap">
                                {DAYS_OF_WEEK.map(day => (
                                    <div key={day} style={{ width: "50%" }} className="mb-1">
                                        <Form.Check
                                            type="checkbox"
                                            label={`Every ${cap(day)}`}
                                            checked={editForm.daysOfWeek.includes(day)}
                                            onChange={e => setEditForm(f => ({
                                                ...f,
                                                date: "", // choosing weekdays clears the single date
                                                daysOfWeek: e.target.checked
                                                    ? [...f.daysOfWeek, day]
                                                    : f.daysOfWeek.filter(d => d !== day),
                                            }))}
                                        />
                                    </div>
                                ))}
                            </div>
                            <div className="text-center fw-bold my-2">OR</div>
                            <Form.Control
                                type="date"
                                value={editForm.date ?? ""}
                                min={todayISO()}
                                disabled={editForm.daysOfWeek.length > 0}
                                onChange={e => setEditForm(f => ({ ...f, date: e.target.value, daysOfWeek: [] }))}
                            />
                            {editFieldErrors.schedule && <div className="text-danger small mt-1">{editFieldErrors.schedule}</div>}
                        </Form.Group>

                        {editTimeOptions.length < 2 ? (
                            <Alert variant="warning" className="py-2 small mb-3">
                                {editForm.staffId
                                    ? "This staff has no working hours on the selected day(s)."
                                    : "The business has no working hours on the selected day(s)."}
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

                        <Form.Check
                            type="checkbox"
                            className="mb-3"
                            label="Also apply this change to all future slots on this weekday (if any exist)"
                            checked={editApplyFuture}
                            onChange={e => setEditApplyFuture(e.target.checked)}
                        />
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
                            label="Also delete all future slots on this weekday (if any exist)"
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
