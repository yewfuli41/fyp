import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { Alert, Container, Spinner, Button } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { userProfile } from "../services/ProfileService";
import { getBusinessStaff, type Staff } from "../services/StaffService";
import { getBusinessServices, type Service } from "../services/ServiceService";
import {
    getServiceSlots, createServiceSlot, updateServiceSlot, reassignServiceSlotStaff, deleteServiceSlot,
    getAvailableStaffForSlot, extractTime,
    type ServiceSlot, type ServiceSlotInput,
} from "../services/ServiceSlotService";
import { applyGraphQLErrors, parseGraphQLErrors } from "../utils/graphqlErrors";
import {
    DEFAULT_START, DEFAULT_END, NONE_COL, addDays, displayDate, emptyForm, timeLines, toISO,
    type WorkingHour,
} from "../utils/serviceSlotHelpers";
import AddServiceSlotModal from "../modals/AddServiceSlotModal";
import ManageServiceSlotModal from "../modals/ManageServiceSlotModal";
import CalendarToolbar from "../components/CalendarToolbar";
import CalendarGrid from "../components/CalendarGrid";
import "../styles/CalendarPage.css";

type CalendarRole = "owner" | "staff" | null;

// Shared by both the owner (all-staff calendar, /service-slots) and a staff
// member (their own calendar, /my-calendar) — the backend already scopes
// every service-slot query/mutation onto a staff caller's own slots, so this
// component just needs to stop offering choices a staff member wouldn't be
// allowed anyway (staff filter, reassigning to someone else, seeing others'
// columns), and to bound the calendar to their own working hours.
export default function CalendarPage() {
    const navigate = useNavigate();
    const { token, user } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [role, setRole] = useState<CalendarRole>(null);
    const isStaff = role === "staff";
    const ownStaffId = isStaff ? String(user?.staffProfile?.staffId ?? "") : "";

    const [workingHours, setWorkingHours] = useState<WorkingHour[]>([]);
    const [staff, setStaff] = useState<Staff[]>([]);
    const [services, setServices] = useState<Service[]>([]);
    const [slots, setSlots] = useState<ServiceSlot[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [pageError, setPageError] = useState("");

    const [selectedDate, setSelectedDate] = useState<Date>(new Date());
    const [staffFilter, setStaffFilter] = useState("");
    const [serviceFilter, setServiceFilter] = useState("");
    const dateInputRef = useRef<HTMLInputElement>(null);

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

    // Manage modal — full edit (date/time/staff/packages), only available
    // while the slot has no booking yet; once booked, only reassign/delete above apply.
    const [editForm, setEditForm] = useState<ServiceSlotInput>(emptyForm(toISO(new Date())));
    const [editFormServiceId, setEditFormServiceId] = useState("");
    const [editApplyFuture, setEditApplyFuture] = useState(false);
    const [editFormError, setEditFormError] = useState("");
    const [editFieldErrors, setEditFieldErrors] = useState<Record<string, string>>({});
    const [isEditSaving, setIsEditSaving] = useState(false);

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
                const profile = profileResult.data?.userProfile;
                const business = profile?.businessProfile;
                const staffProfile = profile?.staffProfile;

                if (business) {
                    setRole("owner");
                    setWorkingHours(business.workingHours ?? []);
                    const [staffRes, svcRes] = await Promise.all([
                        getBusinessStaff(activeToken),
                        getBusinessServices(activeToken),
                    ]);
                    setStaff(staffRes.data?.displayStaff ?? []);
                    setServices(svcRes.data?.displayServices ?? []);
                } else if (staffProfile) {
                    setRole("staff");
                    // A staff member's own working hours bound the calendar's
                    // time range — no separate "business hours" fetch needed.
                    const hours = staffProfile.workingHours ?? [];
                    setWorkingHours(hours);
                    setStaff([{
                        staffId: String(staffProfile.staffId), name: user?.username ?? "Me",
                        email: user?.email ?? "", mustResetPassword: false,
                        contactNumber: user?.contactNumber ?? "", workingHours: hours,
                    }]);
                    const svcRes = await getBusinessServices(activeToken);
                    setServices(svcRes.data?.displayServices ?? []);
                } else {
                    setRole(null);
                }
            } catch {
                setPageError("Failed to load calendar.");
            } finally {
                setIsLoading(false);
            }
        };
        init();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [activeToken]);

    useEffect(() => {
        if (role) fetchSlots();
    }, [role, fetchSlots]);

    // Business-hours (or, for a staff member, their own) envelope across all
    // days → calendar time range + picker bounds.
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

    // Calendar columns: a staff member always sees just their own single
    // column (no filter, no Unassigned column). The owner sees filtered
    // staff, only the Unassigned column, or all staff + an Unassigned column
    // (owner-managed slots) when viewing everyone.
    const staffColumns: { staffId: string; name: string }[] = isStaff
        ? staff.map(s => ({ staffId: s.staffId, name: s.name }))
        : staffFilter === NONE_COL
            ? [{ staffId: NONE_COL, name: "Owner-managed" }]
            : staffFilter
                ? staff.filter(s => s.staffId === staffFilter).map(s => ({ staffId: s.staffId, name: s.name }))
                : [...staff.map(s => ({ staffId: s.staffId, name: s.name })), { staffId: NONE_COL, name: "Owner-managed" }];

    // ── Add helpers ───────────────────────────────────────────────────────────
    const openAdd = (prefill?: Partial<ServiceSlotInput>) => {
        setForm({
            ...emptyForm(isoDate), startTime: rangeStart, endTime: lines[1] ?? rangeEnd, ...prefill,
            ...(isStaff ? { staffId: ownStaffId } : {}),
        });
        setFormServiceId("");
        setFormError("");
        setFieldErrors({});
        setShowAdd(true);
    };

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

        const firstPkg = slot.serviceSlotPackages[0];
        setEditFormServiceId(firstPkg?.servicePackage.service.serviceId ?? "");
        setEditForm({
            staffId: isStaff ? ownStaffId : (slot.staff?.staffId ?? ""),
            date: slot.date,
            daysOfWeek: [],
            startTime: extractTime(slot.startTime),
            endTime: extractTime(slot.endTime),
            servicePackageIds: slot.serviceSlotPackages.map(p => p.servicePackage.servicePackageId),
        });
        setEditApplyFuture(false);
        setEditFormError("");
        setEditFieldErrors({});

        // Reassigning to someone else is an owner-only action — no need to
        // fetch candidates (the backend would reject the query for staff anyway).
        if (activeToken && !isStaff) {
            const res = await getAvailableStaffForSlot(activeToken, slot.serviceSlotId);
            setAvailableStaff(res.data?.availableStaffForSlot ?? []);
        }
    };

    const handleUpdate = async () => {
        if (!activeToken || !managing) return;
        setIsEditSaving(true);
        setEditFormError("");
        setEditFieldErrors({});
        try {
            const result = await updateServiceSlot(activeToken, managing.serviceSlotId, editForm, editApplyFuture);
            if (applyGraphQLErrors(result, {
                setFieldErrors: setEditFieldErrors, setFormError: setEditFormError, fallbackMessage: "Failed to update slot",
            })) return;
            setManaging(null);
            fetchSlots();
        } catch {
            setEditFormError("Something went wrong. Please try again.");
        } finally {
            setIsEditSaving(false);
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
            <Container className="d-flex justify-content-center align-items-center calendar-loading">
                <Spinner animation="border" variant="primary" />
            </Container>
        );
    }

    if (!role) {
        return (
            <Container className="py-5">
                <h1 className="mb-3 fs-1">Calendar</h1>
                <Alert variant="warning" className="calendar-page-alert">
                    You need to register a business profile before managing service slots.
                </Alert>
                <Button variant="primary" onClick={() => navigate("/register-business")}>Register Business Profile</Button>
            </Container>
        );
    }

    return (
        <Container fluid className="py-4 px-4">
            <CalendarToolbar
                onBack={() => navigate(isStaff ? "/profile" : "/services")}
                onAddClick={() => openAdd()}
                dateLabel={displayDate(selectedDate)}
                isoDate={isoDate}
                dateInputRef={dateInputRef}
                onPrevDate={() => setSelectedDate(addDays(selectedDate, -1))}
                onNextDate={() => setSelectedDate(addDays(selectedDate, 1))}
                onDateChange={value => setSelectedDate(new Date(`${value}T00:00:00`))}
                {...(isStaff ? {} : { staffFilter, setStaffFilter, staff })}
                serviceFilter={serviceFilter}
                setServiceFilter={setServiceFilter}
                services={services}
            />

            {pageError && <Alert variant="danger">{pageError}</Alert>}

            <CalendarGrid
                staffColumns={staffColumns}
                lines={lines}
                intervals={intervals}
                rangeEnd={rangeEnd}
                slots={slots}
                onAddSlot={openAdd}
                onManageSlot={openManage}
            />

            <AddServiceSlotModal
                show={showAdd}
                onHide={() => setShowAdd(false)}
                formError={formError}
                fieldErrors={fieldErrors}
                setFieldErrors={setFieldErrors}
                form={form}
                setForm={setForm}
                formServiceId={formServiceId}
                setFormServiceId={setFormServiceId}
                services={services}
                staff={staff}
                workingHours={workingHours}
                lines={lines}
                onSubmit={handleCreate}
                isSaving={isSaving}
                lockedStaffId={isStaff ? ownStaffId : undefined}
            />

            <ManageServiceSlotModal
                show={!!managing}
                onHide={() => setManaging(null)}
                managing={managing}
                manageError={manageError}
                services={services}
                staff={staff}
                workingHours={workingHours}
                lines={lines}
                reassignTo={reassignTo}
                setReassignTo={setReassignTo}
                availableStaff={availableStaff}
                onReassign={handleReassign}
                manageBusy={manageBusy}
                editForm={editForm}
                setEditForm={setEditForm}
                editFormServiceId={editFormServiceId}
                setEditFormServiceId={setEditFormServiceId}
                editFormError={editFormError}
                editFieldErrors={editFieldErrors}
                setEditFieldErrors={setEditFieldErrors}
                editApplyFuture={editApplyFuture}
                setEditApplyFuture={setEditApplyFuture}
                onUpdate={handleUpdate}
                isEditSaving={isEditSaving}
                deleteFuture={deleteFuture}
                setDeleteFuture={setDeleteFuture}
                onDelete={handleDelete}
                lockedStaffId={isStaff ? ownStaffId : undefined}
                allowReassign={!isStaff}
            />
        </Container>
    );
}
