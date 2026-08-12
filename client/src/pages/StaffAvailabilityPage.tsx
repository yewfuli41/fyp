import { useState, useEffect, useCallback } from "react";
import { Accordion, Alert, Badge, Button, Card, Container, Form, Modal, Spinner, Table } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { userProfile } from "../services/ProfileService";
import { getBusinessStaff, type Staff, type WorkingHour, type SlotReassignment } from "../services/StaffService";
import { staffHoursConflicts, updateStaffWorkingHours } from "../services/StaffService";
import {
    businessLeaveApplications, approveLeaveApplication, rejectLeaveApplication,
    type LeaveApplication,
} from "../services/LeaveService";
import { getAvailableStaffForSlot, extractTime, type ServiceSlot } from "../services/ServiceSlotService";
import type { BookingDetail } from "../services/BookingService";
import WorkingHoursEditor from "../components/WorkingHoursEditor";
import RescheduleBookingModal from "../modals/RescheduleBookingModal";
import { parseGraphQLErrors } from "../utils/graphqlErrors";
import { notifyPendingCountsChanged } from "../utils/pendingCounts";
import { todayISO } from "../utils/serviceSlotHelpers";
import { IconEye, IconHistory } from "../components/icons";
import CalendarIcon from "../assets/calendar.png"

// One thing that needs a replacement staff before an action (leave approval
// or a working-hours edit) can go through — normalizes the two different
// shapes that produce these (a leave's affectedBookings vs. a
// staffHoursConflicts ServiceSlot) into one.
interface Conflict {
    serviceSlotId: string;
    label: string;
}

