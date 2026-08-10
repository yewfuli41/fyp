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
    DEFAULT_START, DEFAULT_END, NONE_COL, addDays, displayDate, emptyForm, openIntervals, timeLines, toISO,
    weekdayName, type WorkingHour,
} from "../utils/serviceSlotHelpers";
import AddServiceSlotModal, { type TimeRange } from "../modals/AddServiceSlotModal";
import ManageServiceSlotModal from "../modals/ManageServiceSlotModal";
import RescheduleBookingModal from "../modals/RescheduleBookingModal";
import ConfirmDeleteModal from "../modals/ConfirmDeleteModal";
import ConfirmActionModal from "../modals/ConfirmActionModal";
import RecordWalkInModal, { type WalkInSubmission } from "../modals/RecordWalkInModal";
import CalendarToolbar from "../components/CalendarToolbar";
import CalendarGrid from "../components/CalendarGrid";
import CalendarLegend from "../components/CalendarLegend";
import BookingRequestsPanel from "../components/BookingRequestsPanel";
import {
    getBusinessBookings, acceptBooking, rejectBooking, cancelBooking, recordWalkIn, type BookingDetail,
} from "../services/BookingService";
import { businessLeaveApplications, myLeaveApplications, type LeaveApplication } from "../services/LeaveService";
import { notifyPendingCountsChanged } from "../utils/pendingCounts";
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
    const [bookings, setBookings] = useState<BookingDetail[]>([]);
    const [leaves, setLeaves] = useState<LeaveApplication[]>([]);
    const [bookingBusyId, setBookingBusyId] = useState<string | null>(null);
    const [reschedulingBooking, setReschedulingBooking] = useState<BookingDetail | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [pageError, setPageError] = useState("");

    const [selectedDate, setSelectedDate] = useState<Date>(new Date());
    // Multi-select checkbox filters (empty = show all). Staff uses NONE_COL for
    // owner-managed (unassigned) slots.
    const [staffFilters, setStaffFilters] = useState<string[]>([]);
    const [serviceFilters, setServiceFilters] = useState<string[]>([]);
    const dateInputRef = useRef<HTMLInputElement>(null);

    // Add modal
    const [showAdd, setShowAdd] = useState(false);
    const [form, setForm] = useState<ServiceSlotInput>(emptyForm(toISO(new Date())));
    // Specific dates to create slots for — see AddServiceSlotModal.
    const [addDates, setAddDates] = useState<string[]>([]);
    // Every time range to create a slot for — see AddServiceSlotModal.
    const [addRanges, setAddRanges] = useState<TimeRange[]>([]);
    const [formServiceId, setFormServiceId] = useState("");
    const [formError, setFormError] = useState("");
    const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
    const [isSaving, setIsSaving] = useState(false);

    // Record Walk-In modal
    const [showWalkIn, setShowWalkIn] = useState(false);
    const [walkInFormError, setWalkInFormError] = useState("");
    const [isSavingWalkIn, setIsSavingWalkIn] = useState(false);

    // Manage modal
    const [managing, setManaging] = useState<ServiceSlot | null>(null);
    const [availableStaff, setAvailableStaff] = useState<{ staffId: string; name: string }[]>([]);
    const [reassignTo, setReassignTo] = useState("");
    const [deleteFuture, setDeleteFuture] = useState(false);
    const [manageError, setManageError] = useState("");
    const [manageBusy, setManageBusy] = useState(false);
    const [confirmingDeleteSlot, setConfirmingDeleteSlot] = useState(false);
    const [deleteSlotError, setDeleteSlotError] = useState("");
    const [editForm, setEditForm] = useState<ServiceSlotInput>(emptyForm(toISO(new Date())));
    const [editFormServiceId, setEditFormServiceId] = useState("");
    const [editFormError, setEditFormError] = useState("");
    const [editFieldErrors, setEditFieldErrors] = useState<Record<string, string>>({});
    const [isEditSaving, setIsEditSaving] = useState(false);

    const isoDate = toISO(selectedDate);

    // Fetch every slot for the day; staff/service filtering is done client-side
    // so the checkbox filters can be multi-select.
    const fetchSlots = useCallback(async () => {
        if (!activeToken) return;
        const result = await getServiceSlots(activeToken, toISO(selectedDate));
        if (result.errors?.length) setPageError(result.errors[0].message);
        else setSlots(result.data?.displayServiceSlots ?? []);
    }, [activeToken, selectedDate]);

    const fetchBookings = useCallback(async () => {
        if (!activeToken) return;
        const result = await getBusinessBookings(activeToken);
        if (!result.errors?.length) setBookings(result.data?.businessBookings ?? []);
    }, [activeToken]);

    // businessLeaveApplications is owner-only (it resolves the caller's own
    // business profile) — a staff caller instead fetches just their own
    // applications, which is all their single column needs anyway.
    const fetchLeaves = useCallback(async () => {
        if (!activeToken || !role) return;
        if (isStaff) {
            const result = await myLeaveApplications(activeToken);
            if (!result.errors?.length) setLeaves(result.data?.myLeaveApplications ?? []);
        } else {
            const result = await businessLeaveApplications(activeToken);
            if (!result.errors?.length) setLeaves(result.data?.businessLeaveApplications ?? []);
        }
    }, [activeToken, role, isStaff]);

    // Requests panel: PENDING (awaiting the business's own decision) plus
    // RESCHEDULED (the business's own reschedule proposal, awaiting the
    // customer) — a rescheduled booking still needs to stay visible here so it
    // doesn't vanish just because it's no longer the business's turn to Accept
    // (BookingRequestsPanel hides Accept for it, but Reject/Reschedule remain).
    const pendingBookings = bookings.filter(b => b.status === "PENDING" || b.status === "RESCHEDULED");
    const bookingBySlot = useMemo(() => {
        const map: Record<string, BookingDetail> = {};
        for (const b of bookings) {
            if (b.status === "PENDING" || b.status === "ACCEPTED" || b.status === "RESCHEDULED") map[b.serviceSlotId] = b;
        }
        return map;
    }, [bookings]);

    // onError, when given, receives the failure message instead of it going
    // to the page-level alert — used by the confirm-dialog flow below so the
    // error shows inline in the dialog instead of behind it.
    const runBookingAction = async (
        b: BookingDetail,
        fn: (token: string, id: string) => Promise<{ errors?: { message: string }[] }>,
        onError?: (message: string) => void,
    ): Promise<boolean> => {
        if (!activeToken) return false;
        setBookingBusyId(b.bookingId);
        setPageError("");
        try {
            const result = await fn(activeToken, b.bookingId);
            if (result.errors?.length) {
                const message = result.errors[0].message;
                if (onError) onError(message); else setPageError(message);
                return false;
            }
            setManaging(null);
            await Promise.all([fetchBookings(), fetchSlots()]);
            notifyPendingCountsChanged();
            return true;
        } catch {
            const message = "Something went wrong. Please try again.";
            if (onError) onError(message); else setPageError(message);
            return false;
        } finally {
            setBookingBusyId(null);
        }
    };

    // Reject/cancel confirmation — gates the three destructive booking
    // actions (reject a request, reject a reschedule proposal, cancel a
    // booking) behind a "are you sure?" step instead of firing immediately.
    const [pendingBookingAction, setPendingBookingAction] = useState<{
        booking: BookingDetail;
        fn: (token: string, id: string) => Promise<{ errors?: { message: string }[] }>;
        title: string;
        body: string;
        confirmLabel: string;
    } | null>(null);
    const [pendingBookingActionError, setPendingBookingActionError] = useState("");

    const confirmBookingAction = (
        booking: BookingDetail,
        fn: (token: string, id: string) => Promise<{ errors?: { message: string }[] }>,
        title: string,
        body: string,
        confirmLabel: string,
    ) => {
        setPendingBookingActionError("");
        setPendingBookingAction({ booking, fn, title, body, confirmLabel });
    };

    const handleConfirmedBookingAction = async () => {
        if (!pendingBookingAction) return;
        const ok = await runBookingAction(pendingBookingAction.booking, pendingBookingAction.fn, setPendingBookingActionError);
        if (ok) setPendingBookingAction(null);
    };

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

    useEffect(() => {
        if (role) fetchBookings();
    }, [role, fetchBookings]);

    useEffect(() => {
        if (role) fetchLeaves();
    }, [role, fetchLeaves]);

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

    // Every possible column for the owner (all staff + Owner-managed); a staff
    // member always sees just their own single column.
    const allColumns: { staffId: string; name: string }[] = isStaff
        ? staff.map(s => ({ staffId: s.staffId, name: s.name }))
        : [...staff.map(s => ({ staffId: s.staffId, name: s.name })), { staffId: NONE_COL, name: "Owner-managed" }];

    // Checked staff filters narrow which columns show (none checked = all).
    const staffColumns = (!isStaff && staffFilters.length > 0)
        ? allColumns.filter(c => staffFilters.includes(c.staffId))
        : allColumns;

    // Client-side service filter: keep slots offering at least one selected service.
    const filteredSlots = serviceFilters.length === 0
        ? slots
        : slots.filter(slot => slot.serviceSlotOptions.some(o => serviceFilters.includes(o.serviceOption.service.serviceId)));

    // Per-column open intervals on the selected weekday — owner-managed slots
    // follow business hours, everyone else follows their own working hours.
    const openHoursByColumn = useMemo(() => {
        const weekday = weekdayName(isoDate);
        const map: Record<string, { start: string; end: string }[]> = {};
        for (const col of staffColumns) {
            const hours = col.staffId === NONE_COL
                ? workingHours
                : (staff.find(s => s.staffId === col.staffId)?.workingHours ?? []);
            map[col.staffId] = openIntervals(hours, weekday);
        }
        return map;
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [staffColumns, staff, workingHours, isoDate]);

    // Which columns have the staff on approved leave for the selected day —
    // Owner-managed never does (no staff to be on leave). Reassigning a
    // booked slot away from a staff going on leave only ever changes who's
    // assigned (see resolveSlotConflicts server-side), never which service
    // option it's for, so this is purely about showing the day as blocked.
    const onLeaveByColumn = useMemo(() => {
        const map: Record<string, boolean> = {};
        for (const col of staffColumns) {
            if (col.staffId === NONE_COL) continue;
            map[col.staffId] = leaves.some(l =>
                l.staffId === col.staffId && l.status === "APPROVED"
                && l.startDate <= isoDate && l.endDate >= isoDate
            );
        }
        return map;
    }, [staffColumns, leaves, isoDate]);

    // ── Add helpers ───────────────────────────────────────────────────────────
    const openAdd = (prefill?: Partial<ServiceSlotInput>) => {
        const startTime = rangeStart, endTime = lines[1] ?? rangeEnd;
        setForm({ ...emptyForm(isoDate), startTime, endTime, ...prefill, ...(isStaff ? { staffId: ownStaffId } : {}) });
        setAddDates(prefill?.daysOfWeek?.length ? [""] : [isoDate]);
        setAddRanges([{ startTime, endTime }]);
        setFormServiceId("");
        setFormError("");
        setFieldErrors({});
        setShowAdd(true);
    };

    const handleCreate = async () => {
        if (!activeToken) return;
        const datesToCreate = addDates.filter(Boolean);
        if (form.daysOfWeek.length === 0 && datesToCreate.length === 0) {
            setFieldErrors({ schedule: "Select at least one date." });
            return;
        }
        setIsSaving(true);
        setFormError("");
        setFieldErrors({});
        try {
            // Weekday-recurring: one mutation per time range (the backend
            // expands each into its own set of future occurrences). Specific
            // dates: one slot per date × time range, sequentially, so a
            // failure partway through still leaves what came before it created.
            const inputs: ServiceSlotInput[] = form.daysOfWeek.length > 0
                ? addRanges.map(range => ({ ...form, ...range }))
                : datesToCreate.flatMap(date => addRanges.map(range => ({ ...form, date, daysOfWeek: [], ...range })));
            for (const input of inputs) {
                const result = await createServiceSlot(activeToken, input);
                if (applyGraphQLErrors(result, { setFieldErrors, setFormError, fallbackMessage: "Failed to create slot" })) return;
            }
            setShowAdd(false);
            fetchSlots();
        } catch {
            setFormError("Something went wrong. Please try again.");
        } finally {
            setIsSaving(false);
        }
    };

    const handleRecordWalkIn = async (input: WalkInSubmission) => {
        if (!activeToken) return;
        setIsSavingWalkIn(true);
        setWalkInFormError("");
        try {
            const result = await recordWalkIn(activeToken, input);
            if (applyGraphQLErrors(result, { setFormError: setWalkInFormError, fallbackMessage: "Failed to record walk-in" })) return;
            setShowWalkIn(false);
            await Promise.all([fetchSlots(), fetchBookings()]);
        } catch {
            setWalkInFormError("Something went wrong. Please try again.");
        } finally {
            setIsSavingWalkIn(false);
        }
    };

    // ── Manage helpers ──────────────────────────────────────────────────────────
    const openManage = async (slot: ServiceSlot) => {
        setManaging(slot);
        setReassignTo("");
        setDeleteFuture(false);
        setManageError("");
        setConfirmingDeleteSlot(false);
        setDeleteSlotError("");

        const firstPkg = slot.serviceSlotOptions[0];
        setEditFormServiceId(firstPkg?.serviceOption.service.serviceId ?? "");
        setEditForm({
            staffId: isStaff ? ownStaffId : (slot.staff?.staffId ?? ""),
            date: slot.date,
            daysOfWeek: [],
            startTime: extractTime(slot.startTime),
            endTime: extractTime(slot.endTime),
            serviceOptionIds: slot.serviceSlotOptions.map(p => p.serviceOption.serviceOptionId),
        });
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
            const result = await updateServiceSlot(activeToken, managing.serviceSlotId, editForm);
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

    const openDeleteSlotConfirm = () => {
        setDeleteSlotError("");
        setConfirmingDeleteSlot(true);
    };

    const handleDelete = async () => {
        if (!activeToken || !managing) return;
        setManageBusy(true);
        setDeleteSlotError("");
        try {
            const result = await deleteServiceSlot(activeToken, managing.serviceSlotId, deleteFuture);
            if (result.errors?.length) {
                const parsed = parseGraphQLErrors(result, "Failed to delete slot");
                const partialMessage = parsed.fieldErrors.deleteFutureRecurring;
                if (partialMessage) {
                    setConfirmingDeleteSlot(false);
                    setManaging(null);
                    setPageError(partialMessage);
                    fetchSlots();
                    fetchBookings();
                    return;
                }
                setDeleteSlotError(parsed.formError || Object.values(parsed.fieldErrors)[0] || result.errors[0].message);
                return;
            }
            setConfirmingDeleteSlot(false);
            setManaging(null);
            fetchSlots();
        } catch {
            setDeleteSlotError("Something went wrong. Please try again.");
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
        <Container fluid className="py-5 px-4">
           
            <BookingRequestsPanel
                pending={pendingBookings}
                busyId={bookingBusyId}
                onAccept={b => runBookingAction(b, acceptBooking)}
                onReject={b => confirmBookingAction(
                    b, rejectBooking, "Reject booking request",
                    `Reject the booking request from ${b.customerName}? This can't be undone.`,
                    "Reject",
                )}
                onReschedule={b => setReschedulingBooking(b)}
            />

            <CalendarToolbar
                onAddClick={() => openAdd()}
                onWalkInClick={() => { setWalkInFormError(""); setShowWalkIn(true); }}
                dateLabel={displayDate(selectedDate)}
                isoDate={isoDate}
                dateInputRef={dateInputRef}
                onPrevDate={() => setSelectedDate(addDays(selectedDate, -1))}
                onNextDate={() => setSelectedDate(addDays(selectedDate, 1))}
                onDateChange={value => setSelectedDate(new Date(`${value}T00:00:00`))}
                {...(isStaff ? {} : { staffFilters, setStaffFilters, staff })}
                serviceFilters={serviceFilters}
                setServiceFilters={setServiceFilters}
                services={services}
            />

            {pageError && <Alert variant="danger">{pageError}</Alert>}

            <CalendarLegend />

            <CalendarGrid
                staffColumns={staffColumns}
                lines={lines}
                intervals={intervals}
                rangeEnd={rangeEnd}
                slots={filteredSlots}
                openHoursByColumn={openHoursByColumn}
                onLeaveByColumn={onLeaveByColumn}
                onAddSlot={openAdd}
                onManageSlot={openManage}
                bookingBySlot={bookingBySlot}
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
                dates={addDates}
                setDates={setAddDates}
                ranges={addRanges}
                setRanges={setAddRanges}
                services={services}
                staff={staff}
                workingHours={workingHours}
                lines={lines}
                onSubmit={handleCreate}
                isSaving={isSaving}
                lockedStaffId={isStaff ? ownStaffId : undefined}
            />

            <RecordWalkInModal
                show={showWalkIn}
                onHide={() => setShowWalkIn(false)}
                services={services}
                staff={staff}
                workingHours={workingHours}
                lines={lines}
                leaves={leaves}
                initialDate={isoDate}
                onSubmit={handleRecordWalkIn}
                isSaving={isSavingWalkIn}
                formError={walkInFormError}
                lockedStaffId={isStaff ? ownStaffId : undefined}
            />

            <ManageServiceSlotModal
                show={!!managing && !confirmingDeleteSlot && !pendingBookingAction}
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
                onUpdate={handleUpdate}
                isEditSaving={isEditSaving}
                lockedStaffId={isStaff ? ownStaffId : undefined}
                deleteFuture={deleteFuture}
                setDeleteFuture={setDeleteFuture}
                onDelete={openDeleteSlotConfirm}
                allowReassign={!isStaff}
                booking={managing ? bookingBySlot[managing.serviceSlotId] : null}
                onCancelBooking={() => {
                    const b = managing && bookingBySlot[managing.serviceSlotId];
                    if (b) confirmBookingAction(
                        b, cancelBooking, "Cancel booking",
                        `Cancel ${b.customerName}'s booking? This can't be undone.`,
                        "Yes, cancel",
                    );
                }}
                onRescheduleBooking={() => {
                    const b = managing && bookingBySlot[managing.serviceSlotId];
                    if (b) { setManaging(null); setReschedulingBooking(b); }
                }}
                onRejectBooking={() => {
                    const b = managing && bookingBySlot[managing.serviceSlotId];
                    if (b) confirmBookingAction(
                        b, rejectBooking, "Reject reschedule",
                        "Reject this reschedule proposal? This can't be undone.",
                        "Reject",
                    );
                }}
            />

            <ConfirmDeleteModal
                show={confirmingDeleteSlot}
                title="Delete service slot"
                itemName={deleteFuture ? "this slot and every later occurrence of this series, from this date onward" : "this service slot"}
                warningNote={deleteFuture ? "Slots that already have a booking are kept, not deleted. Earlier occurrences before this date are also kept." : undefined}
                error={deleteSlotError}
                isDeleting={manageBusy}
                onCancel={() => {
                    setConfirmingDeleteSlot(false);
                    setManaging(null);
                }}
                onConfirm={handleDelete}
            />

            <ConfirmActionModal
                show={!!pendingBookingAction}
                title={pendingBookingAction?.title ?? ""}
                body={pendingBookingAction?.body ?? ""}
                confirmLabel={pendingBookingAction?.confirmLabel ?? "Confirm"}
                confirmingLabel="Working..."
                error={pendingBookingActionError}
                isBusy={!!pendingBookingAction && bookingBusyId === pendingBookingAction.booking.bookingId}
                onCancel={() => setPendingBookingAction(null)}
                onConfirm={handleConfirmedBookingAction}
            />

            <RescheduleBookingModal
                show={!!reschedulingBooking}
                onHide={() => setReschedulingBooking(null)}
                booking={reschedulingBooking}
                token={activeToken}
                onDone={() => {
                    setReschedulingBooking(null);
                    fetchBookings();
                    fetchSlots();
                    notifyPendingCountsChanged();
                }}
                lockedStaffId={isStaff ? ownStaffId : undefined}
            />
        </Container>
    );
}
