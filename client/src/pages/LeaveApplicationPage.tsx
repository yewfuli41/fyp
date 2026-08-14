import React, { useState, useEffect, useCallback } from "react";
import { Alert, Badge, Button, Card, Container, Form, Modal, Spinner, Table } from "react-bootstrap";
import { useAuth } from "../auth/AuthContext";
import {
    applyLeave, myLeaveApplications, updateLeaveApplication, deleteLeaveApplication,
    type LeaveApplication, type LeaveStatus,
} from "../services/LeaveService";
import { parseGraphQLErrors } from "../utils/graphqlErrors";
import { todayISO } from "../utils/serviceSlotHelpers";
import { IconHistory, IconPencil, IconDownload } from "../components/icons";
import { notifyPendingCountsChanged } from "../utils/pendingCounts";
import ConfirmDeleteModal from "../modals/ConfirmDeleteModal";

const STATUS_VARIANT: Record<LeaveStatus, string> = {
    PENDING: "warning",
    APPROVED: "success",
    REJECTED: "danger",
};

export default function LeaveApplicationPage() {
    const { token } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [applications, setApplications] = useState<LeaveApplication[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [pageError, setPageError] = useState("");

    // Apply Leave form
    const [startDate, setStartDate] = useState("");
    const [endDate, setEndDate] = useState("");
    const [reason, setReason] = useState("");
    const [fileName, setFileName] = useState("");
    const [fileDataUrl, setFileDataUrl] = useState("");
    const [formError, setFormError] = useState("");
    const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
    const [isSubmitting, setIsSubmitting] = useState(false);

    const MAX_FILE_BYTES = 5 * 1024 * 1024; // 5 MB, matches the server-side cap

    const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) {
            setFileName("");
            setFileDataUrl("");
            return;
        }
        if (file.size > MAX_FILE_BYTES) {
            setFieldErrors(prev => ({ ...prev, fileUrl: "Attachment is too large (max 5 MB)." }));
            e.target.value = "";
            setFileName("");
            setFileDataUrl("");
            return;
        }
        setFieldErrors(prev => ({ ...prev, fileUrl: "" }));
        const reader = new FileReader();
        reader.onload = () => {
            setFileDataUrl(typeof reader.result === "string" ? reader.result : "");
            setFileName(file.name);
        };
        reader.readAsDataURL(file);
    };

    const clearFile = () => {
        setFileName("");
        setFileDataUrl("");
        setFieldErrors(prev => ({ ...prev, fileUrl: "" }));
    };

    // Delete confirm
    const [deleting, setDeleting] = useState<LeaveApplication | null>(null);
    const [deleteError, setDeleteError] = useState("");
    const [isDeleting, setIsDeleting] = useState(false);

    // Editing a pending application's reason, inline in the table row
    const [editingId, setEditingId] = useState<string | null>(null);
    const [editValue, setEditValue] = useState("");
    const [editError, setEditError] = useState("");
    const [isSavingEdit, setIsSavingEdit] = useState(false);

    // Rejection reason viewer
    const [viewingRemark, setViewingRemark] = useState<LeaveApplication | null>(null);

    // History modal — past applications (endDate already gone by) are
    // hidden from the default table, which only needs to answer "what's
    // still current or coming up"; History is where old ones live on.
    const [showHistory, setShowHistory] = useState(false);
    const [historyDate, setHistoryDate] = useState("");

    const fetchApplications = useCallback(async () => {
        if (!activeToken) return;
        const result = await myLeaveApplications(activeToken);
        if (result.errors?.length) setPageError(result.errors[0].message);
        else setApplications(result.data?.myLeaveApplications ?? []);
    }, [activeToken]);

    useEffect(() => {
        fetchApplications().finally(() => setIsLoading(false));
    }, [fetchApplications]);

    const handleApply = async () => {
        if (!activeToken) return;
        setIsSubmitting(true);
        setFormError("");
        setFieldErrors({});
        try {
            // End date is optional — a single-day leave only needs the start date.
            const result = await applyLeave(activeToken, {
                startDate, endDate: endDate || startDate,
                justification: reason || undefined,
                fileUrl: fileDataUrl || undefined,
            });
            const parsed = parseGraphQLErrors(result, "Failed to apply for leave");
            if (parsed.hasErrors) {
                setFieldErrors(parsed.fieldErrors);
                setFormError(parsed.formError || Object.values(parsed.fieldErrors)[0] || "");
                return;
            }
            setStartDate("");
            setEndDate("");
            setReason("");
            clearFile();
            fetchApplications();
        } catch {
            setFormError("Something went wrong. Please try again.");
        } finally {
            setIsSubmitting(false);
        }
    };

    const startEdit = (app: LeaveApplication) => {
        setEditingId(app.leaveId);
        setEditValue(app.justification ?? "");
        setEditError("");
    };

    const cancelEdit = () => {
        setEditingId(null);
        setEditError("");
    };

    const saveEdit = async () => {
        if (!activeToken || !editingId) return;
        setIsSavingEdit(true);
        setEditError("");
        try {
            const result = await updateLeaveApplication(activeToken, editingId, editValue);
            const parsed = parseGraphQLErrors(result, "Failed to update reason");
            if (parsed.hasErrors) {
                setEditError(parsed.formError || Object.values(parsed.fieldErrors)[0] || "Failed to update reason");
                return;
            }
            setEditingId(null);
            fetchApplications();
        } catch {
            setEditError("Something went wrong. Please try again.");
        } finally {
            setIsSavingEdit(false);
        }
    };

    const handleDelete = async () => {
        if (!activeToken || !deleting) return;
        setIsDeleting(true);
        setDeleteError("");
        try {
            const result = await deleteLeaveApplication(activeToken, deleting.leaveId);
            if (result.errors?.length) {
                setDeleteError(result.errors[0].message);
            } else {
                setDeleting(null);
                fetchApplications();
                // Deleting a decided application (the only kind the navbar's
                // "My Leave" badge counts) should drop the badge right away.
                notifyPendingCountsChanged();
            }
        } catch {
            setDeleteError("Something went wrong. Please try again.");
        } finally {
            setIsDeleting(false);
        }
    };

    if (isLoading) {
        return (
            <Container className="d-flex justify-content-center align-items-center" style={{ minHeight: "50vh" }}>
                <Spinner animation="border" variant="primary" />
            </Container>
        );
    }

    const today = todayISO();
    const currentApplications = applications.filter(a => a.endDate >= today);
    const pastApplications = applications.filter(a => a.endDate < today);
    const filteredHistory = pastApplications.filter(a => !historyDate || (a.startDate <= historyDate && a.endDate >= historyDate));

    return (
        <Container className="py-5" style={{ maxWidth: 900 }}>
            <div className="d-flex justify-content-between align-items-center mb-4">
                <h1 className="fs-1 mb-0">Leave Applications</h1>
                <Button
                    variant="outline-secondary"
                    className="d-inline-flex align-items-center gap-2"
                    onClick={() => { setHistoryDate(""); setShowHistory(true); }}
                >
                    <IconHistory size={15} /> History
                </Button>
            </div>

            {pageError && <Alert variant="danger">{pageError}</Alert>}

            {applications.length === 0 ? (
                <Alert variant="info">You haven't applied for any leave yet.</Alert>
            ) : currentApplications.length === 0 ? (
                <Alert variant="info">
                    No current or upcoming leave applications.{" "}
                    <Button variant="link" className="p-0 align-baseline" onClick={() => setShowHistory(true)}>
                        View history →
                    </Button>
                </Alert>
            ) : (
                <Table bordered hover responsive className="align-middle mb-5">
                    <thead>
                        <tr>
                            <th>Dates</th>
                            <th>Reason</th>
                            <th>Status</th>
                            <th style={{ width: 140 }}>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {currentApplications.map(app => (
                            <tr key={app.leaveId}>
                                <td>
                                    <div>{app.startDate === app.endDate ? app.startDate : `${app.startDate} – ${app.endDate}`}</div>
                                    {app.fileUrl && (
                                        <a
                                            href={app.fileUrl}
                                            download={`leave-${app.leaveId}-attachment`}
                                            className="d-inline-flex align-items-center gap-1 small mt-1"
                                        >
                                            <IconDownload size={12} /> Download
                                        </a>
                                    )}
                                </td>
                                <td style={{ minWidth: 220 }}>
                                    {editingId === app.leaveId ? (
                                        <div>
                                            <Form.Control
                                                as="textarea"
                                                rows={2}
                                                autoFocus
                                                value={editValue}
                                                onChange={e => setEditValue(e.target.value)}
                                                isInvalid={!!editError}
                                            />
                                            {editError && (
                                                <div className="invalid-feedback d-block">{editError}</div>
                                            )}
                                            <div className="d-flex gap-2 mt-2">
                                                <Button
                                                    size="sm" variant="primary"
                                                    onClick={saveEdit} disabled={isSavingEdit}
                                                >
                                                    {isSavingEdit ? "Saving..." : "Save"}
                                                </Button>
                                                <Button
                                                    size="sm" variant="outline-secondary"
                                                    onClick={cancelEdit} disabled={isSavingEdit}
                                                >
                                                    Cancel
                                                </Button>
                                            </div>
                                        </div>
                                    ) : (
                                        <div className="d-flex align-items-start justify-content-between gap-2">
                                            {app.justification || <span className="text-muted fst-italic">None</span>}
                                            {app.status === "PENDING" && (
                                                <Button
                                                    variant="link" size="sm"
                                                    className="p-0 flex-shrink-0 d-inline-flex align-items-center gap-1"
                                                    onClick={() => startEdit(app)}
                                                >
                                                    <IconPencil size={13} /> Edit
                                                </Button>
                                            )}
                                        </div>
                                    )}
                                </td>
                                <td>
                                    <Badge bg={STATUS_VARIANT[app.status]}>{app.status}</Badge>
                                    {app.status === "REJECTED" && app.remark && (
                                        <Button
                                            variant="link" size="sm" className="p-0 ms-2"
                                            onClick={() => setViewingRemark(app)}
                                        >
                                            Why?
                                        </Button>
                                    )}
                                </td>
                                <td>
                                    <Button
                                        variant="outline-danger" size="sm"
                                        onClick={() => { setDeleting(app); setDeleteError(""); }}
                                    >
                                        Delete
                                    </Button>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </Table>
            )}

            <Card style={{ maxWidth: 500 }}>
                <Card.Body>
                    <Card.Title className="mb-3">Apply Leave</Card.Title>
                    <Form.Group className="mb-3">
                        <Form.Label>Start Date <span className="text-danger">*</span></Form.Label>
                        <Form.Control
                            type="date"
                            min={todayISO()}
                            value={startDate}
                            onChange={e => {
                                setStartDate(e.target.value);
                                setFieldErrors(prev => ({ ...prev, startDate: "" }));
                            }}
                            isInvalid={!!fieldErrors.startDate}
                        />
                        <Form.Control.Feedback type="invalid">{fieldErrors.startDate}</Form.Control.Feedback>
                    </Form.Group>
                    <Form.Group className="mb-3">
                        <Form.Label>End Date</Form.Label>
                        <Form.Control
                            type="date"
                            min={startDate || todayISO()}
                            placeholder="Same as start date"
                            value={endDate}
                            onChange={e => {
                                setEndDate(e.target.value);
                                setFieldErrors(prev => ({ ...prev, endDate: "" }));
                            }}
                            isInvalid={!!fieldErrors.endDate}
                        />
                        <Form.Text muted>Leave blank for a single-day leave.</Form.Text>
                        <Form.Control.Feedback type="invalid">{fieldErrors.endDate}</Form.Control.Feedback>
                    </Form.Group>
                    <Form.Group className="mb-3">
                        <Form.Label>Reason</Form.Label>
                        <Form.Control
                            as="textarea"
                            rows={3}
                            value={reason}
                            onChange={e => setReason(e.target.value)}
                            isInvalid={!!fieldErrors.justification}
                        />
                        <Form.Control.Feedback type="invalid">{fieldErrors.justification}</Form.Control.Feedback>
                    </Form.Group>
                    <Form.Group className="mb-3">
                        <Form.Label>Supporting Document</Form.Label>
                        <Form.Control
                            type="file"
                            accept="image/*,.pdf"
                            onChange={handleFileChange}
                            isInvalid={!!fieldErrors.fileUrl}
                        />
                        <Form.Text muted>Optional, e.g. a medical certificate. Max 5 MB.</Form.Text>
                        <Form.Control.Feedback type="invalid">{fieldErrors.fileUrl}</Form.Control.Feedback>
                        {fileName && (
                            <div className="d-flex align-items-center gap-2 mt-1 small">
                                <span className="text-muted">{fileName}</span>
                                <Button variant="link" size="sm" className="p-0" onClick={clearFile}>
                                    Remove
                                </Button>
                            </div>
                        )}
                    </Form.Group>
                    {formError && <Alert variant="danger" className="py-2">{formError}</Alert>}
                    <Button
                        variant="primary" onClick={handleApply}
                        disabled={isSubmitting || !startDate}
                    >
                        {isSubmitting ? "Submitting..." : "Submit"}
                    </Button>
                </Card.Body>
            </Card>

            <ConfirmDeleteModal
                show={!!deleting}
                title="Delete Leave Application"
                itemName={deleting ? `leave from ${deleting.startDate} to ${deleting.endDate}` : ""}
                error={deleteError}
                isDeleting={isDeleting}
                onCancel={() => setDeleting(null)}
                onConfirm={handleDelete}
            />

            <ConfirmDeleteModal
                show={!!viewingRemark}
                title="Rejection Reason"
                error={viewingRemark?.remark ?? ""}
                isDeleting={false}
                onCancel={() => setViewingRemark(null)}
                onConfirm={() => setViewingRemark(null)}
            />

            <Modal show={showHistory} onHide={() => setShowHistory(false)} size="lg">
                <Modal.Header closeButton><Modal.Title>Leave History</Modal.Title></Modal.Header>
                <Modal.Body>
                    <Form.Group className="mb-3" style={{ maxWidth: 220 }}>
                        <Form.Label className="small fw-semibold">Filter by date</Form.Label>
                        <Form.Control type="date" value={historyDate} onChange={e => setHistoryDate(e.target.value)} />
                    </Form.Group>

                    {filteredHistory.length === 0 ? (
                        <Alert variant="info" className="mb-0">
                            {pastApplications.length === 0 ? "No past leave applications." : "No past applications match that date."}
                        </Alert>
                    ) : (
                        <div style={{ maxHeight: 420, overflowY: "auto" }}>
                            <Table bordered hover responsive className="align-middle mb-0">
                                <thead>
                                    <tr>
                                        <th>Dates</th>
                                        <th>Reason</th>
                                        <th>Status</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {filteredHistory.map(app => (
                                        <tr key={app.leaveId}>
                                            <td>
                                                <div>{app.startDate === app.endDate ? app.startDate : `${app.startDate} – ${app.endDate}`}</div>
                                                {app.fileUrl && (
                                                    <a
                                                        href={app.fileUrl}
                                                        download={`leave-${app.leaveId}-attachment`}
                                                        className="d-inline-flex align-items-center gap-1 small mt-1"
                                                    >
                                                        <IconDownload size={12} /> Download
                                                    </a>
                                                )}
                                            </td>
                                            <td>{app.justification || <span className="text-muted fst-italic">None</span>}</td>
                                            <td>
                                                <Badge bg={STATUS_VARIANT[app.status]}>{app.status}</Badge>
                                                {app.status === "REJECTED" && app.remark && (
                                                    <Button
                                                        variant="link" size="sm" className="p-0 ms-2"
                                                        onClick={() => setViewingRemark(app)}
                                                    >
                                                        Why?
                                                    </Button>
                                                )}
                                            </td>
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
