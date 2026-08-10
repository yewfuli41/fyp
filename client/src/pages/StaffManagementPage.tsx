import { useState, useEffect, useCallback } from "react";
import {
    Alert, Button, Container, Form, OverlayTrigger, Spinner, Table, Tooltip
} from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import {
    getBusinessStaff, updateStaff, deleteStaff,
    type Staff, type UpdateStaffInput
} from "../services/StaffService";
import { userProfile } from "../services/ProfileService";
import { parseGraphQLErrors } from "../utils/graphqlErrors";
import { FIELD_LIMITS, shouldClearEmailError } from "../utils/fieldLimits";
import lockIcon from "../assets/lock.png";
import ConfirmDeleteModal from "../modals/ConfirmDeleteModal";

// Backend validation fields → table columns being edited.
type RowErrors = Partial<Record<"staffName" | "staffEmail" | "staffContactNumber" | "position", string>>;

export default function StaffManagementPage() {
    const navigate = useNavigate();
    const { token } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [hasBusiness, setHasBusiness] = useState<boolean | null>(null);
    const [businessName, setBusinessName] = useState("");
    const [staff, setStaff] = useState<Staff[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [pageError, setPageError] = useState("");
    const [search, setSearch] = useState("");

    // Inline edit state
    const [editingId, setEditingId] = useState<string | null>(null);
    const [draft, setDraft] = useState<UpdateStaffInput>({ name: "", email: "", contactNumber: "", position: "" });
    const [rowErrors, setRowErrors] = useState<RowErrors>({});
    const [rowFormError, setRowFormError] = useState("");
    const [isSaving, setIsSaving] = useState(false);

    // Delete confirm state
    const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
    const [deletingStaff, setDeletingStaff] = useState<Staff | null>(null);
    const [isDeleting, setIsDeleting] = useState(false);
    const [deleteError, setDeleteError] = useState("");

    const fetchStaff = useCallback(async () => {
        if (!activeToken) return;
        try {
            const result = await getBusinessStaff(activeToken);
            if (result.errors?.length) {
                setPageError(result.errors[0].message);
            } else {
                setStaff(result.data?.displayStaff ?? []);
            }
        } catch {
            setPageError("Failed to load staff.");
        }
    }, [activeToken]);

    useEffect(() => {
        if (!activeToken) return;

        const init = async () => {
            try {
                const profileResult = await userProfile(activeToken);
                const business = profileResult.data?.userProfile?.businessProfile;
                setHasBusiness(!!business);
                if (business) {
                    setBusinessName(business.businessName ?? "");
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

    // ── Edit helpers ────────────────────────────────────────────────────────────

    const startEdit = (member: Staff) => {
        setEditingId(member.staffId);
        setDraft({
            name: member.name,
            email: member.email,
            contactNumber: member.contactNumber,
            position: member.position ?? "",
        });
        setRowErrors({});
        setRowFormError("");
    };

    const cancelEdit = () => {
        setEditingId(null);
        setRowErrors({});
        setRowFormError("");
    };

    const handleSave = async (staffId: string) => {
        if (!activeToken) return;
        setIsSaving(true);
        setRowErrors({});
        setRowFormError("");
        try {
            const result = await updateStaff(activeToken, staffId, draft);
            const parsed = parseGraphQLErrors(result, "Failed to update staff");
            if (parsed.hasErrors) {
                const known: RowErrors = {
                    staffName: parsed.fieldErrors.staffName,
                    staffEmail: parsed.fieldErrors.staffEmail,
                    staffContactNumber: parsed.fieldErrors.staffContactNumber,
                    position: parsed.fieldErrors.position,
                };
                setRowErrors(known);
                // Surface any error not tied to an editable column (e.g. staff not found).
                const hasKnownField = Object.values(known).some(Boolean);
                setRowFormError(parsed.formError || (hasKnownField ? "" : "Failed to update staff"));
                return;
            }
            setEditingId(null);
            fetchStaff();
        } catch {
            setRowFormError("Something went wrong. Please try again.");
        } finally {
            setIsSaving(false);
        }
    };

    // ── Delete helpers ──────────────────────────────────────────────────────────

    const openDelete = (member: Staff) => {
        if (member.hasBooking) {
            setPageError("Deletion disabled - booking exists.");
            return;
        }
        setPageError("");
        setDeletingStaff(member);
        setDeleteError("");
        setShowDeleteConfirm(true);
    };

    const handleDelete = async () => {
        if (!activeToken || !deletingStaff) return;
        setIsDeleting(true);
        setDeleteError("");
        try {
            const result = await deleteStaff(activeToken, deletingStaff.staffId);
            if (result.errors?.length) {
                setDeleteError(result.errors[0].message);
            } else {
                setShowDeleteConfirm(false);
                fetchStaff();
            }
        } catch {
            setDeleteError("Something went wrong. Please try again.");
        } finally {
            setIsDeleting(false);
        }
    };

    const filteredStaff = staff.filter(member => {
        const q = search.toLowerCase();
        return (
            member.name.toLowerCase().includes(q) ||
            member.email.toLowerCase().includes(q) ||
            member.contactNumber.toLowerCase().includes(q) ||
            (member.position ?? "").toLowerCase().includes(q)
        );
    });

    // ── Render ──────────────────────────────────────────────────────────────────

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
                <h1 className="mb-3 fs-1">Staff Management</h1>
                <Alert variant="warning" style={{ maxWidth: 1200 }}>
                    You need to register a business profile before managing staff.
                </Alert>
                <Button variant="primary" onClick={() => navigate("/register-business")}>
                    Register Business Profile
                </Button>
            </Container>
        );
    }

    return (
        <Container className="py-5">
            <h1 className="mb-4 fs-1 text-start">
                Staff Management
            </h1>

            <Form.Group className="mb-4" style={{ maxWidth: 400 }}>
                <Form.Label className="fw-bold">Search Staff</Form.Label>
                <Form.Control
                    type="text"
                    placeholder="Search staff..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                />
            </Form.Group>

            <div className="d-flex justify-content-between align-items-center mb-3">
                <h2 className="h4 fw-bold mb-0 text-start">Staff Table</h2>
                <Button variant="primary" onClick={() => navigate("/register-staff")}>
                    + Add Staff
                </Button>
            </div>

            {pageError && <Alert variant="danger">{pageError}</Alert>}

            {filteredStaff.length === 0 ? (
                <Alert variant="info" style={{ maxWidth: 1200 }}>
                    {staff.length === 0
                        ? 'No staff yet. Click "Add Staff" to get started.'
                        : "No staff match your search."}
                </Alert>
            ) : (
                <Table bordered hover responsive className="align-middle">
                    <thead>
                        <tr>
                            <th>Name</th>
                            <th>Contact Number</th>
                            <th>Email</th>
                            <th>Position</th>
                            <th style={{ width: 180 }}>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {filteredStaff.map(member => {
                            const isEditing = editingId === member.staffId;
                            return (
                                <tr key={member.staffId}>
                                    <td>
                                        {isEditing ? (
                                            <Form.Control
                                                type="text"
                                                value={draft.name}
                                                onChange={(e) => {
                                                    setDraft(prev => ({ ...prev, name: e.target.value }));
                                                    setRowErrors(prev => ({ ...prev, staffName: "" }));
                                                }}
                                                maxLength={FIELD_LIMITS.staffName}
                                                isInvalid={!!rowErrors.staffName}
                                            />
                                        ) : (
                                            member.name
                                        )}
                                        {isEditing && rowErrors.staffName && (
                                            <div className="text-danger small mt-1">{rowErrors.staffName}</div>
                                        )}
                                    </td>
                                    <td>
                                        {isEditing ? (
                                            <Form.Control
                                                type="text"
                                                value={draft.contactNumber}
                                                onChange={(e) => {
                                                    setDraft(prev => ({ ...prev, contactNumber: e.target.value }));
                                                    setRowErrors(prev => ({ ...prev, staffContactNumber: "" }));
                                                }}
                                                maxLength={FIELD_LIMITS.staffContactNumber}
                                                isInvalid={!!rowErrors.staffContactNumber}
                                            />
                                        ) : (
                                            member.contactNumber
                                        )}
                                        {isEditing && rowErrors.staffContactNumber && (
                                            <div className="text-danger small mt-1">{rowErrors.staffContactNumber}</div>
                                        )}
                                    </td>
                                    <td>
                                        {/* Email can only be edited before the staff logs in for the
                                            first time (mustResetPassword). Locked rows show a lock icon. */}
                                        {isEditing && member.mustResetPassword ? (
                                            <>
                                                <Form.Control
                                                    type="text"
                                                    value={draft.email}
                                                    onChange={(e) => {
                                                        setDraft(prev => ({ ...prev, email: e.target.value }));
                                                        if (shouldClearEmailError(rowErrors.staffEmail, e.target.value))
                                                            setRowErrors(prev => ({ ...prev, staffEmail: "" }));
                                                    }}
                                                    maxLength={FIELD_LIMITS.email}
                                                    isInvalid={!!rowErrors.staffEmail}
                                                />
                                                {rowErrors.staffEmail && (
                                                    <div className="text-danger small mt-1">{rowErrors.staffEmail}</div>
                                                )}
                                            </>
                                        ) : (
                                            <span className="d-inline-flex align-items-center gap-1">
                                                {member.email}
                                                {!member.mustResetPassword && (
                                                    <OverlayTrigger
                                                        placement="top"
                                                        overlay={
                                                            <Tooltip id={`lock-${member.staffId}`}>
                                                                Email can only be changed before the staff logs in for the first time.
                                                            </Tooltip>
                                                        }
                                                    >
                                                        <img
                                                            src={lockIcon}
                                                            alt="Email locked"
                                                            style={{ width: 14, height: 14, cursor: "help" }}
                                                        />
                                                    </OverlayTrigger>
                                                )}
                                            </span>
                                        )}
                                    </td>
                                    <td>
                                        {isEditing ? (
                                            <Form.Control
                                                type="text"
                                                value={draft.position}
                                                onChange={(e) => {
                                                    setDraft(prev => ({ ...prev, position: e.target.value }));
                                                    setRowErrors(prev => ({ ...prev, position: "" }));
                                                }}
                                                maxLength={FIELD_LIMITS.position}
                                                isInvalid={!!rowErrors.position}
                                            />
                                        ) : (
                                            member.position || "-"
                                        )}
                                        {isEditing && rowErrors.position && (
                                            <div className="text-danger small mt-1">{rowErrors.position}</div>
                                        )}
                                    </td>
                                    <td>
                                        {isEditing ? (
                                            <div>
                                                {rowFormError && (
                                                    <div className="text-danger small mb-1">{rowFormError}</div>
                                                )}
                                                <div className="d-flex gap-2">
                                                    <Button
                                                        variant="primary"
                                                        size="sm"
                                                        onClick={() => handleSave(member.staffId)}
                                                        disabled={isSaving}
                                                    >
                                                        {isSaving ? "Saving..." : "Save"}
                                                    </Button>
                                                    <Button
                                                        variant="outline-secondary"
                                                        size="sm"
                                                        onClick={cancelEdit}
                                                        disabled={isSaving}
                                                    >
                                                        Cancel
                                                    </Button>
                                                </div>
                                            </div>
                                        ) : (
                                            <div className="d-flex gap-2">
                                                <Button
                                                    variant="outline-primary"
                                                    size="sm"
                                                    onClick={() => startEdit(member)}
                                                    disabled={editingId !== null}
                                                >
                                                    Edit
                                                </Button>
                                                <Button
                                                    variant={member.hasBooking ? "secondary" : "danger"}
                                                    size="sm"
                                                    onClick={() => openDelete(member)}
                                                    disabled={editingId !== null}
                                                >
                                                    Delete
                                                </Button>
                                            </div>
                                        )}
                                    </td>
                                </tr>
                            );
                        })}
                    </tbody>
                </Table>
            )}

            <ConfirmDeleteModal
                show={showDeleteConfirm}
                title="Delete Staff"
                itemName={deletingStaff?.name}
                error={deleteError}
                isDeleting={isDeleting}
                onCancel={() => setShowDeleteConfirm(false)}
                onConfirm={handleDelete}
            />
        </Container>
    );
}
