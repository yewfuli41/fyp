import { useState, useEffect, useCallback } from "react";
import { Alert, Button, Card, Container, Form, Modal, Spinner } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { userProfile } from "../services/ProfileService";
import {
    getBusinessStaff, staffHoursConflicts, updateStaffWorkingHours,
    type Staff, type WorkingHour, type SlotReassignment,
} from "../services/StaffService";
import { getAvailableStaffForSlot, extractTime, type ServiceSlot } from "../services/ServiceSlotService";
import WorkingHoursEditor from "../components/WorkingHoursEditor";
import { parseGraphQLErrors } from "../utils/graphqlErrors";
import { IconChevronLeft } from "../components/icons";

// One booked slot that needs a replacement staff before a working-hours edit
// can go through.
interface Conflict {
    serviceSlotId: string;
    label: string;
}

function ReassignmentModal({
    show, title, conflicts, token, busy, error, excludeStaffId, onCancel, onConfirm,
}: {
    show: boolean;
    title: string;
    conflicts: Conflict[];
    token: string;
    busy: boolean;
    error: string;
    // The staff member whose hours are being reduced, when that is what
    // triggered this dialog. They are never a valid pick for their own
    // conflicting slot — the whole point is that they stop working then — and
    // availableStaffForSlot still lists them because it reads their saved
    // hours, which the edit has not written yet. Filtered out here so the
    // dialog cannot offer a choice the server will reject.
    excludeStaffId?: string;
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
                            {(options[c.serviceSlotId] ?? [])
                                .filter(s => s.staffId !== excludeStaffId)
                                .map(s => (
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

export default function StaffSchedulePage() {
    const navigate = useNavigate();
    const { token } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [hasBusiness, setHasBusiness] = useState<boolean | null>(null);
    const [businessWorkingHours, setBusinessWorkingHours] = useState<WorkingHour[]>([]);
    const [staffList, setStaffList] = useState<Staff[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [pageError, setPageError] = useState("");

    const [selectedStaffId, setSelectedStaffId] = useState<string>("");
    const [staffSearch, setStaffSearch] = useState("");
    const [editHours, setEditHours] = useState<WorkingHour[]>([]);
    const [hoursFieldErrors, setHoursFieldErrors] = useState<Record<string, string>>({});
    const [hoursFormError, setHoursFormError] = useState("");
    const [hoursSuccessMessage, setHoursSuccessMessage] = useState("");
    const [isSavingHours, setIsSavingHours] = useState(false);
    const [hoursConflicts, setHoursConflicts] = useState<ServiceSlot[] | null>(null);
    const [hoursConflictError, setHoursConflictError] = useState("");
    const [hoursConflictBusy, setHoursConflictBusy] = useState(false);
    // Slots with no booking that fall outside the new hours — these are
    // unassigned outright (no replacement needed), so we just warn about them
    // in a final confirmation step after any replacement picks are made.
    const [hoursUnbookedConflicts, setHoursUnbookedConflicts] = useState<ServiceSlot[]>([]);
    const [showRemovalConfirm, setShowRemovalConfirm] = useState(false);
    const [pendingReassignments, setPendingReassignments] = useState<SlotReassignment[]>([]);

    const fetchStaff = useCallback(async () => {
        if (!activeToken) return;
        const staffRes = await getBusinessStaff(activeToken);
        if (staffRes.errors?.length) setPageError(staffRes.errors[0].message);
        else setStaffList(staffRes.data?.displayStaff ?? []);
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
                    await fetchStaff();
                }
            } catch {
                setPageError("Failed to load profile.");
            } finally {
                setIsLoading(false);
            }
        };
        init();
    }, [activeToken, fetchStaff]);

    // WorkingHoursEditor expects plain "HH:MM:SS" strings (that's what
    // RegisterStaffPage, its other caller, builds locally when adding a new
    // row) — but a staff's saved hours come back from GraphQL as full RFC
    // datetimes ("0000-01-01T09:30:00Z"). Re-shape before handing them to
    // the editor, or its time dropdowns silently fail to match any option.
    const toEditableHours = (hours: WorkingHour[]): WorkingHour[] =>
        hours.map(wh => ({ day: wh.day, startTime: extractTime(wh.startTime) + ":00", endTime: extractTime(wh.endTime) + ":00" }));

    // Keep the editor's selection valid, and seed it with that staff's
    // current hours whenever the selection changes.
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

    const filteredStaffList = staffList.filter(s => s.name.toLowerCase().includes(staffSearch.toLowerCase()));
    const selectedStaff = staffList.find(s => s.staffId === selectedStaffId);

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
        fetchStaff();
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
                <h1 className="mb-3 fs-1">Staff Schedule</h1>
                <Alert variant="warning">You need to register a business profile before managing staff schedules.</Alert>
                <Button variant="primary" onClick={() => navigate("/register-business")}>Register Business Profile</Button>
            </Container>
        );
    }

    return (
        <Container className="py-5" style={{ maxWidth: 900 }}>
            {pageError && <Alert variant="danger">{pageError}</Alert>}

            <Button
                variant="link"
                className="px-0 mb-2 d-inline-flex align-items-center gap-1 text-decoration-none"
                onClick={() => navigate("/staff")}
            >
                <IconChevronLeft size={14} /> Back to Staff Management
            </Button>
            <h1 className="fs-1 mb-4 text-start">Manage Staff Schedule</h1>

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
                excludeStaffId={selectedStaffId}
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
                <Modal.Header closeButton>
                    <Modal.Title>
                        Remove {hoursUnbookedConflicts.length} slot{hoursUnbookedConflicts.length === 1 ? "" : "s"}?
                    </Modal.Title>
                </Modal.Header>
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
        </Container>
    );
}
