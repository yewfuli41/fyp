import { useEffect, useMemo, type Dispatch, type SetStateAction } from "react";
import { Alert, Button, Form, Modal } from "react-bootstrap";
import type { Service } from "../services/ServiceService";
import type { Staff } from "../services/StaffService";
import type { ServiceSlotInput } from "../services/ServiceSlotService";
import { cap, computeTimeOptions, defaultPackageId, todayISO, type WorkingHour } from "../utils/serviceSlotHelpers";
import { DAYS_OF_WEEK } from "../utils/time";

interface AddServiceSlotModalProps {
    show: boolean;
    onHide: () => void;
    formError: string;
    fieldErrors: Record<string, string>;
    setFieldErrors: Dispatch<SetStateAction<Record<string, string>>>;
    form: ServiceSlotInput;
    setForm: Dispatch<SetStateAction<ServiceSlotInput>>;
    formServiceId: string;
    setFormServiceId: Dispatch<SetStateAction<string>>;
    services: Service[];
    staff: Staff[];
    workingHours: WorkingHour[];
    lines: string[];
    onSubmit: () => void;
    isSaving: boolean;
    // When set (a staff member, not the owner, is creating the slot), the
    // Staff field is hidden — the backend forces the slot onto this staff ID
    // regardless of what's in `form.staffId`.
    lockedStaffId?: string;
}

