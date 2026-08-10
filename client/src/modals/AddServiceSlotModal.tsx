import { useEffect, useMemo, type Dispatch, type SetStateAction } from "react";
import { Alert, Button, Form, Modal } from "react-bootstrap";
import type { Service } from "../services/ServiceService";
import type { Staff } from "../services/StaffService";
import type { ServiceSlotInput } from "../services/ServiceSlotService";
import {
    cap, computeTimeOptions, defaultOptionId, isOptionSelectableFor, nowHHMM, todayISO, weekdayName,
    type WorkingHour,
} from "../utils/serviceSlotHelpers";
import { DAYS_OF_WEEK } from "../utils/time";

export interface TimeRange {
    startTime: string;
    endTime: string;
}

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
    // Specific dates to create slots for — mutually exclusive with
    // form.daysOfWeek, same as the single date this replaces. Always has at
    // least one entry (possibly blank, not yet picked); every filled-in date
    // gets a slot for every time range below (see CalendarPage's handleCreate).
    dates: string[];
    setDates: Dispatch<SetStateAction<string[]>>;
    // Every time range to create a slot for, for every date above (or, in
    // weekday-recurring mode, one recurring schedule per range). Always has
    // at least one entry.
    ranges: TimeRange[];
    setRanges: Dispatch<SetStateAction<TimeRange[]>>;
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
    dates, setDates, ranges, setRanges, services, staff, workingHours, lines, onSubmit, isSaving, lockedStaffId,
}: AddServiceSlotModalProps) {
    const filledDates = dates.filter(Boolean);

    // An option/service must be valid on EVERY selected date, since all
    // dates share the same serviceOptionIds when the slots are created.
    // Weekday-recurring mode has no concrete dates to check (it spans many
    // future occurrences), so an upcoming/limited-time option stays
    // selectable as long as it hasn't fully expired yet — the backend
    // re-resolves each option per occurrence regardless, only creating
    // occurrences the option actually covers.
    const checkDates = form.daysOfWeek.length > 0 ? [] : (filledDates.length > 0 ? filledDates : [todayISO()]);

    const selectedService = services.find(s => s.serviceId === formServiceId);
    const activeOptions = selectedService?.serviceOptions.filter(
        o => checkDates.every(d => isOptionSelectableFor(o, d, form.daysOfWeek)),
    ) ?? [];
    // A service with no option offered on every target date can't be selected here.
    const selectableServices = services.filter(
        s => s.serviceOptions.some(o => checkDates.every(d => isOptionSelectableFor(o, d, form.daysOfWeek))),
    );

    const timeOptions = useMemo(() => {
        if (form.daysOfWeek.length > 0) return computeTimeOptions(form, staff, workingHours, lines);
        // Multiple dates can span different weekdays with different working
        // hours — reuse computeTimeOptions' weekday-intersection logic by
        // handing it the distinct weekdays those dates fall on.
        const weekdays = Array.from(new Set(checkDates.map(weekdayName)));
        let options = computeTimeOptions({ ...form, date: "", daysOfWeek: weekdays }, staff, workingHours, lines);
        if (checkDates.includes(todayISO())) {
            const now = nowHHMM();
            options = options.filter(t => t > now);
        }
        return options;
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [form.staffId, form.daysOfWeek, dates, staff, workingHours, lines]);

    // Keep every chosen range within the currently allowed window. The last
    // time option can never be a valid start (there'd be no room left for an
    // end after it — matches the Start <select>, which excludes it too).
    useEffect(() => {
        if (!show) return;
        const startOptions = timeOptions.slice(0, -1);
        if (startOptions.length === 0) return;
        setRanges(rs => {
            let changed = false;
            const next = rs.map(r => {
                const start = startOptions.includes(r.startTime) ? r.startTime : startOptions[0];
                const endOpts = timeOptions.filter(t => t > start);
                const end = endOpts.includes(r.endTime) ? r.endTime : endOpts[0];
                if (start === r.startTime && end === r.endTime) return r;
                changed = true;
                return { startTime: start, endTime: end };
            });
            return changed ? next : rs;
        });
    }, [timeOptions, show, setRanges]);

    // A new row defaults to the earliest start time not already used by
    // another row, so it doesn't just look like a copy the user has to edit away.
    const addRange = () => {
        const startOptions = timeOptions.slice(0, -1);
        const usedStarts = new Set(ranges.map(r => r.startTime));
        const start = startOptions.find(t => !usedStarts.has(t)) ?? startOptions[0];
        const endOpts = timeOptions.filter(t => t > start);
        setRanges(rs => [...rs, { startTime: start, endTime: endOpts[0] ?? start }]);
    };

    const totalSlots = (filledDates.length > 0 ? filledDates.length : 1) * ranges.length;

    return (
        <Modal show={show} onHide={onHide} backdrop="static">
            <Modal.Header closeButton><Modal.Title>Add Service Slot</Modal.Title></Modal.Header>
            <Modal.Body>
                <Alert variant="info" className="py-2 small">
                    One slot only accepts one booking. Create additional slots if you want to accept multiple customers at the same time.
                </Alert>

                <Form.Group className="mb-3">
                    <Form.Label>Service <span className="text-danger">*</span></Form.Label>
                    <Form.Select
                        value={formServiceId}
                        onChange={e => {
                            const serviceId = e.target.value;
                            setFormServiceId(serviceId);
                            const pkgId = defaultOptionId(services, serviceId, checkDates[0], form.daysOfWeek);
                            setForm(f => ({ ...f, serviceOptionIds: pkgId ? [pkgId] : [] }));
                            setFieldErrors(prev => ({ ...prev, serviceOptionIds: "" }));
                        }}
                    >
                        <option value="">Select a service</option>
                        {selectableServices.map(s => <option key={s.serviceId} value={s.serviceId}>{s.serviceName}</option>)}
                    </Form.Select>
                    {!selectedService && fieldErrors.serviceOptionIds && (
                        <div className="text-danger small mt-1">{fieldErrors.serviceOptionIds}</div>
                    )}
                </Form.Group>

                {selectedService && (
                    <Form.Group className="mb-3">
                        <Form.Label>Options <span className="text-danger">*</span></Form.Label>
                        {activeOptions.length === 0 && (
                            <div className="text-muted small">This service has no options.</div>
                        )}
                        {activeOptions.map(pkg => (
                            <Form.Check
                                key={pkg.serviceOptionId}
                                type="checkbox"
                                label={pkg.serviceOptionItems.length > 0
                                    ? `${pkg.serviceOptionName} - ${pkg.serviceOptionItems.map(i => i.serviceOptionItemName).join(", ")}`
                                    : pkg.serviceOptionName}
                                checked={form.serviceOptionIds.includes(pkg.serviceOptionId)}
                                onChange={e => setForm(f => ({
                                    ...f,
                                    serviceOptionIds: e.target.checked
                                        ? [...f.serviceOptionIds, pkg.serviceOptionId]
                                        : f.serviceOptionIds.filter(id => id !== pkg.serviceOptionId),
                                }))}
                            />
                        ))}
                        {fieldErrors.serviceOptionIds && <div className="text-danger small">{fieldErrors.serviceOptionIds}</div>}
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
                    <Form.Label className="fw-semibold">Select Days of the Week or Date(s) <span className="text-danger">*</span></Form.Label>
                    <div className="d-flex flex-wrap">
                        {DAYS_OF_WEEK.map(day => (
                            <div key={day} style={{ width: "50%" }} className="mb-1">
                                <Form.Check
                                    type="checkbox"
                                    label={`Every ${cap(day)}`}
                                    checked={form.daysOfWeek.includes(day)}
                                    onChange={e => {
                                        if (e.target.checked) setDates([""]); // choosing weekdays clears the picked dates
                                        setForm(f => ({
                                            ...f,
                                            daysOfWeek: e.target.checked
                                                ? [...f.daysOfWeek, day]
                                                : f.daysOfWeek.filter(d => d !== day),
                                        }));
                                    }}
                                />
                            </div>
                        ))}
                    </div>
                    <div className="text-center fw-bold my-2">OR</div>
                    {dates.map((d, i) => (
                        <div key={i} className="d-flex gap-2 mb-2 align-items-center">
                            <Form.Control
                                type="date"
                                value={d}
                                min={todayISO()}
                                disabled={form.daysOfWeek.length > 0}
                                onChange={e => setDates(ds => ds.map((x, idx) => (idx === i ? e.target.value : x)))}
                            />
                            {dates.length > 1 && (
                                <Button
                                    variant="link"
                                    className="p-0 text-danger"
                                    aria-label="Remove this date"
                                    onClick={() => setDates(ds => ds.filter((_, idx) => idx !== i))}
                                >
                                    Remove
                                </Button>
                            )}
                        </div>
                    ))}
                    <Button
                        variant="link"
                        className="p-0"
                        disabled={form.daysOfWeek.length > 0}
                        onClick={() => setDates(ds => [...ds, ""])}
                    >
                        + Add another date
                    </Button>
                    {fieldErrors.schedule && <div className="text-danger small mt-1">{fieldErrors.schedule}</div>}
                </Form.Group>

                {timeOptions.length < 2 ? (
                    <Alert variant="warning" className="py-2 small mb-0">
                        {form.staffId
                            ? "This staff has no working hours on the selected day(s)."
                            : "The business has no working hours on the selected day(s)."}
                    </Alert>
                ) : (
                    <Form.Group className="mb-3">
                        <Form.Label className="fw-semibold">Time Range(s) <span className="text-danger">*</span></Form.Label>
                        {ranges.map((r, i) => (
                            <div key={i} className="d-flex gap-3 align-items-end mb-2">
                                <Form.Group className="flex-fill">
                                    {i === 0 && <Form.Label className="small">Start</Form.Label>}
                                    <Form.Select
                                        value={r.startTime}
                                        onChange={e => {
                                            const newStart = e.target.value;
                                            setRanges(rs => rs.map((x, idx) => {
                                                if (idx !== i) return x;
                                                const endOpts = timeOptions.filter(t => t > newStart);
                                                const end = endOpts.includes(x.endTime) ? x.endTime : (endOpts[0] ?? x.endTime);
                                                return { startTime: newStart, endTime: end };
                                            }));
                                        }}
                                        isInvalid={i === 0 && !!fieldErrors.startTime}
                                    >
                                        {timeOptions.slice(0, -1).map(t => <option key={t} value={t}>{t}</option>)}
                                    </Form.Select>
                                    {i === 0 && <Form.Control.Feedback type="invalid">{fieldErrors.startTime}</Form.Control.Feedback>}
                                </Form.Group>
                                <Form.Group className="flex-fill">
                                    {i === 0 && <Form.Label className="small">End</Form.Label>}
                                    <Form.Select
                                        value={r.endTime}
                                        onChange={e => setRanges(rs => rs.map((x, idx) => (idx === i ? { ...x, endTime: e.target.value } : x)))}
                                    >
                                        {timeOptions.filter(t => t > r.startTime).map(t => <option key={t} value={t}>{t}</option>)}
                                    </Form.Select>
                                </Form.Group>
                                {ranges.length > 1 && (
                                    <Button
                                        variant="link"
                                        className="p-0 text-danger mb-2"
                                        aria-label="Remove this time range"
                                        onClick={() => setRanges(rs => rs.filter((_, idx) => idx !== i))}
                                    >
                                        Remove
                                    </Button>
                                )}
                            </div>
                        ))}
                        <Button variant="link" className="p-0" onClick={addRange}>
                            + Add another time range
                        </Button>
                    </Form.Group>
                )}

                {totalSlots > 1 && (
                    <Alert variant="info" className="py-2 small">
                        {filledDates.length > 1 && ranges.length > 1
                            ? `A slot will be created for each of the ${filledDates.length} dates × ${ranges.length} time ranges (${totalSlots} slots total).`
                            : filledDates.length > 1
                                ? `A separate slot will be created for each of the ${filledDates.length} selected dates.`
                                : `A separate slot will be created for each of the ${ranges.length} time ranges above.`}
                        {" "}If one fails (e.g. a conflict), slots already created before it are kept.
                    </Alert>
                )}

                {formError && <Alert variant="danger" className="mb-0 mt-3">{formError}</Alert>}
            </Modal.Body>
            <Modal.Footer>
                <Button variant="outline-secondary" onClick={onHide}>Cancel</Button>
                <Button
                    variant="primary"
                    onClick={onSubmit}
                    disabled={isSaving || timeOptions.length < 2 || (form.daysOfWeek.length === 0 && filledDates.length === 0)}
                >
                    {isSaving ? "Saving..." : "Save"}
                </Button>
            </Modal.Footer>
        </Modal>
    );
}
