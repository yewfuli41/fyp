import { useState, useEffect, useCallback, useMemo } from "react";
import { Alert, Button, Container, Form, Modal, Spinner } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { userProfile } from "../services/ProfileService";
import { getBusinessStaff, type Staff } from "../services/StaffService";
import { getBusinessServices, type Service } from "../services/ServiceService";
import {
    getServiceSlots, createServiceSlot, reassignServiceSlotStaff, deleteServiceSlot,
    getAvailableStaffForSlot, extractTime,
    type ServiceSlot, type ServiceSlotInput,
} from "../services/ServiceSlotService";
import { applyGraphQLErrors, parseGraphQLErrors } from "../utils/graphqlErrors";
import { DAYS_OF_WEEK } from "../utils/time";

const ROW_HEIGHT = 48; // px per 30-min interval
const DEFAULT_START = "09:00";
const DEFAULT_END = "18:00";
const NONE_COL = "__none__"; // calendar column for owner-managed (unassigned) slots

// ── date / time helpers ──────────────────────────────────────────────────────
const toISO = (d: Date) =>
    `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;

const displayDate = (d: Date) =>
    d.toLocaleDateString("en-GB", { weekday: "short", day: "2-digit", month: "short", year: "numeric" });

const addDays = (d: Date, n: number) => {
    const next = new Date(d);
    next.setDate(next.getDate() + n);
    return next;
};

// 30-min time lines between start and end, inclusive of end.
const timeLines = (start: string, end: string): string[] => {
    const [sh, sm] = start.split(":").map(Number);
    const [eh, em] = end.split(":").map(Number);
    const startMin = sh * 60 + sm;
    const endMin = eh * 60 + em;
    const lines: string[] = [];
    for (let m = startMin; m <= endMin; m += 30) {
        lines.push(`${String(Math.floor(m / 60)).padStart(2, "0")}:${String(m % 60).padStart(2, "0")}`);
    }
    return lines;
};

interface WorkingHour { day: string; startTime: string; endTime: string }

const cap = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

// weekday name (lowercase) of a YYYY-MM-DD date, timezone-safe.
const weekdayName = (date: string): string => {
    const d = new Date(`${date}T00:00:00`);
    return DAYS_OF_WEEK[(d.getDay() + 6) % 7]; // getDay: 0=Sun → our array is Mon-first
};

const emptyForm = (date: string): ServiceSlotInput => ({
    staffId: "", date, daysOfWeek: [], startTime: DEFAULT_START, endTime: DEFAULT_END,
    servicePackageIds: [],
});

export default function CalendarPage() {
    const navigate = useNavigate();
    const { token } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [hasBusiness, setHasBusiness] = useState<boolean | null>(null);
    const [workingHours, setWorkingHours] = useState<WorkingHour[]>([]);
    const [staff, setStaff] = useState<Staff[]>([]);
    const [services, setServices] = useState<Service[]>([]);
    const [slots, setSlots] = useState<ServiceSlot[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [pageError, setPageError] = useState("");

    const [selectedDate, setSelectedDate] = useState<Date>(new Date());
    const [staffFilter, setStaffFilter] = useState("");
    const [serviceFilter, setServiceFilter] = useState("");

    // Add modal
    const [showAdd, setShowAdd] = useState(false);
    const [form, setForm] = useState<ServiceSlotInput>(emptyForm(toISO(new Date())));
    const [formServiceId, setFormServiceId] = useState("");
    const [formError, setFormError] = useState("");
    const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
    const [isSaving, setIsSaving] = useState(false);

    // Manage modal
    const [managing, setManaging] = useState<ServiceSlot | null>(null);
    const [availableStaff, setAvailableStaff] = useState<{ staffId: string; name: string }[]>([]);
    const [reassignTo, setReassignTo] = useState("");
    const [deleteFuture, setDeleteFuture] = useState(false);
    const [manageError, setManageError] = useState("");
    const [manageBusy, setManageBusy] = useState(false);

    const isoDate = toISO(selectedDate);

    const fetchSlots = useCallback(async () => {
        if (!activeToken) return;
        const unassignedOnly = staffFilter === NONE_COL;
        const result = await getServiceSlots(
            activeToken,
            toISO(selectedDate),
            unassignedOnly ? undefined : (staffFilter || undefined),
            serviceFilter || undefined,
            unassignedOnly || undefined,
        );
        if (result.errors?.length) setPageError(result.errors[0].message);
        else setSlots(result.data?.displayServiceSlots ?? []);
    }, [activeToken, selectedDate, staffFilter, serviceFilter]);

    useEffect(() => {
        if (!activeToken) return;
        const init = async () => {
            try {
                const profileResult = await userProfile(activeToken);
                const business = profileResult.data?.userProfile?.businessProfile;
                setHasBusiness(!!business);
                if (business) {
                    setWorkingHours(business.workingHours ?? []);
                    const [staffRes, svcRes] = await Promise.all([
                        getBusinessStaff(activeToken),
                        getBusinessServices(activeToken),
                    ]);
                    setStaff(staffRes.data?.displayStaff ?? []);
                    setServices(svcRes.data?.displayServices ?? []);
                }
            } catch {
                setPageError("Failed to load calendar.");
            } finally {
                setIsLoading(false);
            }
        };
        init();
    }, [activeToken]);

    useEffect(() => {
        if (hasBusiness) fetchSlots();
    }, [hasBusiness, fetchSlots]);

    // Business-hours envelope across all days → calendar time range + picker bounds.
    const [rangeStart, rangeEnd] = useMemo(() => {
        if (workingHours.length === 0) return [DEFAULT_START, DEFAULT_END];
        let min = "23:59", max = "00:00";
        for (const wh of workingHours) {
            const s = extractTime(wh.startTime), e = extractTime(wh.endTime);
            if (s < min) min = s;
            if (e > max) max = e;
        }
        return [min, max];
    }, [workingHours]);

    const lines = useMemo(() => timeLines(rangeStart, rangeEnd), [rangeStart, rangeEnd]);
    const intervals = Math.max(lines.length - 1, 1);

    // Calendar columns: filtered staff, only the Unassigned column, or all staff +
    // an Unassigned column (owner-managed slots) when viewing everyone.
    const staffColumns: { staffId: string; name: string }[] =
        staffFilter === NONE_COL
            ? [{ staffId: NONE_COL, name: "Unassigned" }]
            : staffFilter
                ? staff.filter(s => s.staffId === staffFilter).map(s => ({ staffId: s.staffId, name: s.name }))
                : [...staff.map(s => ({ staffId: s.staffId, name: s.name })), { staffId: NONE_COL, name: "Unassigned" }];

    // ── Add helpers ───────────────────────────────────────────────────────────
    const openAdd = (prefill?: Partial<ServiceSlotInput>) => {
        setForm({ ...emptyForm(isoDate), startTime: rangeStart, endTime: lines[1] ?? rangeEnd, ...prefill });
        setFormServiceId("");
        setFormError("");
        setFieldErrors({});
        setShowAdd(true);
    };

    const selectedService = services.find(s => s.serviceId === formServiceId);

    // Allowed slot times = working-hours envelope of the chosen staff (or the
    // business, for owner-managed) across the selected weekday(s) / date.
    const timeOptions = useMemo(() => {
        const days = form.daysOfWeek.length > 0
            ? form.daysOfWeek
            : (form.date ? [weekdayName(form.date)] : []);
        const source: WorkingHour[] = form.staffId
            ? (staff.find(s => s.staffId === form.staffId)?.workingHours ?? [])
            : workingHours;
        const relevant = source.filter(wh => days.length === 0 || days.includes(wh.day));
        if (relevant.length === 0) return days.length === 0 ? lines : [];
        let min = "23:59", max = "00:00";
        for (const wh of relevant) {
            const s = extractTime(wh.startTime), e = extractTime(wh.endTime);
            if (s < min) min = s;
            if (e > max) max = e;
        }
        return timeLines(min, max);
    }, [form.staffId, form.daysOfWeek, form.date, staff, workingHours, lines]);

    // Keep the chosen start/end within the currently allowed range.
    useEffect(() => {
        if (!showAdd || timeOptions.length === 0) return;
        setForm(f => {
            let start = timeOptions.includes(f.startTime) ? f.startTime : timeOptions[0];
            const endOpts = timeOptions.filter(t => t > start);
            let end = endOpts.includes(f.endTime) ? f.endTime : (endOpts[0] ?? timeOptions[timeOptions.length - 1]);
            if (start === f.startTime && end === f.endTime) return f;
            return { ...f, startTime: start, endTime: end };
        });
    }, [timeOptions, showAdd]);

    const handleCreate = async () => {
        if (!activeToken) return;
        setIsSaving(true);
        setFormError("");
        setFieldErrors({});
        try {
            const result = await createServiceSlot(activeToken, form);
            if (applyGraphQLErrors(result, { setFieldErrors, setFormError, fallbackMessage: "Failed to create slot" })) return;
            setShowAdd(false);
            fetchSlots();
        } catch {
            setFormError("Something went wrong. Please try again.");
        } finally {
            setIsSaving(false);
        }
    };

    // ── Manage helpers ──────────────────────────────────────────────────────────
    const openManage = async (slot: ServiceSlot) => {
        setManaging(slot);
        setReassignTo("");
        setDeleteFuture(false);
        setManageError("");
        if (activeToken) {
            const res = await getAvailableStaffForSlot(activeToken, slot.serviceSlotId);
            setAvailableStaff(res.data?.availableStaffForSlot ?? []);
        }
    };

    const handleReassign = async () => {
        if (!activeToken || !managing || !reassignTo) return;
        setManageBusy(true);
        setManageError("");
        try {
            const target = reassignTo === "__unassign__" ? null : reassignTo;
            const result = await reassignServiceSlotStaff(activeToken, managing.serviceSlotId, target);
            const parsed = parseGraphQLErrors(result, "Failed to reassign staff");
            if (parsed.hasErrors) {
                setManageError(parsed.formError || Object.values(parsed.fieldErrors)[0] || "Failed to reassign staff");
                return;
            }
            setManaging(null);
            fetchSlots();
        } catch {
            setManageError("Something went wrong. Please try again.");
        } finally {
            setManageBusy(false);
        }
    };

    const handleDelete = async () => {
        if (!activeToken || !managing) return;
        setManageBusy(true);
        setManageError("");
        try {
            const result = await deleteServiceSlot(activeToken, managing.serviceSlotId, deleteFuture);
            if (result.errors?.length) {
                setManageError(result.errors[0].message);
                return;
            }
            setManaging(null);
            fetchSlots();
        } catch {
            setManageError("Something went wrong. Please try again.");
        } finally {
            setManageBusy(false);
        }
    };

    // ── render ────────────────────────────────────────────────────────────────
    if (isLoading) {
        return (
            <Container className="d-flex justify-content-center align-items-center" style={{ minHeight: "50vh" }}>
                <Spinner animation="border" variant="primary" />
            </Container>
        );
    }

    if (!hasBusiness) {
        return (
            <Container className="py-5">
                <h1 className="mb-3 fs-1">Calendar</h1>
                <Alert variant="warning" style={{ maxWidth: 1200 }}>
                    You need to register a business profile before managing service slots.
                </Alert>
                <Button variant="primary" onClick={() => navigate("/register-business")}>Register Business Profile</Button>
            </Container>
        );
    }

    return (
        <Container fluid className="py-4 px-4">
            <Button variant="link" className="px-0 mb-2 text-decoration-none" onClick={() => navigate("/services")}>
                &larr; Back to Services
            </Button>

            <div className="d-flex flex-wrap justify-content-between align-items-center mb-4 gap-3">
                <h1 className="fs-2 fw-bold mb-0">Calendar View</h1>
                <Button variant="primary" onClick={() => openAdd()}>Add Service Slots</Button>
            </div>

            <div className="d-flex flex-wrap align-items-end gap-4 mb-4">
                <div className="d-flex align-items-center gap-2">
                    <Button variant="outline-secondary" size="sm" onClick={() => setSelectedDate(addDays(selectedDate, -1))}>&lt;</Button>
                    <div className="fw-semibold" style={{ minWidth: 150, textAlign: "center" }}>{displayDate(selectedDate)}</div>
                    <Button variant="outline-secondary" size="sm" onClick={() => setSelectedDate(addDays(selectedDate, 1))}>&gt;</Button>
                </div>
                <div>
                    <Form.Label className="fw-semibold mb-1 small">Select Staff</Form.Label>
                    <Form.Select value={staffFilter} onChange={e => setStaffFilter(e.target.value)} style={{ minWidth: 200 }}>
                        <option value="">All Staffs</option>
                        <option value={NONE_COL}>Unassigned (owner-managed)</option>
                        {staff.map(s => <option key={s.staffId} value={s.staffId}>{s.name}</option>)}
                    </Form.Select>
                </div>
                <div>
                    <Form.Label className="fw-semibold mb-1 small">Select Service</Form.Label>
                    <Form.Select value={serviceFilter} onChange={e => setServiceFilter(e.target.value)} style={{ minWidth: 200 }}>
                        <option value="">All Services</option>
                        {services.map(s => <option key={s.serviceId} value={s.serviceId}>{s.serviceName}</option>)}
                    </Form.Select>
                </div>
            </div>

            {pageError && <Alert variant="danger">{pageError}</Alert>}

            {staffColumns.length === 0 ? (
                <Alert variant="info">No staff to display. Add staff to start scheduling slots.</Alert>
            ) : (
                <div style={{
                    overflow: "auto",
                    maxHeight: "calc(100vh - 260px)",
                    minHeight: 320,
                    border: "1px solid #dee2e6",
                    borderRadius: 8,
                }}>
                    <div
                        style={{
                            display: "grid",
                            gridTemplateColumns: `70px repeat(${staffColumns.length}, minmax(150px, 1fr))`,
                            gridTemplateRows: `42px repeat(${intervals}, ${ROW_HEIGHT}px)`,
                            minWidth: 70 + staffColumns.length * 150,
                        }}
                    >
                        {/* top-left corner (sticky on both axes) */}
                        <div style={{
                            gridColumn: 1, gridRow: 1, position: "sticky", top: 0, left: 0, zIndex: 4,
                            background: "#fff", borderBottom: "1px solid #dee2e6", borderRight: "1px solid #eee",
                        }} />

                        {/* staff header row (sticky to top while scrolling) */}
                        {staffColumns.map((s, c) => (
                            <div key={s.staffId} style={{
                                gridColumn: c + 2, gridRow: 1, position: "sticky", top: 0, zIndex: 3,
                                background: "#fff", borderBottom: "1px solid #dee2e6", borderLeft: "1px solid #eee",
                                display: "flex", alignItems: "center", justifyContent: "center", fontWeight: 600,
                            }}>
                                {s.name}
                            </div>
                        ))}

                        {/* time labels (sticky to left while scrolling) */}
                        {lines.slice(0, intervals).map((t, i) => (
                            <div key={t} style={{
                                gridColumn: 1, gridRow: i + 2, position: "sticky", left: 0, zIndex: 2,
                                background: "#fff", borderRight: "1px solid #eee", paddingTop: 2, paddingRight: 6,
                                textAlign: "right", fontSize: 12, color: "#6c757d",
                            }}>
                                {t}
                            </div>
                        ))}

                        {/* background cells (clickable to add) */}
                        {staffColumns.map((s, c) =>
                            lines.slice(0, intervals).map((t, i) => (
                                <div
                                    key={`${s.staffId}-${t}`}
                                    onClick={() => openAdd({ staffId: s.staffId === NONE_COL ? "" : s.staffId, startTime: t, endTime: lines[i + 1] ?? rangeEnd })}
                                    style={{
                                        gridColumn: c + 2, gridRow: i + 2,
                                        borderTop: "1px solid #f0f0f0", borderLeft: "1px solid #eee", cursor: "pointer",
                                    }}
                                />
                            ))
                        )}

                        {/* slot blocks */}
                        {slots.map(slot => {
                            const colId = slot.staff ? slot.staff.staffId : NONE_COL;
                            const colIdx = staffColumns.findIndex(s => s.staffId === colId);
                            if (colIdx < 0) return null;
                            const start = extractTime(slot.startTime);
                            const end = extractTime(slot.endTime);
                            let startLine = lines.indexOf(start);
                            let endLine = lines.indexOf(end);
                            if (startLine < 0) startLine = 0;
                            if (endLine < 0) endLine = lines.length - 1;
                            const pkg = slot.serviceSlotPackages[0];
                            const label = pkg
                                ? `${pkg.servicePackage.service.serviceName} — ${pkg.servicePackage.servicePackageName}`
                                : "Slot";
                            return (
                                <div
                                    key={slot.serviceSlotId}
                                    onClick={() => openManage(slot)}
                                    style={{
                                        gridColumn: colIdx + 2,
                                        gridRow: `${startLine + 2} / ${endLine + 2}`,
                                        margin: 2, padding: "4px 6px", borderRadius: 6, cursor: "pointer",
                                        background: "linear-gradient(135deg,#4f9ad6,#2f7cc0)", color: "#fff",
                                        fontSize: 12, overflow: "hidden", zIndex: 1,
                                    }}
                                >
                                    <div style={{ fontWeight: 600 }}>{start}–{end}</div>
                                    <div>{label}</div>
                                    {slot.serviceSlotPackages.length > 1 && (
                                        <div style={{ opacity: 0.85 }}>+{slot.serviceSlotPackages.length - 1} more</div>
                                    )}
                                </div>
                            );
                        })}
                    </div>
                </div>
            )}

            {/* ── Add Service Slots modal ─────────────────────────────────────── */}
            <Modal show={showAdd} onHide={() => setShowAdd(false)} backdrop="static">
                <Modal.Header closeButton><Modal.Title>Add Service Slot</Modal.Title></Modal.Header>
                <Modal.Body>
                    {formError && <Alert variant="danger">{formError}</Alert>}

                    <Form.Group className="mb-3">
                        <Form.Label>Service <span className="text-danger">*</span></Form.Label>
                        <Form.Select
                            value={formServiceId}
                            onChange={e => { setFormServiceId(e.target.value); setForm(f => ({ ...f, servicePackageIds: [] })); }}
                        >
                            <option value="">Select a service</option>
                            {services.map(s => <option key={s.serviceId} value={s.serviceId}>{s.serviceName}</option>)}
                        </Form.Select>
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

                    <Form.Group className="mb-3">
                        <Form.Label>Staff</Form.Label>
                        <Form.Select
                            value={form.staffId}
                            onChange={e => setForm(f => ({ ...f, staffId: e.target.value }))}
                            isInvalid={!!fieldErrors.staffId}
                        >
                            <option value="">Unassigned (owner-managed)</option>
                            {staff.map(s => <option key={s.staffId} value={s.staffId}>{s.name}</option>)}
                        </Form.Select>
                        <Form.Control.Feedback type="invalid">{fieldErrors.staffId}</Form.Control.Feedback>
                    </Form.Group>

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
                            disabled={form.daysOfWeek.length > 0}
                            onChange={e => setForm(f => ({ ...f, date: e.target.value, daysOfWeek: [] }))}
                        />
                        {fieldErrors.schedule && <div className="text-danger small mt-1">{fieldErrors.schedule}</div>}
                    </Form.Group>

                    {timeOptions.length === 0 ? (
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
                                    onChange={e => setForm(f => ({ ...f, startTime: e.target.value }))}
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
                    <Button variant="outline-secondary" onClick={() => setShowAdd(false)}>Cancel</Button>
                    <Button variant="primary" onClick={handleCreate} disabled={isSaving || timeOptions.length === 0}>
                        {isSaving ? "Saving..." : "Save"}
                    </Button>
                </Modal.Footer>
            </Modal>

            {/* ── Manage slot modal ──────────────────────────────────────────── */}
            <Modal show={!!managing} onHide={() => setManaging(null)}>
                <Modal.Header closeButton><Modal.Title>Manage Service Slot</Modal.Title></Modal.Header>
                <Modal.Body>
                    {manageError && <Alert variant="danger">{manageError}</Alert>}
                    {managing && (
                        <>
                            <p className="mb-1"><strong>Date:</strong> {managing.date}</p>
                            <p className="mb-1"><strong>Time:</strong> {extractTime(managing.startTime)}–{extractTime(managing.endTime)}</p>
                            <p className="mb-1"><strong>Staff:</strong> {managing.staff?.name ?? "—"}</p>
                            <p className="mb-3"><strong>Packages:</strong>{" "}
                                {managing.serviceSlotPackages.map(p => p.servicePackage.servicePackageName).join(", ") || "—"}
                            </p>

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
                            <Button variant="primary" size="sm" className="mb-4" onClick={handleReassign} disabled={!reassignTo || manageBusy}>
                                Save reassignment
                            </Button>

                            <hr />
                            <Form.Check
                                type="checkbox"
                                className="mb-2"
                                label="Also delete all future slots on this weekday"
                                checked={deleteFuture}
                                onChange={e => setDeleteFuture(e.target.checked)}
                            />
                            <Button variant="danger" size="sm" onClick={handleDelete} disabled={manageBusy}>
                                {manageBusy ? "Working..." : "Delete slot"}
                            </Button>
                        </>
                    )}
                </Modal.Body>
            </Modal>
        </Container>
    );
}
