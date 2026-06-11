import { useState, useEffect, useCallback } from "react";
import {
    Alert, Button, Card, Col, Container, Form, Modal, Row, Spinner
} from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import {
    getBusinessServices, createService, updateService, deleteService,
    type Service, type ServiceInput, type ServicePackageInput
} from "../services/ServiceService";
import { userProfile } from "../services/ProfileService";
import { applyGraphQLErrors } from "../utils/graphqlErrors";
import { FIELD_LIMITS } from "../utils/fieldLimits";

const emptyPackage = (): ServicePackageInput => ({
    servicePackageName: "",
    description: "",
    packageItems: [{ packageItemName: "" }],
});

const emptyInput = (): ServiceInput => ({
    serviceName: "",
    description: "",
    servicePackages: [],
});

export default function ServicePage() {
    const navigate = useNavigate();
    const { token } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [hasBusiness, setHasBusiness] = useState<boolean | null>(null);
    const [services, setServices] = useState<Service[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [pageError, setPageError] = useState("");

    // Form modal state
    const [showForm, setShowForm] = useState(false);
    const [editingService, setEditingService] = useState<Service | null>(null);
    const [formInput, setFormInput] = useState<ServiceInput>(emptyInput());
    const [formError, setFormError] = useState("");
    const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
    const [isSubmitting, setIsSubmitting] = useState(false);

    // Delete confirm modal state
    const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
    const [deletingService, setDeletingService] = useState<Service | null>(null);
    const [isDeleting, setIsDeleting] = useState(false);
    const [deleteError, setDeleteError] = useState("");

    const fetchServices = useCallback(async () => {
        if (!activeToken) return;
        try {
            const result = await getBusinessServices(activeToken);
            if (result.errors?.length) {
                setPageError(result.errors[0].message);
            } else {
                setServices(result.data?.businessServices ?? []);
            }
        } catch {
            setPageError("Failed to load services.");
        }
    }, [activeToken]);

    useEffect(() => {
        if (!activeToken) return;

        const init = async () => {
            try {
                const profileResult = await userProfile(activeToken);
                const hasBusinessProfile = !!profileResult.data?.userProfile?.businessProfile;
                setHasBusiness(hasBusinessProfile);
                if (hasBusinessProfile) {
                    await fetchServices();
                }
            } catch {
                setPageError("Failed to load profile.");
            } finally {
                setIsLoading(false);
            }
        };

        init();
    }, [activeToken, navigate, fetchServices]);

    // ── Form helpers ───────────────────────────────────────────────────────────

    const openAdd = () => {
        setEditingService(null);
        setFormInput(emptyInput());
        setFormError("");
        setFieldErrors({});
        setShowForm(true);
    };

    const openEdit = (svc: Service) => {
        setEditingService(svc);
        setFormInput({
            serviceName: svc.serviceName,
            description: svc.description ?? "",
            servicePackages: svc.servicePackages.length > 0
                ? svc.servicePackages.map(pkg => ({
                    servicePackageName: pkg.servicePackageName,
                    description: pkg.description ?? "",
                    packageItems: pkg.packageItems.length > 0
                        ? pkg.packageItems.map(item => ({ packageItemName: item.packageItemName }))
                        : [{ packageItemName: "" }],
                }))
                : [emptyPackage()],
        });
        setFormError("");
        setFieldErrors({});
        setShowForm(true);
    };

    const handleFormSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!activeToken) return;

        setFormError("");
        setFieldErrors({});
        setIsSubmitting(true);

        const payload: ServiceInput = {
            serviceName: formInput.serviceName,
            description: formInput.description || undefined,
            servicePackages: formInput.servicePackages
                .filter(pkg => pkg.servicePackageName.trim())
                .map(pkg => ({
                    servicePackageName: pkg.servicePackageName,
                    description: pkg.description || undefined,
                    packageItems: pkg.packageItems.filter(i => i.packageItemName.trim()),
                })),
        };

        try {
            const result = editingService
                ? await updateService(activeToken, editingService.serviceId, payload)
                : await createService(activeToken, payload);

            if (applyGraphQLErrors(result, {
                setFieldErrors,
                setFormError,
                fallbackMessage: "Operation failed",
            })) return;

            setShowForm(false);
            fetchServices();
        } catch {
            setFormError("Something went wrong. Please try again.");
        } finally {
            setIsSubmitting(false);
        }
    };

    // ── Package / item helpers ─────────────────────────────────────────────────

    const updatePackage = (pkgIdx: number, field: keyof ServicePackageInput, value: string) => {
        setFormInput(prev => {
            const pkgs = [...prev.servicePackages];
            pkgs[pkgIdx] = { ...pkgs[pkgIdx], [field]: value };
            return { ...prev, servicePackages: pkgs };
        });
    };

    const addPackage = () =>
        setFormInput(prev => ({ ...prev, servicePackages: [...prev.servicePackages, emptyPackage()] }));

    const removePackage = (pkgIdx: number) =>
        setFormInput(prev => ({
            ...prev,
            servicePackages: prev.servicePackages.filter((_, i) => i !== pkgIdx),
        }));

    const updateItem = (pkgIdx: number, itemIdx: number, value: string) => {
        setFormInput(prev => {
            const pkgs = [...prev.servicePackages];
            const items = [...pkgs[pkgIdx].packageItems];
            items[itemIdx] = { packageItemName: value };
            pkgs[pkgIdx] = { ...pkgs[pkgIdx], packageItems: items };
            return { ...prev, servicePackages: pkgs };
        });
    };

    const addItem = (pkgIdx: number) =>
        setFormInput(prev => {
            const pkgs = [...prev.servicePackages];
            pkgs[pkgIdx] = {
                ...pkgs[pkgIdx],
                packageItems: [...pkgs[pkgIdx].packageItems, { packageItemName: "" }],
            };
            return { ...prev, servicePackages: pkgs };
        });

    const removeItem = (pkgIdx: number, itemIdx: number) =>
        setFormInput(prev => {
            const pkgs = [...prev.servicePackages];
            pkgs[pkgIdx] = {
                ...pkgs[pkgIdx],
                packageItems: pkgs[pkgIdx].packageItems.filter((_, i) => i !== itemIdx),
            };
            return { ...prev, servicePackages: pkgs };
        });

    // ── Delete helpers ─────────────────────────────────────────────────────────

    const openDelete = (svc: Service) => {
        setDeletingService(svc);
        setDeleteError("");
        setShowDeleteConfirm(true);
    };

    const handleDelete = async () => {
        if (!activeToken || !deletingService) return;
        setIsDeleting(true);
        setDeleteError("");
        try {
            const result = await deleteService(activeToken, deletingService.serviceId);
            if (result.errors?.length) {
                setDeleteError(result.errors[0].message);
            } else {
                setShowDeleteConfirm(false);
                fetchServices();
            }
        } catch {
            setDeleteError("Something went wrong. Please try again.");
        } finally {
            setIsDeleting(false);
        }
    };

    // ── Render ─────────────────────────────────────────────────────────────────

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
                <h1 className="mb-3 fs-1">Services</h1>
                <Alert variant="warning" style={{ maxWidth: 1200 }}>
                    You need to register a business profile before managing services.
                </Alert>
                <Button variant="primary" onClick={() => navigate("/register-business")}>
                    Register Business Profile
                </Button>
            </Container>
        );
    }

    return (
        <Container className="py-5">
            <Button
                variant="link"
                className="px-0 mb-2 text-decoration-none"
                onClick={() => navigate("/profile")}
            >
                &larr; Back
            </Button>

            <div className="d-flex justify-content-between align-items-center flex-wrap gap-2 mb-4">
                <h1 className="mb-2 fs-1">Services</h1>
                <div className="d-flex gap-2">
                    <Button variant="primary" onClick={openAdd}>Add Service</Button>
                    <Button variant="outline-secondary" onClick={() => navigate("/service-slots")}>
                        Manage Service Slots
                    </Button>
                </div> 
            </div>

            {pageError && <Alert variant="danger">{pageError}</Alert>}

            {services.length === 0 ? (
                <Alert variant="info" style={{ maxWidth: 1200 }}>No services yet. Click "Add Service" to get started.</Alert>
            ) : (
                <Row className="g-4">
                    {services.map(svc => (
                        <Col key={svc.serviceId} md={6}>
                            <Card className="shadow-sm h-100">
                                <Card.Body className="p-4">
                                    <div className="d-flex justify-content-between align-items-start mb-2">
                                        <h2 className="h4 fw-bold mb-0">{svc.serviceName}</h2>
                                        <div className="d-flex gap-2 flex-shrink-0">
                                            <Button variant="primary" size="sm" onClick={() => openEdit(svc)}>
                                                Edit
                                            </Button>
                                            <Button variant="secondary" size="sm" onClick={() => openDelete(svc)}>
                                                Delete
                                            </Button>
                                        </div>
                                    </div>

                                    <p className="text-muted mb-0 text-start">
                                        {svc.description || "No description"}
                                    </p>
                                    <br></br>
                                    <h3 className="h6 fw-bold mt-3 mb-2">Service Packages</h3>
                                    {svc.servicePackages.length === 0 ? (
                                        <p className="text-muted mb-0">No packages</p>
                                    ) : (
                                        <div className="d-flex flex-column gap-2">
                                            {svc.servicePackages.map(pkg => (
                                                <div key={pkg.servicePackageId}>
                                                    <div className="fw-semibold">
                                                        {pkg.servicePackageName}
                                                        {pkg.description && (
                                                            <span className="text-muted fw-normal ms-1">— {pkg.description}</span>
                                                        )}
                                                    </div>
                                                    {pkg.packageItems.length > 0 && (
                                                        <ul className="mb-0 mt-1 ps-3">
                                                            {pkg.packageItems.map(item => (
                                                                <li key={item.packageItemId} className="text-muted small">
                                                                    {item.packageItemName}
                                                                </li>
                                                            ))}
                                                        </ul>
                                                    )}
                                                </div>
                                            ))}
                                        </div>
                                    )}
                                </Card.Body>
                            </Card>
                        </Col>
                    ))}
                </Row>
            )}

            {/* ── Add / Edit Modal ────────────────────────────────────────────── */}
            <Modal show={showForm} onHide={() => setShowForm(false)} size="lg" backdrop="static">
                <Modal.Header closeButton>
                    <Modal.Title>{editingService ? "Edit Service" : "Add Service"}</Modal.Title>
                </Modal.Header>
                <Form onSubmit={handleFormSubmit}>
                    <Modal.Body>
                        {formError && <Alert variant="danger">{formError}</Alert>}

                        <Form.Group className="mb-3">
                            <Form.Label>Service Name <span className="text-danger">*</span></Form.Label>
                            <Form.Control
                                type="text"
                                value={formInput.serviceName}
                                onChange={e => setFormInput(prev => ({ ...prev, serviceName: e.target.value }))}
                                maxLength={FIELD_LIMITS.serviceName}
                                isInvalid={!!fieldErrors.serviceName}
                            />
                            <Form.Control.Feedback type="invalid">{fieldErrors.serviceName}</Form.Control.Feedback>
                        </Form.Group>

                        <Form.Group className="mb-4">
                            <Form.Label>Description</Form.Label>
                            <Form.Control
                                as="textarea"
                                rows={2}
                                value={formInput.description}
                                onChange={e => setFormInput(prev => ({ ...prev, description: e.target.value }))}
                            />
                        </Form.Group>

                        <div className="d-flex justify-content-between align-items-center mb-2">
                            <h6 className="mb-0">Service Packages</h6>
                            <Button size="sm" variant="outline-secondary" onClick={addPackage}>
                                + Add Package
                            </Button>
                        </div>

                        {formInput.servicePackages.map((pkg, pkgIdx) => {
                            const isDefault = pkg.servicePackageName === formInput.serviceName;
                            return (
                            <div key={pkgIdx} className="border rounded p-3 mb-3 bg-light">
                                <div className="d-flex justify-content-between align-items-start mb-2">
                                    <strong>Package {pkgIdx + 1}</strong>
                                    {isDefault ? (
                                        <small className="text-muted fst-italic">Default package</small>
                                    ) : (
                                        <Button
                                            size="sm"
                                            variant="outline-danger"
                                            onClick={() => removePackage(pkgIdx)}
                                        >
                                            Remove
                                        </Button>
                                    )}
                                </div>

                                <Form.Group className="mb-2">
                                    <Form.Label>Package Name </Form.Label>
                                    <Form.Control
                                        type="text"
                                        value={pkg.servicePackageName}
                                        onChange={e => updatePackage(pkgIdx, "servicePackageName", e.target.value)}
                                        maxLength={FIELD_LIMITS.servicePackageName}
                                    />
                                </Form.Group>

                                <Form.Group className="mb-3">
                                    <Form.Label>Package Description</Form.Label>
                                    <Form.Control
                                        type="text"
                                        value={pkg.description ?? ""}
                                        onChange={e => updatePackage(pkgIdx, "description", e.target.value)}
                                    />
                                </Form.Group>

                                <div className="d-flex justify-content-between align-items-center mb-2">
                                    <small className="text-muted fw-semibold">Package Items</small>
                                    <Button size="sm" variant="link" onClick={() => addItem(pkgIdx)}>
                                        + Add Item
                                    </Button>
                                </div>

                                {pkg.packageItems.map((item, itemIdx) => (
                                    <div key={itemIdx} className="d-flex gap-2 mb-2">
                                        <Form.Control
                                            type="text"
                                            placeholder="Item name"
                                            value={item.packageItemName}
                                            onChange={e => updateItem(pkgIdx, itemIdx, e.target.value)}
                                            maxLength={FIELD_LIMITS.packageItemName}
                                        />
                                        {pkg.packageItems.length > 1 && (
                                            <Button
                                                size="sm"
                                                variant="outline-danger"
                                                onClick={() => removeItem(pkgIdx, itemIdx)}
                                            >
                                                ✕
                                            </Button>
                                        )}
                                    </div>
                                ))}
                            </div>
                            );
                        })}
                    </Modal.Body>
                    <Modal.Footer>
                        <Button variant="outline-secondary" onClick={() => setShowForm(false)}>
                            Cancel
                        </Button>
                        <Button variant="primary" type="submit" disabled={isSubmitting}>
                            {isSubmitting ? "Saving..." : editingService ? "Save Changes" : "Create Service"}
                        </Button>
                    </Modal.Footer>
                </Form>
            </Modal>

            {/* ── Delete Confirm Modal ────────────────────────────────────────── */}
            <Modal show={showDeleteConfirm} onHide={() => setShowDeleteConfirm(false)}>
                <Modal.Header closeButton>
                    <Modal.Title>Delete Service</Modal.Title>
                </Modal.Header>
                <Modal.Body>
                    {deleteError ? (
                        <Alert variant="danger">{deleteError}</Alert>
                    ) : (
                        <p>
                            Are you sure you want to delete <strong>{deletingService?.serviceName}</strong>?
                            This action cannot be undone.
                        </p>
                    )}
                </Modal.Body>
                <Modal.Footer>
                    <Button variant="outline-secondary" onClick={() => setShowDeleteConfirm(false)}>
                        {deleteError ? "Close" : "Cancel"}
                    </Button>
                    {!deleteError && (
                        <Button variant="danger" onClick={handleDelete} disabled={isDeleting}>
                            {isDeleting ? "Deleting..." : "Delete"}
                        </Button>
                    )}
                </Modal.Footer>
            </Modal>
        </Container>
    );
}
