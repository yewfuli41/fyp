import { useState, useEffect, useCallback } from "react";
import { Alert, Badge, Button, Card, Container, Form, Modal, Spinner } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { userProfile } from "../services/ProfileService";
import { getBusinessStaff, type Staff } from "../services/StaffService";
import {
    businessLeaveApplications, approveLeaveApplication, rejectLeaveApplication,
    type LeaveApplication, type LeaveReschedule,
} from "../services/LeaveService";
import type { BookingDetail } from "../services/BookingService";
import RescheduleBookingModal from "../modals/RescheduleBookingModal";
import LeaveFilterBar, { leaveMatchesFilters } from "../components/LeaveFilterBar";
import LeaveTable from "../components/LeaveTable";
import LeaveHistoryModal from "../components/LeaveHistoryModal";
import { parseGraphQLErrors } from "../utils/graphqlErrors";
import { notifyPendingCountsChanged } from "../utils/pendingCounts";
import { todayISO } from "../utils/serviceSlotHelpers";
import { IconEye, IconHistory, IconDownload } from "../components/icons";
import CalendarIcon from "../assets/calendar.png";

export default function StaffLeavePage() {
    const navigate = useNavigate();
    const { token } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [hasBusiness, setHasBusiness] = useState<boolean | null>(null);
    const [staffList, setStaffList] = useState<Staff[]>([]);
    const [leaves, setLeaves] = useState<LeaveApplication[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [pageError, setPageError] = useState("");

    // Reject (also the way an already-approved leave is reversed).
    const [rejectingLeave, setRejectingLeave] = useState<LeaveApplication | null>(null);
    const [rejectRemark, setRejectRemark] = useState("");
    const [rejectError, setRejectError] = useState("");
    const [rejectBusy, setRejectBusy] = useState(false);
    const [leaveBusyId, setLeaveBusyId] = useState<string | null>(null);

    // Approving a leave that has affected bookings walks the owner through
    // picking a new slot for every one of them first. Each pick is only
    // staged, never written — the whole batch is submitted with the approval
    // once the queue clears, and the server applies it in one transaction.
    // Cancelling any step abandons the approval entirely: nothing was ever
    // committed, so the leave is still pending and no customer was moved.
    const [approvingLeaveWithBookings, setApprovingLeaveWithBookings] = useState<LeaveApplication | null>(null);
    const [rescheduleQueue, setRescheduleQueue] = useState<BookingDetail[]>([]);
    const [stagedReschedules, setStagedReschedules] = useState<LeaveReschedule[]>([]);

    const [approvedStaffId, setApprovedStaffId] = useState("");
    const [approvedDate, setApprovedDate] = useState("");
    const [showApprovedHistory, setShowApprovedHistory] = useState(false);

    const [rejectedStaffId, setRejectedStaffId] = useState("");
    const [rejectedDate, setRejectedDate] = useState("");
    const [showRejectedHistory, setShowRejectedHistory] = useState(false);

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
                if (business) await fetchAll();
            } catch {
                setPageError("Failed to load profile.");
            } finally {
                setIsLoading(false);
            }
        };
        init();
    }, [activeToken, fetchAll]);

    const today = todayISO();
    const pendingLeaves = leaves.filter(l => l.status === "PENDING");

    // Each decided status splits the same way: still relevant (the leave
    // hasn't finished yet) in the list, already over in its History modal.
    const byStatus = (status: string, past: boolean) =>
        leaves.filter(l => l.status === status && (past ? l.endDate < today : l.endDate >= today));

    const currentApproved = byStatus("APPROVED", false);
    const pastApproved = byStatus("APPROVED", true);
    const currentRejected = byStatus("REJECTED", false);
    const pastRejected = byStatus("REJECTED", true);

    const filteredApproved = currentApproved.filter(l => leaveMatchesFilters(l, approvedStaffId, approvedDate));
    const filteredRejected = currentRejected.filter(l => leaveMatchesFilters(l, rejectedStaffId, rejectedDate));

    // ── approve / reject ─────────────────────────────────────────────────────

    const approveNow = async (leave: LeaveApplication, reschedules: LeaveReschedule[] = []) => {
        if (!activeToken) return;
        setLeaveBusyId(leave.leaveId);
        setPageError("");
        try {
            const result = await approveLeaveApplication(activeToken, leave.leaveId, reschedules);
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
        setApprovingLeaveWithBookings(leave);
        setRescheduleQueue(leave.affectedBookings);
        setStagedReschedules([]);
    };

    // Cancelling any step abandons the whole approval — no reschedule was
    // ever written and approveLeaveApplication was never called, so the leave
    // and every one of its bookings are exactly as they were.
    const cancelApprovalFlow = () => {
        setApprovingLeaveWithBookings(null);
        setRescheduleQueue([]);
        setStagedReschedules([]);
    };

    const stageReschedule = (newSlotOptionId: string) => {
        const current = rescheduleQueue[0];
        if (!current) return;
        // Built locally rather than read back from state — the last pick has
        // to be in the batch we submit on this very tick.
        const staged = [...stagedReschedules, { bookingId: current.bookingId, newSlotOptionId }];
        const rest = rescheduleQueue.slice(1);
        setStagedReschedules(staged);
        setRescheduleQueue(rest);
        if (rest.length === 0 && approvingLeaveWithBookings) {
            approveNow(approvingLeaveWithBookings, staged);
            setApprovingLeaveWithBookings(null);
            setStagedReschedules([]);
        }
    };

    const approvalQueueNote = approvingLeaveWithBookings
        ? `This booking is affected by ${approvingLeaveWithBookings.staffName}'s leave. ` +
          `The leave is approved once every booking is rescheduled — ` +
          `closing this without finishing leaves it pending.`
        : undefined;

    const openRejectModal = (leave: LeaveApplication) => {
        setRejectingLeave(leave);
        setRejectRemark("");
        setRejectError("");
    };

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

    // ── render ───────────────────────────────────────────────────────────────

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
                <h1 className="mb-3 fs-1">Staff Leave</h1>
                <Alert variant="warning">You need to register a business profile before managing staff leave.</Alert>
                <Button variant="primary" onClick={() => navigate("/register-business")}>Register Business Profile</Button>
            </Container>
        );
    }

    return (
        <Container className="py-5" style={{ maxWidth: 900 }}>
            {pageError && <Alert variant="danger">{pageError}</Alert>}

            <h1 className="fs-1 mb-4 text-start">Staff Leave</h1>

            {/* ── Pending ──────────────────────────────────────────────── */}
            <div className="d-flex justify-content-between align-items-center flex-wrap gap-2 mb-3">
                <div className="d-flex align-items-center">
                    <h2 className="h4 fw-bold mb-0">Pending Applications</h2>
                    <Badge bg={pendingLeaves.length > 0 ? "danger" : "secondary"} pill className="ms-2">
                        {pendingLeaves.length}
                    </Badge>
                </div>
                {/* Both decided lists sit further down the page — jump to them
                    rather than making the owner scroll past the pending cards. */}
                <div className="d-flex gap-2">
                    <Button
                        href="#approved-leaves"
                        variant="outline-primary" size="sm"
                        className="d-inline-flex align-items-center gap-2"
                    >
                        <IconEye size={15} /> Approved Leaves
                    </Button>
                    <Button
                        href="#rejected-leaves"
                        variant="outline-danger" size="sm"
                        className="d-inline-flex align-items-center gap-2"
                    >
                        <IconEye size={15} /> Rejected Leaves
                    </Button>
                </div>
            </div>

            {pendingLeaves.length === 0 ? (
                <Alert variant="info">No pending leave applications.</Alert>
            ) : (
                pendingLeaves.map(leave => (
                    <Card key={leave.leaveId} className="shadow-sm border-0 mb-3">
                        <Card.Body>
                            <div className="d-flex justify-content-between align-items-start flex-wrap gap-3">
                                <div className="flex-grow-1">
                                    <h4 className="fw-bold mb-3 text-start">{leave.staffName}</h4>
                                    <div className="d-flex align-items-center mb-2">
                                        <img src={CalendarIcon} alt="" width={34} className="me-3" />
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
                                    <div className="text-start">
                                        {leave.fileUrl && (
                                            <a
                                                href={leave.fileUrl}
                                                download={`leave-${leave.leaveId}-attachment`}
                                                className="d-inline-flex align-items-center gap-1 mt-2 small"
                                            >
                                                <IconDownload size={14} /> Download attachment
                                            </a>
                                        )}
                                    </div>
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
                                        <Button variant="danger" onClick={() => openRejectModal(leave)}>
                                            Reject
                                        </Button>
                                    </div>
                                    {leave.affectedBookings.length > 0 && (
                                        <Alert variant="warning" className="py-2 mb-0">
                                            <strong>{leave.affectedBookings.length}</strong> booking(s) affected
                                        </Alert>
                                    )}
                                </div>
                            </div>
                        </Card.Body>
                    </Card>
                ))
            )}

            {/* ── Approved ─────────────────────────────────────────────── */}
            <div id="approved-leaves" className="d-flex justify-content-between align-items-center mb-3 mt-5">
                <h2 className="h4 fw-bold mb-0">Approved Leaves</h2>
                <Button
                    variant="outline-secondary" size="sm"
                    className="d-inline-flex align-items-center gap-2"
                    onClick={() => setShowApprovedHistory(true)}
                >
                    <IconHistory size={15} /> History
                </Button>
            </div>
            <LeaveFilterBar
                staffList={staffList}
                staffId={approvedStaffId}
                onStaffChange={setApprovedStaffId}
                date={approvedDate}
                onDateChange={setApprovedDate}
            />
            <LeaveTable
                leaves={filteredApproved}
                onReject={openRejectModal}
                busyLeaveId={leaveBusyId}
                emptyMessage={currentApproved.length === 0
                    ? "No current or upcoming approved leaves."
                    : "No approved leaves match your filters."}
            />

            {/* ── Rejected ─────────────────────────────────────────────── */}
            <div id="rejected-leaves" className="d-flex justify-content-between align-items-center mb-3 mt-5">
                <h2 className="h4 fw-bold mb-0">Rejected Leaves</h2>
                <Button
                    variant="outline-secondary" size="sm"
                    className="d-inline-flex align-items-center gap-2"
                    onClick={() => setShowRejectedHistory(true)}
                >
                    <IconHistory size={15} /> History
                </Button>
            </div>
            <LeaveFilterBar
                staffList={staffList}
                staffId={rejectedStaffId}
                onStaffChange={setRejectedStaffId}
                date={rejectedDate}
                onDateChange={setRejectedDate}
            />
            <LeaveTable
                leaves={filteredRejected}
                showReason
                emptyMessage={currentRejected.length === 0
                    ? "No current or upcoming rejected leaves."
                    : "No rejected leaves match your filters."}
            />

            {/* Reject remark — also how an approved leave is reversed. */}
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

            {/* Approve-with-affected-bookings queue — pops up right after
                clicking Approve on a leave that has some; cancelling any
                step abandons the whole approval. */}
            <RescheduleBookingModal
                show={rescheduleQueue.length > 0}
                onHide={cancelApprovalFlow}
                booking={rescheduleQueue[0] ?? null}
                token={activeToken}
                note={approvalQueueNote}
                onStage={stageReschedule}
                // The staff is off for this whole range, so none of their own
                // slots in it may be offered as a destination.
                unavailable={approvingLeaveWithBookings ? {
                    staffId: approvingLeaveWithBookings.staffId,
                    from: approvingLeaveWithBookings.startDate,
                    until: approvingLeaveWithBookings.endDate,
                } : undefined}
            />

            <LeaveHistoryModal
                show={showApprovedHistory}
                onHide={() => setShowApprovedHistory(false)}
                title="Approved Leaves History"
                leaves={pastApproved}
                staffList={staffList}
                initialStaffId={approvedStaffId}
                emptyMessage="No past approved leaves."
                noMatchMessage="No past approved leaves match your filters."
            />

            <LeaveHistoryModal
                show={showRejectedHistory}
                onHide={() => setShowRejectedHistory(false)}
                title="Rejected Leaves History"
                leaves={pastRejected}
                staffList={staffList}
                showReason
                initialStaffId={rejectedStaffId}
                emptyMessage="No past rejected leaves."
                noMatchMessage="No past rejected leaves match your filters."
            />
        </Container>
    );
}