function ReassignmentModal({
    show, title, conflicts, token, busy, error, onCancel, onConfirm,
}: {
    show: boolean;
    title: string;
    conflicts: Conflict[];
    token: string;
    busy: boolean;
    error: string;
    onCancel: () => void;
    onConfirm: (reassignments: SlotReassignment[]) => void;
}) {
    const [options, setOptions] = useState<Record<string, { staffId: string; name: string }[]>>({});
    const [picks, setPicks] = useState<Record<string, string>>({});

    useEffect(() => {
        if (!show) return;
        setPicks({});
        setOptions({});
        Promise.all(conflicts.map(async c => {
            const res = await getAvailableStaffForSlot(token, c.serviceSlotId);
            return [c.serviceSlotId, res.data?.availableStaffForSlot ?? []] as const;
        })).then(entries => setOptions(Object.fromEntries(entries)));
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [show, conflicts.map(c => c.serviceSlotId).join(",")]);

    const allPicked = conflicts.every(c => picks[c.serviceSlotId]);

    return (
        <Modal show={show} onHide={onCancel} backdrop="static">
            <Modal.Header closeButton><Modal.Title>{title}</Modal.Title></Modal.Header>
            <Modal.Body>
                <p className="text-muted small">
                    These slots already have a booking — pick a replacement staff for each before continuing.
                </p>
                {conflicts.map(c => (
                    <Form.Group key={c.serviceSlotId} className="mb-3">
                        <Form.Label>{c.label}</Form.Label>
                        <Form.Select
                            value={picks[c.serviceSlotId] ?? ""}
                            onChange={e => setPicks(prev => ({ ...prev, [c.serviceSlotId]: e.target.value }))}
                        >
                            <option value="">Select…</option>
                            <option value="__unassign__">Unassigned (owner-managed)</option>
                            {(options[c.serviceSlotId] ?? []).map(s => (
                                <option key={s.staffId} value={s.staffId}>{s.name}</option>
                            ))}
                        </Form.Select>
                    </Form.Group>
                ))}
                {error && <Alert variant="danger" className="py-2">{error}</Alert>}
            </Modal.Body>
            <Modal.Footer>
                <Button variant="outline-secondary" onClick={onCancel}>Cancel</Button>
                <Button
                    variant="primary" disabled={!allPicked || busy}
                    onClick={() => onConfirm(conflicts.map(c => ({
                        serviceSlotId: c.serviceSlotId,
                        staffId: picks[c.serviceSlotId] === "__unassign__" ? "" : picks[c.serviceSlotId],
                    })))}
                >
                    {busy ? "Saving..." : "Confirm"}
                </Button>
            </Modal.Footer>
        </Modal>
    );
}

export default function StaffAvailabilityPage() {
    const navigate = useNavigate();
    const { token } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [hasBusiness, setHasBusiness] = useState<boolean | null>(null);
    const [businessWorkingHours, setBusinessWorkingHours] = useState<WorkingHour[]>([]);
    const [staffList, setStaffList] = useState<Staff[]>([]);
    const [leaves, setLeaves] = useState<LeaveApplication[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [pageError, setPageError] = useState("");

    // Pending leave approve/reject
    const [rejectingLeave, setRejectingLeave] = useState<LeaveApplication | null>(null);
    const [rejectRemark, setRejectRemark] = useState("");
    const [rejectError, setRejectError] = useState("");
    const [rejectBusy, setRejectBusy] = useState(false);
    // Approving a leave that has affected bookings walks through
    // rescheduling every one of them first — the leave itself isn't
    // approved (approveLeaveApplication isn't even called yet) until the
    // whole queue clears. Cancelling any step abandons the approval
    // entirely: nothing was committed, the leave is still just pending.
    const [approvingLeaveWithBookings, setApprovingLeaveWithBookings] = useState<LeaveApplication | null>(null);
    const [rescheduleQueue, setRescheduleQueue] = useState<BookingDetail[]>([]);
    const [leaveBusyId, setLeaveBusyId] = useState<string | null>(null);

    // Manage staff schedule
    const [selectedStaffId, setSelectedStaffId] = useState<string>("");
    const [staffSearch, setStaffSearch] = useState("");
    const [editHours, setEditHours] = useState<WorkingHour[]>([]);
    const [hoursFieldErrors, setHoursFieldErrors] = useState<Record<string, string>>({});
    const [hoursFormError, setHoursFormError] = useState("");
    const [hoursSuccessMessage, setHoursSuccessMessage] = useState("");
    const [isSavingHours, setIsSavingHours] = useState(false);
    const [hoursConflicts, setHoursConflicts] = useState<ServiceSlot[] | null>(null);
    const [hoursConflictError, setHoursConflictError] = useState("");
    // Slots with no booking that fall outside the new hours — these are
    // unassigned outright (no replacement needed), so we just warn about them
    // in a final confirmation step after any replacement picks are made.
    const [hoursUnbookedConflicts, setHoursUnbookedConflicts] = useState<ServiceSlot[]>([]);
    const [showRemovalConfirm, setShowRemovalConfirm] = useState(false);
    const [pendingReassignments, setPendingReassignments] = useState<SlotReassignment[]>([]);

    // Approved leaves filters
    const [approvedStaffId, setApprovedStaffId] = useState("");
    const [approvedDate, setApprovedDate] = useState("");
    const [hoursConflictBusy, setHoursConflictBusy] = useState(false);

    // Approved-leaves history modal — past leaves (already over) are hidden
    // from the default list, same idea as staff's own LeaveApplicationPage.
    const [showApprovedHistory, setShowApprovedHistory] = useState(false);
    const [historyStaffId, setHistoryStaffId] = useState("");
    const [historyDate, setHistoryDate] = useState("");

    const fetchAll = useCallback(async () => {
        if (!activeToken) return;
        const [staffRes, leaveRes] = await Promise.all([
            getBusinessStaff(activeToken),
            businessLeaveApplications(activeToken),
        ]);
        if (staffRes.errors?.length) setPageError(staffRes.errors[0].message);
        else setStaffList(staffRes.data?.displayStaff ?? []);
        if (leaveRes.errors?.length) setPageError(leaveRes.errors[0].message);
        else setLeaves(leaveRes.data?.businessLeaveApplications ?? []);
    }, [activeToken]);

    useEffect(() => {
        if (!activeToken) return;
        const init = async () => {
            try {
                const profileResult = await userProfile(activeToken);
                const business = profileResult.data?.userProfile?.businessProfile;
                setHasBusiness(!!business);
                if (business) {
                    setBusinessWorkingHours(business.workingHours ?? []);
                    await fetchAll();
                }
            } catch {
                setPageError("Failed to load profile.");
            } finally {
                setIsLoading(false);
            }
        };
        init();
    }, [activeToken, fetchAll]);

    // WorkingHoursEditor expects plain "HH:MM:SS" strings (that's what
    // RegisterStaffPage, its other caller, builds locally when adding a new
    // row) — but a staff's saved hours come back from GraphQL as full RFC
    // datetimes ("0000-01-01T09:30:00Z"). Re-shape before handing them to
    // the editor, or its time dropdowns silently fail to match any option.
    const toEditableHours = (hours: WorkingHour[]): WorkingHour[] =>
        hours.map(wh => ({ day: wh.day, startTime: extractTime(wh.startTime) + ":00", endTime: extractTime(wh.endTime) + ":00" }));

    // Keep the schedule editor's selection valid, and seed it with that
    // staff's current hours whenever the selection changes.
    useEffect(() => {
        if (staffList.length === 0) return;
        const current = staffList.find(s => s.staffId === selectedStaffId) ?? staffList[0];
        setSelectedStaffId(current.staffId);
        setEditHours(toEditableHours(current.workingHours ?? []));
        setHoursFieldErrors({});
        setHoursFormError("");
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [staffList]);

    const selectStaff = (staffId: string) => {
        setSelectedStaffId(staffId);
        setEditHours(toEditableHours(staffList.find(s => s.staffId === staffId)?.workingHours ?? []));
        setHoursFieldErrors({});
        setHoursFormError("");
        setHoursSuccessMessage("");
    };

    // Cancelling the reassignment or removal-confirmation modal abandons the
    // whole edit, not just that step — the hours shown in the editor snap
    // back to what's actually saved for this staff member, discarding
    // whatever the user had changed before clicking Save.
    const resetEditHoursToSaved = () => {
        setEditHours(toEditableHours(staffList.find(s => s.staffId === selectedStaffId)?.workingHours ?? []));
        setHoursFieldErrors({});
        setHoursFormError("");
        setHoursSuccessMessage("");
    };

    const pendingLeaves = leaves.filter(l => l.status === "PENDING");

    const today = todayISO();

    // Approved leaves: filterable by staff and by a single date — a leave
    // matches if that date falls within its [startDate, endDate] range.
    // Default view only ever shows current/upcoming ones (endDate not yet
    // passed) — leaves that are already over live in the History modal instead.
    const approvedLeaves = leaves.filter(l => {
        if (l.status !== "APPROVED") return false;
        if (l.endDate < today) return false;
        if (approvedStaffId && l.staffId !== approvedStaffId) return false;
        if (approvedDate && (l.startDate > approvedDate || l.endDate < approvedDate)) return false;
        return true;
    });

    const pastApprovedLeaves = leaves.filter(l => l.status === "APPROVED" && l.endDate < today);
    const filteredApprovedHistory = pastApprovedLeaves.filter(l => {
        if (historyStaffId && l.staffId !== historyStaffId) return false;
        if (historyDate && (l.startDate > historyDate || l.endDate < historyDate)) return false;
        return true;
    });

    const filteredStaffList = staffList.filter(s => s.name.toLowerCase().includes(staffSearch.toLowerCase()));

    // ── Approve / reject ─────────────────────────────────────────────────────

    // The actual approveLeaveApplication call — never blocks on picking
    // replacements server-side (see its doc comment). Called directly for a
    // leave with no affected bookings, or once the reschedule queue below
    // has been fully cleared for one that has some.
    const approveNow = async (leave: LeaveApplication) => {
        if (!activeToken) return;
        setLeaveBusyId(leave.leaveId);
        setPageError("");
        try {
            const result = await approveLeaveApplication(activeToken, leave.leaveId);
            const parsed = parseGraphQLErrors(result, "Failed to approve leave");
            if (parsed.hasErrors) {
                setPageError(parsed.formError || Object.values(parsed.fieldErrors)[0] || "Failed to approve leave");
                return;
            }
            fetchAll();
            notifyPendingCountsChanged();
        } catch {
            setPageError("Something went wrong. Please try again.");
        } finally {
            setLeaveBusyId(null);
        }
    };

    const handleApproveClick = (leave: LeaveApplication) => {
        if (leave.affectedBookings.length === 0) {
            approveNow(leave);
            return;
        }
        // Walk through rescheduling every affected booking first — the
        // approve mutation isn't called until the queue is empty.
        setApprovingLeaveWithBookings(leave);
        setRescheduleQueue(leave.affectedBookings);
    };

    // Cancelling any step of the reschedule queue abandons the whole
    // approval — approveLeaveApplication was never called, so the leave is
    // exactly as it was: still pending.
    const cancelApprovalFlow = () => {
        setApprovingLeaveWithBookings(null);
        setRescheduleQueue([]);
    };

    const advanceApprovalQueue = () => {
        const rest = rescheduleQueue.slice(1);
        setRescheduleQueue(rest);
        if (rest.length === 0 && approvingLeaveWithBookings) {
            approveNow(approvingLeaveWithBookings);
            setApprovingLeaveWithBookings(null);
        }
        fetchAll();
    };

    const approvalQueueTotal = approvingLeaveWithBookings?.affectedBookings.length ?? 0;
    const approvalQueueNote = approvingLeaveWithBookings
        ? `Reschedule booking ${approvalQueueTotal - rescheduleQueue.length + 1} of ${approvalQueueTotal} affected by ` +
          `${approvingLeaveWithBookings.staffName}'s leave. The leave is approved once every booking is rescheduled — ` +
          `closing this without finishing leaves it pending.`
        : undefined;

    const handleReject = async () => {
        if (!activeToken || !rejectingLeave) return;
        setRejectBusy(true);
        setRejectError("");
        try {
            const result = await rejectLeaveApplication(activeToken, rejectingLeave.leaveId, rejectRemark);
            const parsed = parseGraphQLErrors(result, "Failed to reject leave");
            if (parsed.hasErrors) {
                setRejectError(parsed.formError || Object.values(parsed.fieldErrors)[0] || "Failed to reject leave");
                return;
            }
            setRejectingLeave(null);
            setRejectRemark("");
            fetchAll();
            notifyPendingCountsChanged();
        } catch {
            setRejectError("Something went wrong. Please try again.");
        } finally {
            setRejectBusy(false);
        }
    };

    // ── Manage staff schedule ────────────────────────────────────────────────

    const handleSaveHours = async () => {
        if (!activeToken || !selectedStaffId) return;
        setIsSavingHours(true);
        setHoursFormError("");
        setHoursFieldErrors({});
        setHoursSuccessMessage("");
        try {
            const conflictRes = await staffHoursConflicts(activeToken, selectedStaffId, editHours);
            if (conflictRes.errors?.length) {
                const parsed = parseGraphQLErrors(conflictRes, "Failed to check for conflicts");
                setHoursFieldErrors(parsed.fieldErrors);
                setHoursFormError(parsed.formError || Object.values(parsed.fieldErrors)[0] || "");
                return;
            }
            const conflicts = conflictRes.data?.staffHoursConflicts ?? [];
            const booked = conflicts.filter(s => s.hasBooking);
            const unbooked = conflicts.filter(s => !s.hasBooking);
            setHoursUnbookedConflicts(unbooked);
            if (booked.length > 0) {
                setHoursConflicts(booked);
                setHoursConflictError("");
                return;
            }
            if (unbooked.length > 0) {
                setPendingReassignments([]);
                setShowRemovalConfirm(true);
                return;
            }
            await saveHours([]);
        } finally {
            setIsSavingHours(false);
        }
    };

    const saveHours = async (reassignments: SlotReassignment[]) => {
        if (!activeToken || !selectedStaffId) return;
        const result = await updateStaffWorkingHours(activeToken, selectedStaffId, editHours, reassignments);
        const parsed = parseGraphQLErrors(result, "Failed to update working hours");
        if (parsed.hasErrors) {
            if (hoursConflicts || showRemovalConfirm) {
                setHoursConflictError(parsed.formError || Object.values(parsed.fieldErrors)[0] || "Failed to update working hours");
            } else {
                setHoursFieldErrors(parsed.fieldErrors);
                setHoursFormError(parsed.formError || Object.values(parsed.fieldErrors)[0] || "");
            }
            return;
        }
        setHoursConflicts(null);
        setShowRemovalConfirm(false);
        setHoursUnbookedConflicts([]);
        setPendingReassignments([]);
        setHoursSuccessMessage("Working hours saved.");
        fetchAll();
    };

    // Fires when the replacement-staff picker is confirmed. If there are also
    // unbooked slots that will simply be unassigned, don't save yet — show the
    // removal-confirmation step next instead of committing immediately.
    const confirmHoursReassignment = async (reassignments: SlotReassignment[]) => {
        if (hoursUnbookedConflicts.length > 0) {
            setPendingReassignments(reassignments);
            setHoursConflicts(null);
            setHoursConflictError("");
            setShowRemovalConfirm(true);
            return;
        }
        setHoursConflictBusy(true);
        try {
            await saveHours(reassignments);
        } finally {
            setHoursConflictBusy(false);
        }
    };

    const confirmRemoval = async () => {
        setHoursConflictBusy(true);
        try {
            await saveHours(pendingReassignments);
        } finally {
            setHoursConflictBusy(false);
        }
    };

    const cancelRemoval = () => {
        setShowRemovalConfirm(false);
        setHoursUnbookedConflicts([]);
        setPendingReassignments([]);
        setHoursConflictError("");
        resetEditHoursToSaved();
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
                <h1 className="mb-3 fs-1">Staff Availability</h1>
                <Alert variant="warning">You need to register a business profile before managing staff availability.</Alert>
                <Button variant="primary" onClick={() => navigate("/register-business")}>Register Business Profile</Button>
            </Container>
        );
    }

    const selectedStaff = staffList.find(s => s.staffId === selectedStaffId);

    return (
        <Container className="py-5" style={{ maxWidth: 900 }}>
            {pageError && <Alert variant="danger">{pageError}</Alert>}

            <div className="d-flex justify-content-between align-items-center mb-4">
    <h1 className="fs-1 mb-0">Staff Availability</h1>

    <Button
        href="#approved-leaves"
        variant="outline-primary"
        className="d-inline-flex align-items-center gap-2"
    >
        <IconEye size={16} /> View Approved Leaves
    </Button>
</div>

            <Accordion className="mb-4">
    <Accordion.Item eventKey="0">
        <Accordion.Header>
            <div className="d-flex align-items-center">
                Pending Leave Applications
                <Badge bg="danger" pill className="ms-2">
                    {pendingLeaves.length}
                </Badge>
            </div>
        </Accordion.Header>

        <Accordion.Body>

            {pendingLeaves.length === 0 ? (
                <Alert variant="info" className="mb-0">
                    No pending leave applications.
                </Alert>
            ) : (
                pendingLeaves.map((leave) => (
                    <Card
                        key={leave.leaveId}
                        className="shadow-sm border-0 mb-3"
                    >
                        <Card.Body>

                            <div className="d-flex justify-content-between align-items-start flex-wrap">

                                <div className="flex-grow-1">

                                    <h4 className="fw-bold mb-3">
                                        {leave.staffName}
                                    </h4>

                                    <div className="d-flex align-items-center mb-2">

                                        <img
                                            src={CalendarIcon}
                                            alt="calendar"
                                            width={34}
                                            className="me-3"
                                        />

                                        <div className="fw-semibold">
                                            {leave.startDate === leave.endDate
                                                ? leave.startDate
                                                : `${leave.startDate} - ${leave.endDate}`}
                                        </div>

                                    </div>

                                    {leave.justification && (
                                        <div className="text-secondary fst-italic mt-3 text-start">
                                            "{leave.justification}"
                                        </div>
                                    )}

                                </div>

                                <div className="text-end">

                                    <div className="d-flex gap-2 justify-content-end mb-3">

                                        <Button
                                            variant="success"
                                            disabled={leaveBusyId === leave.leaveId}
                                            onClick={() => handleApproveClick(leave)}
                                        >
                                            {leaveBusyId === leave.leaveId ? "Approving..." : "Approve"}
                                        </Button>

                                        <Button
                                            variant="danger"
                                            onClick={() => {
                                                setRejectingLeave(leave);
                                                setRejectRemark("");
                                                setRejectError("");
                                            }}
                                        >
                                            Reject
                                        </Button>

                                    </div>

                                    {leave.affectedBookings.length > 0 && (
                                        <Alert
                                            variant="warning"
                                            className="py-2 mb-0"
                                        >
                                            <strong>
                                                {leave.affectedBookings.length}
                                            </strong>{" "}
                                            booking(s) affected
                                        </Alert>
                                    )}

                                </div>

                            </div>

                        </Card.Body>
                    </Card>
                ))
            )}

        </Accordion.Body>
    </Accordion.Item>
</Accordion>

            <h2 className="h4 fw-bold mb-3">Manage Staff Schedule</h2>
            {staffList.length === 0 ? (
                <Alert variant="info">No staff yet.</Alert>
            ) : (
                <>
                    <div className="d-flex gap-2 mb-3" style={{ maxWidth: 500 }}>
                        <Form.Control
                            type="text"
                            placeholder="Search staff..."
                            value={staffSearch}
                            onChange={e => setStaffSearch(e.target.value)}
                        />
                        <Form.Select
                            value={selectedStaffId}
                            onChange={e => selectStaff(e.target.value)}
                            disabled={filteredStaffList.length === 0}
                        >
                            {filteredStaffList.length === 0 && <option value="">No staff match your search</option>}
                            {filteredStaffList.map(s => (
                                <option key={s.staffId} value={s.staffId}>{s.name}</option>
                            ))}
                        </Form.Select>
                    </div>

                    {selectedStaff && (
                        <Card className="mb-4">
                            <Card.Body>
                                <WorkingHoursEditor
                                    businessWorkingHours={businessWorkingHours}
                                    workingHours={editHours}
                                    setWorkingHours={update => {
                                        setHoursSuccessMessage("");
                                        setEditHours(update);
                                    }}
                                    fieldErrors={hoursFieldErrors}
                                    setFieldErrors={setHoursFieldErrors}
                                />
                                {hoursFormError && <Alert variant="danger" className="py-2">{hoursFormError}</Alert>}
                                {hoursSuccessMessage && <Alert variant="success" className="py-2">{hoursSuccessMessage}</Alert>}
                                <Button variant="primary" onClick={handleSaveHours} disabled={isSavingHours}>
                                    {isSavingHours ? "Saving..." : "Save Schedule"}
                                </Button>
                            </Card.Body>
                        </Card>
                    )}
                </>
            )}

            <div className="d-flex justify-content-between align-items-center mb-3">
                <h2 className="h4 fw-bold mb-0">Approved Leaves</h2>
                <Button
                    variant="outline-secondary" size="sm"
                    className="d-inline-flex align-items-center gap-2"
                    onClick={() => { setHistoryStaffId(approvedStaffId); setHistoryDate(""); setShowApprovedHistory(true); }}
                >
                    <IconHistory size={15} /> History
                </Button>
            </div>
            <div id="approved-leaves" className="d-flex flex-wrap gap-2 mb-3">
                <Form.Select
                    aria-label="Staff"
                    style={{ maxWidth: 220 }}
                    value={approvedStaffId}
                    onChange={e => setApprovedStaffId(e.target.value)}
                >
                    <option value="">All staff</option>
                    {staffList.map(s => (
                        <option key={s.staffId} value={s.staffId}>{s.name}</option>
                    ))}
                </Form.Select>
                <Form.Control
                    type="date"
                    aria-label="Date"
                    style={{ maxWidth: 170 }}
                    value={approvedDate}
                    onChange={e => setApprovedDate(e.target.value)}
                />
                {(approvedStaffId || approvedDate) && (
                    <Button
                        variant="outline-secondary"
                        onClick={() => { setApprovedStaffId(""); setApprovedDate(""); }}
                    >
                        Clear filters
                    </Button>
                )}
            </div>
            {approvedLeaves.length === 0 ? (
                <Alert variant="info">
                    {leaves.some(l => l.status === "APPROVED" && l.endDate >= today)
                        ? "No approved leaves match your filters."
                        : "No current or upcoming approved leaves."}
                </Alert>
            ) : (
                <div style={{ maxHeight: 420, overflowY: "auto" }}>
                <Table bordered hover responsive className="align-middle">
                    <thead>
                        <tr>
                            <th>Staff</th>
                            <th>Dates</th>
                            <th style={{ width: 120 }}>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {approvedLeaves.map(leave => (
                            <tr key={leave.leaveId}>
                                <td>{leave.staffName}</td>
                                <td>{leave.startDate === leave.endDate ? leave.startDate : `${leave.startDate} – ${leave.endDate}`}</td>
                                <td>
                                    <Button
                                        variant="outline-danger" size="sm" disabled={leaveBusyId === leave.leaveId}
                                        onClick={() => {
                                            setRejectingLeave(leave);
                                            setRejectRemark("");
                                            setRejectError("");
                                        }}
                                    >
                                        Reject
                                    </Button>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </Table>
                </div>
            )}

            {/* Reject remark */}
            <Modal show={!!rejectingLeave} onHide={() => setRejectingLeave(null)} backdrop="static">
                <Modal.Header closeButton><Modal.Title>Reject Leave Application</Modal.Title></Modal.Header>
                <Modal.Body>
                    <Form.Group>
                        <Form.Label>Reason for rejecting <span className="text-danger">*</span></Form.Label>
                        <Form.Control
                            as="textarea" rows={3}
                            value={rejectRemark}
                            onChange={e => setRejectRemark(e.target.value)}
                        />
                    </Form.Group>
                    {rejectError && <Alert variant="danger" className="py-2 mt-2">{rejectError}</Alert>}
                </Modal.Body>
                <Modal.Footer>
                    <Button variant="outline-secondary" onClick={() => setRejectingLeave(null)}>Cancel</Button>
                    <Button variant="danger" onClick={handleReject} disabled={rejectBusy || !rejectRemark.trim()}>
                        {rejectBusy ? "Rejecting..." : "Reject"}
                    </Button>
                </Modal.Footer>
            </Modal>

            {/* Hours-edit-with-conflicts picker */}
            <ReassignmentModal
                show={!!hoursConflicts}
                title="Pick replacement staff"
                conflicts={(hoursConflicts ?? []).map(s => ({
                    serviceSlotId: s.serviceSlotId,
                    label: `${s.date} ${extractTime(s.startTime)}–${extractTime(s.endTime)}`,
                }))}
                token={activeToken ?? ""}
                busy={hoursConflictBusy}
                error={hoursConflictError}
                onCancel={() => {
                    setHoursConflicts(null);
                    setHoursUnbookedConflicts([]);
                    resetEditHoursToSaved();
                }}
                onConfirm={confirmHoursReassignment}
            />

            {/* Final warning — fires after any replacement picks are made
                (or immediately if none were needed), whenever slots with no
                booking fall outside the new hours and will be unassigned. */}
            <Modal show={showRemovalConfirm} onHide={cancelRemoval} backdrop="static">
                <Modal.Header closeButton><Modal.Title>Remove {hoursUnbookedConflicts.length} slot{hoursUnbookedConflicts.length === 1 ? "" : "s"}?</Modal.Title></Modal.Header>
                <Modal.Body>
                    <p>
                        {hoursUnbookedConflicts.length} slot{hoursUnbookedConflicts.length === 1 ? "" : "s"} with no booking
                        {" "}fall{hoursUnbookedConflicts.length === 1 ? "s" : ""} outside the new hours and will be removed
                        from {selectedStaff?.name ?? "this staff member"}'s schedule:
                    </p>
                    <ul className="small text-muted">
                        {hoursUnbookedConflicts.map(s => (
                            <li key={s.serviceSlotId}>{s.date} {extractTime(s.startTime)}–{extractTime(s.endTime)}</li>
                        ))}
                    </ul>
                    <p className="mb-0">Are you sure you want to continue?</p>
                    {hoursConflictError && <Alert variant="danger" className="py-2">{hoursConflictError}</Alert>}
                </Modal.Body>
                <Modal.Footer>
                    <Button variant="outline-secondary" onClick={cancelRemoval}>Cancel</Button>
                    <Button variant="danger" onClick={confirmRemoval} disabled={hoursConflictBusy}>
                        {hoursConflictBusy ? "Saving..." : "Yes, remove and save"}
                    </Button>
                </Modal.Footer>
            </Modal>

            {/* Approve-with-affected-bookings queue — pops up right after
                clicking Approve on a leave that has some; cancelling any
                step abandons the whole approval. */}
            <RescheduleBookingModal
                show={rescheduleQueue.length > 0}
                onHide={cancelApprovalFlow}
                booking={rescheduleQueue[0] ?? null}
                token={activeToken}
                note={approvalQueueNote}
                onDone={advanceApprovalQueue}
            />

            <Modal show={showApprovedHistory} onHide={() => setShowApprovedHistory(false)} size="lg">
                <Modal.Header closeButton><Modal.Title>Approved Leaves History</Modal.Title></Modal.Header>
                <Modal.Body>
                    <div className="d-flex flex-wrap gap-2 mb-3">
                        <Form.Select
                            aria-label="Staff"
                            style={{ maxWidth: 220 }}
                            value={historyStaffId}
                            onChange={e => setHistoryStaffId(e.target.value)}
                        >
                            <option value="">All staff</option>
                            {staffList.map(s => (
                                <option key={s.staffId} value={s.staffId}>{s.name}</option>
                            ))}
                        </Form.Select>
                        <Form.Control
                            type="date"
                            aria-label="Date"
                            style={{ maxWidth: 170 }}
                            value={historyDate}
                            onChange={e => setHistoryDate(e.target.value)}
                        />
                        {(historyStaffId || historyDate) && (
                            <Button
                                variant="outline-secondary"
                                onClick={() => { setHistoryStaffId(""); setHistoryDate(""); }}
                            >
                                Clear filters
                            </Button>
                        )}
                    </div>

                    {filteredApprovedHistory.length === 0 ? (
                        <Alert variant="info" className="mb-0">
                            {pastApprovedLeaves.length === 0 ? "No past approved leaves." : "No past approved leaves match your filters."}
                        </Alert>
                    ) : (
                        <div style={{ maxHeight: 420, overflowY: "auto" }}>
                            <Table bordered hover responsive className="align-middle mb-0">
                                <thead>
                                    <tr>
                                        <th>Staff</th>
                                        <th>Dates</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {filteredApprovedHistory.map(leave => (
                                        <tr key={leave.leaveId}>
                                            <td>{leave.staffName}</td>
                                            <td>{leave.startDate === leave.endDate ? leave.startDate : `${leave.startDate} – ${leave.endDate}`}</td>
                                        </tr>
                                    ))}
                                </tbody>
                            </Table>
                        </div>
                    )}
                </Modal.Body>
            </Modal>
        </Container>
    );
}