export default function AddServiceSlotModal({
    show, onHide, formError, fieldErrors, setFieldErrors, form, setForm, formServiceId, setFormServiceId,
    services, staff, workingHours, lines, onSubmit, isSaving, lockedStaffId,
}: AddServiceSlotModalProps) {
    const selectedService = services.find(s => s.serviceId === formServiceId);

    const timeOptions = useMemo(
        () => computeTimeOptions(form, staff, workingHours, lines),
        [form.staffId, form.daysOfWeek, form.date, staff, workingHours, lines],
    );

    // Keep the chosen start/end within the currently allowed range. The last
    // time option can never be a valid start (there'd be no room left for an
    // end after it — matches the Start <select>, which excludes it too).
    useEffect(() => {
        if (!show) return;
        const startOptions = timeOptions.slice(0, -1);
        if (startOptions.length === 0) return;
        setForm(f => {
            let start = startOptions.includes(f.startTime) ? f.startTime : startOptions[0];
            const endOpts = timeOptions.filter(t => t > start);
            let end = endOpts.includes(f.endTime) ? f.endTime : endOpts[0];
            if (start === f.startTime && end === f.endTime) return f;
            return { ...f, startTime: start, endTime: end };
        });
    }, [timeOptions, show, setForm]);

    return (
        <Modal show={show} onHide={onHide} backdrop="static">
            <Modal.Header closeButton><Modal.Title>Add Service Slot</Modal.Title></Modal.Header>
            <Modal.Body>
                {formError && <Alert variant="danger">{formError}</Alert>}

                <Form.Group className="mb-3">
                    <Form.Label>Service <span className="text-danger">*</span></Form.Label>
                    <Form.Select
                        value={formServiceId}
                        onChange={e => {
                            const serviceId = e.target.value;
                            setFormServiceId(serviceId);
                            const pkgId = defaultPackageId(services, serviceId);
                            setForm(f => ({ ...f, servicePackageIds: pkgId ? [pkgId] : [] }));
                            setFieldErrors(prev => ({ ...prev, servicePackageIds: "" }));
                        }}
                    >
                        <option value="">Select a service</option>
                        {services.map(s => <option key={s.serviceId} value={s.serviceId}>{s.serviceName}</option>)}
                    </Form.Select>
                    {!selectedService && fieldErrors.servicePackageIds && (
                        <div className="text-danger small mt-1">{fieldErrors.servicePackageIds}</div>
                    )}
                </Form.Group>

                {selectedService && (
                    <Form.Group className="mb-3">
                        <Form.Label>Packages <span className="text-danger">*</span></Form.Label>
                        {selectedService.servicePackages.length === 0 && (
                            <div className="text-muted small">This service has no packages.</div>
                        )}
                        {selectedService.servicePackages.map(pkg => (
                            <Form.Check
                                key={pkg.servicePackageId}
                                type="checkbox"
                                label={pkg.servicePackageName}
                                checked={form.servicePackageIds.includes(pkg.servicePackageId)}
                                onChange={e => setForm(f => ({
                                    ...f,
                                    servicePackageIds: e.target.checked
                                        ? [...f.servicePackageIds, pkg.servicePackageId]
                                        : f.servicePackageIds.filter(id => id !== pkg.servicePackageId),
                                }))}
                            />
                        ))}
                        {fieldErrors.servicePackageIds && <div className="text-danger small">{fieldErrors.servicePackageIds}</div>}
                    </Form.Group>
                )}

                {lockedStaffId === undefined && (
                    <Form.Group className="mb-3">
                        <Form.Label>Staff</Form.Label>
                        <Form.Select
                            value={form.staffId}
                            onChange={e => setForm(f => ({ ...f, staffId: e.target.value }))}
                            isInvalid={!!fieldErrors.staffId}
                        >
                            <option value="">Owner-managed</option>
                            {staff.map(s => <option key={s.staffId} value={s.staffId}>{s.name}</option>)}
                        </Form.Select>
                        <Form.Control.Feedback type="invalid">{fieldErrors.staffId}</Form.Control.Feedback>
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
                                    checked={form.daysOfWeek.includes(day)}
                                    onChange={e => setForm(f => ({
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
                        value={form.date ?? ""}
                        min={todayISO()}
                        disabled={form.daysOfWeek.length > 0}
                        onChange={e => setForm(f => ({ ...f, date: e.target.value, daysOfWeek: [] }))}
                    />
                    {fieldErrors.schedule && <div className="text-danger small mt-1">{fieldErrors.schedule}</div>}
                </Form.Group>

                {timeOptions.length < 2 ? (
                    <Alert variant="warning" className="py-2 small mb-0">
                        {form.staffId
                            ? "This staff has no working hours on the selected day(s)."
                            : "The business has no working hours on the selected day(s)."}
                    </Alert>
                ) : (
                    <div className="d-flex gap-3 mb-3">
                        <Form.Group className="flex-fill">
                            <Form.Label>Start <span className="text-danger">*</span></Form.Label>
                            <Form.Select
                                value={form.startTime}
                                onChange={e => {
                                    const newStart = e.target.value;
                                    setForm(f => {
                                        const endOpts = timeOptions.filter(t => t > newStart);
                                        const end = endOpts.includes(f.endTime) ? f.endTime : (endOpts[0] ?? f.endTime);
                                        return { ...f, startTime: newStart, endTime: end };
                                    });
                                }}
                                isInvalid={!!fieldErrors.startTime}
                            >
                                {timeOptions.slice(0, -1).map(t => <option key={t} value={t}>{t}</option>)}
                            </Form.Select>
                            <Form.Control.Feedback type="invalid">{fieldErrors.startTime}</Form.Control.Feedback>
                        </Form.Group>
                        <Form.Group className="flex-fill">
                            <Form.Label>End <span className="text-danger">*</span></Form.Label>
                            <Form.Select
                                value={form.endTime}
                                onChange={e => setForm(f => ({ ...f, endTime: e.target.value }))}
                            >
                                {timeOptions.filter(t => t > form.startTime).map(t => <option key={t} value={t}>{t}</option>)}
                            </Form.Select>
                        </Form.Group>
                    </div>
                )}
            </Modal.Body>
            <Modal.Footer>
                <Button variant="outline-secondary" onClick={onHide}>Cancel</Button>
                <Button variant="primary" onClick={onSubmit} disabled={isSaving || timeOptions.length < 2}>
                    {isSaving ? "Saving..." : "Save"}
                </Button>
            </Modal.Footer>
        </Modal>
    );
}
