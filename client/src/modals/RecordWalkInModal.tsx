import { useEffect, useMemo, useState } from "react";
import { Alert, Button, Form, Modal } from "react-bootstrap";
import type { Service } from "../services/ServiceService";
import type { Staff } from "../services/StaffService";
import type { LeaveApplication } from "../services/LeaveService";
import { computeTimeOptions, defaultOptionId, isOptionSelectableFor, todayISO, type WorkingHour } from "../utils/serviceSlotHelpers";

export interface WalkInSubmission {
    serviceOptionId: string;
    staffId: string; // "" => owner-managed
    date: string;
    startTime: string;
    endTime: string;
}

interface RecordWalkInModalProps {
    show: boolean;
    onHide: () => void;
    services: Service[];
    staff: Staff[];
    workingHours: WorkingHour[];
    lines: string[];
    leaves: LeaveApplication[];
    // Default date the picker opens with — the Calendar page's currently
    // selected day — but the user can change it here.
    initialDate: string;
    onSubmit: (input: WalkInSubmission) => void;
    isSaving: boolean;
    formError: string;
    // When set (a staff member, not the owner, is recording the walk-in), the
    // Staff field is hidden — the backend forces it onto this staff ID anyway.
    lockedStaffId?: string;
}

// Trimmed-down sibling of AddServiceSlotModal: a walk-in only ever needs a
// service (one option, not several), who it's for, and a single date + time
// range — same shape as a regular slot, just created and booked in one step.
export default function RecordWalkInModal({
    show, onHide, services, staff, workingHours, lines, leaves, initialDate, onSubmit, isSaving, formError, lockedStaffId,
}: RecordWalkInModalProps) {
    const [serviceId, setServiceId] = useState("");
    const [optionId, setOptionId] = useState("");
    const [staffId, setStaffId] = useState("");
    const [date, setDate] = useState(initialDate);
    const [range, setRange] = useState({ startTime: "", endTime: "" });
    const { startTime, endTime } = range;

    useEffect(() => {
        if (!show) return;
        setServiceId("");
        setOptionId("");
        setStaffId(lockedStaffId ?? "");
        setDate(initialDate);
        setRange({ startTime: "", endTime: "" });
    }, [show, lockedStaffId, initialDate]);

    const selectableServices = services.filter(s => s.serviceOptions.some(o => isOptionSelectableFor(o, date, [])));
    const selectedService = services.find(s => s.serviceId === serviceId);
    const activeOptions = selectedService?.serviceOptions.filter(o => isOptionSelectableFor(o, date, [])) ?? [];

    const handleServiceChange = (id: string) => {
        setServiceId(id);
        setOptionId(defaultOptionId(services, id, date, []) ?? "");
    };

    const timeOptions = useMemo(
        () => computeTimeOptions({ staffId, date, daysOfWeek: [], startTime: "", endTime: "", serviceOptionIds: [] }, staff, workingHours, lines),
        [staffId, date, staff, workingHours, lines],
    );

    // Keep the chosen range within the currently allowed window, same
    // approach as AddServiceSlotModal/ManageServiceSlotModal.
    useEffect(() => {
        if (!show) return;
        const startOptions = timeOptions.slice(0, -1);
        if (startOptions.length === 0) return;
        setRange(r => {
            const start = startOptions.includes(r.startTime) ? r.startTime : startOptions[0];
            const endOpts = timeOptions.filter(t => t > start);
            const end = endOpts.includes(r.endTime) ? r.endTime : (endOpts[0] ?? start);
            if (start === r.startTime && end === r.endTime) return r;
            return { startTime: start, endTime: end };
        });
    }, [timeOptions, show]);

    // The effective staff being booked for — already reflects lockedStaffId
    // via the reset effect above, so this covers both the owner and staff cases.
    const onLeave = !!staffId && leaves.some(l =>
        l.staffId === staffId && l.status === "APPROVED" && l.startDate <= date && l.endDate >= date);

    const canSubmit = !!optionId && timeOptions.length >= 2 && !onLeave;

    const handleSubmit = () => {
        if (!canSubmit) return;
        onSubmit({
            serviceOptionId: optionId,
            staffId: lockedStaffId ?? staffId,
            date,
            startTime,
            endTime,
        });
    };

    return (
        <Modal show={show} onHide={onHide} backdrop="static">
            <Modal.Header closeButton><Modal.Title>Record Walk-In</Modal.Title></Modal.Header>
            <Modal.Body>
                <Form.Group className="mb-3">
                    <Form.Label>Service <span className="text-danger">*</span></Form.Label>
                    <Form.Select value={serviceId} onChange={e => handleServiceChange(e.target.value)}>
                        <option value="">Select a service</option>
                        {selectableServices.map(s => <option key={s.serviceId} value={s.serviceId}>{s.serviceName}</option>)}
                    </Form.Select>
                </Form.Group>

                {selectedService && (
                    <Form.Group className="mb-3">
                        <Form.Label>Option <span className="text-danger">*</span></Form.Label>
                        {activeOptions.length === 0 && (
                            <div className="text-muted small">This service has no options.</div>
                        )}
                        {activeOptions.map(opt => (
                            <Form.Check
                                key={opt.serviceOptionId}
                                type="radio"
                                name="walkInOption"
                                label={opt.serviceOptionItems.length > 0
                                    ? `${opt.serviceOptionName} - ${opt.serviceOptionItems.map(i => i.serviceOptionItemName).join(", ")}`
                                    : opt.serviceOptionName}
                                checked={optionId === opt.serviceOptionId}
                                onChange={() => setOptionId(opt.serviceOptionId)}
                            />
                        ))}
                    </Form.Group>
                )}

                {lockedStaffId === undefined && (
                    <Form.Group className="mb-3">
                        <Form.Label>Staff</Form.Label>
                        <Form.Select value={staffId} onChange={e => setStaffId(e.target.value)}>
                            <option value="">Owner-managed</option>
                            {staff.map(s => <option key={s.staffId} value={s.staffId}>{s.name}</option>)}
                        </Form.Select>
                    </Form.Group>
                )}

                <Form.Group className="mb-3">
                    <Form.Label>Date <span className="text-danger">*</span></Form.Label>
                    <Form.Control
                        type="date"
                        value={date}
                        min={todayISO()}
                        onChange={e => setDate(e.target.value)}
                    />
                </Form.Group>

                {onLeave ? (
                    <Alert variant="warning" className="py-2 small mb-0">
                        This staff is on approved leave on the selected date — pick a different date or staff.
                    </Alert>
                ) : timeOptions.length < 2 ? (
                    <Alert variant="warning" className="py-2 small mb-0">
                        {staffId
                            ? "This staff has no working hours on the selected date."
                            : "The business has no working hours on the selected date."}
                    </Alert>
                ) : (
                    <div className="d-flex gap-3 mb-3">
                        <Form.Group className="flex-fill">
                            <Form.Label>Start <span className="text-danger">*</span></Form.Label>
                            <Form.Select
                                value={startTime}
                                onChange={e => {
                                    const newStart = e.target.value;
                                    const endOpts = timeOptions.filter(t => t > newStart);
                                    setRange(r => ({ startTime: newStart, endTime: endOpts.includes(r.endTime) ? r.endTime : (endOpts[0] ?? newStart) }));
                                }}
                            >
                                {timeOptions.slice(0, -1).map(t => <option key={t} value={t}>{t}</option>)}
                            </Form.Select>
                        </Form.Group>
                        <Form.Group className="flex-fill">
                            <Form.Label>End <span className="text-danger">*</span></Form.Label>
                            <Form.Select value={endTime} onChange={e => setRange(r => ({ ...r, endTime: e.target.value }))}>
                                {timeOptions.filter(t => t > startTime).map(t => <option key={t} value={t}>{t}</option>)}
                            </Form.Select>
                        </Form.Group>
                    </div>
                )}

                {formError && <Alert variant="danger" className="mb-0">{formError}</Alert>}
            </Modal.Body>
            <Modal.Footer>
                <Button variant="outline-secondary" onClick={onHide}>Cancel</Button>
                <Button variant="primary" onClick={handleSubmit} disabled={isSaving || !canSubmit}>
                    {isSaving ? "Saving..." : "Save"}
                </Button>
            </Modal.Footer>
        </Modal>
    );
}
