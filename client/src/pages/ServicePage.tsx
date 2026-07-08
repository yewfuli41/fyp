import { useState, useEffect, useCallback } from "react";
import { Alert, Button, Card, Col, Container, Form, Row, Spinner } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import {
    getBusinessServices, createService, updateService, deleteService,
    type Service, type ServiceInput
} from "../services/ServiceService";
import { userProfile } from "../services/ProfileService";
import { applyGraphQLErrors } from "../utils/graphqlErrors";
import { emptyInput } from "../utils/serviceFormHelpers";
import ServiceFormModal from "../modals/ServiceFormModal";
import ConfirmDeleteModal from "../modals/ConfirmDeleteModal";

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
    const [search, setSearch] = useState("");
    const fetchServices = useCallback(async () => {
        if (!activeToken) return;
        try {
            const result = await getBusinessServices(activeToken);
            if (result.errors?.length) {
                setPageError(result.errors[0].message);
            } else {
                setServices(result.data?.displayServices ?? []);
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
            serviceOptions: svc.serviceOptions.length > 0
                ? svc.serviceOptions.map(pkg => ({
                    serviceOptionName: pkg.serviceOptionName,
                    description: pkg.description ?? "",
                    serviceOptionItems: pkg.serviceOptionItems.length > 0
                        ? pkg.serviceOptionItems.map(item => ({ serviceOptionItemName: item.serviceOptionItemName }))
                        : [{ serviceOptionItemName: "" }],
                }))
                : [],
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
            serviceOptions: formInput.serviceOptions
                .filter(pkg => pkg.serviceOptionName.trim())
                .map(pkg => ({
                    serviceOptionName: pkg.serviceOptionName,
                    description: pkg.description || undefined,
                    serviceOptionItems: pkg.serviceOptionItems.filter(i => i.serviceOptionItemName.trim()),
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

    const filteredServices = services.filter(svc => {
        const searchText = search.toLowerCase();

        return (
            svc.serviceName.toLowerCase().includes(searchText) ||

            svc.serviceOptions.some(pkg =>
                pkg.serviceOptionName
                    .toLowerCase()
                    .includes(searchText) ||

                pkg.serviceOptionItems.some(item =>
                    item.serviceOptionItemName
                        .toLowerCase()
                        .includes(searchText)
                )
            )
        );
    });
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

            <h1 className="mb-2 fs-1 text-start">Services</h1>
            <div className="d-flex justify-content-between align-items-center mb-4">
                <div className="d-flex gap-2">
                    <Button variant="primary" onClick={openAdd}>Add Service</Button>
                    <Button variant="outline-secondary" onClick={() => navigate("/service-slots")}>
                        Manage Service Slots
                    </Button>
                </div>
                <Form.Control
                    type="text"
                    placeholder="Search services and options..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    style={{ width: "440px" }}
                />
            </div>

            {pageError && <Alert variant="danger">{pageError}</Alert>}

            {filteredServices.length === 0 ? (
                <Alert variant="info" style={{ maxWidth: 1200 }}>No services yet. Click "Add Service" to get started.</Alert>
            ) : (
                <Row className="g-4">
                    {filteredServices.map(svc => (
                        <Col key={svc.serviceId} md={6}>
                            <Card className="shadow-sm h-100">
                                <Card.Body className="p-4">
                                    <div className="d-flex justify-content-between align-items-start mb-2">
                                        <h2 className="h4 fw-bold mb-0">{svc.serviceName}</h2>
                                        <div className="d-flex gap-2 flex-shrink-0">
                                            <Button variant="primary" size="sm" onClick={() => openEdit(svc)}>
                                                Edit
                                            </Button>
                                            <Button variant="danger" size="sm" onClick={() => openDelete(svc)}>
                                                Delete
                                            </Button>
                                        </div>
                                    </div>

                                    <p className="text-muted mb-0 text-start">
                                        {svc.description || "No description"}
                                    </p>
                                    <br></br>
                                    <h3 className="h6 fw-bold mt-3 mb-2 text-start">Service Options</h3>
                                    {svc.serviceOptions.length === 0 ? (
                                        <p className="text-muted mb-0">No options</p>
                                    ) : (
                                        <div className="d-flex flex-column gap-2 text-start">
                                            {svc.serviceOptions.map(pkg => (
                                                <div key={pkg.serviceOptionId}>
                                                    <div className="fw-semibold">
                                                        {pkg.serviceOptionName}
                                                        {pkg.description && (
                                                            <span className="text-muted fw-normal ms-1">— {pkg.description}</span>
                                                        )}
                                                    </div>
                                                    {pkg.serviceOptionItems.length > 0 && (
                                                        <ul className="mb-0 mt-1 ps-3">
                                                            {pkg.serviceOptionItems.map(item => (
                                                                <li key={item.serviceOptionItemId} className="text-muted small">
                                                                    {item.serviceOptionItemName}
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

            <ServiceFormModal
                show={showForm}
                onHide={() => setShowForm(false)}
                isEditing={!!editingService}
                formInput={formInput}
                setFormInput={setFormInput}
                formError={formError}
                fieldErrors={fieldErrors}
                setFieldErrors={setFieldErrors}
                isSubmitting={isSubmitting}
                onSubmit={handleFormSubmit}
            />

            <ConfirmDeleteModal
                show={showDeleteConfirm}
                title="Delete Service"
                itemName={deletingService?.serviceName}
                error={deleteError}
                isDeleting={isDeleting}
                onCancel={() => setShowDeleteConfirm(false)}
                onConfirm={handleDelete}
            />
        </Container>
    );
}
