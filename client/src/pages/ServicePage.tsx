import { useState, useEffect, useCallback } from "react";
import { Alert, Badge, Button, Card, Col, Container, Form, Row, Spinner } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import {
    getBusinessServices, createService, updateService, deleteService,
    type Service, type ServiceInput
} from "../services/ServiceService";
import { userProfile } from "../services/ProfileService";
import { getBusinessBookings } from "../services/BookingService";
import { parseGraphQLErrors } from "../utils/graphqlErrors";
import { emptyInput, getOptionStatus, OPTION_STATUS_LABEL, OPTION_STATUS_VARIANT } from "../utils/serviceFormHelpers";
import ServiceFormModal from "../modals/ServiceFormModal";
import ConfirmDeleteModal from "../modals/ConfirmDeleteModal";

export default function ServicePage() {
    const navigate = useNavigate();
    const { token } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [hasBusiness, setHasBusiness] = useState<boolean | null>(null);
    const [pendingCount, setPendingCount] = useState(0);
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
                    const bookingsRes = await getBusinessBookings(activeToken);
                    const bookings = bookingsRes.data?.businessBookings ?? [];
                    setPendingCount(bookings.filter(b => b.status === "PENDING").length);
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
                    serviceOptionId: pkg.serviceOptionId,
                    serviceOptionName: pkg.serviceOptionName,
                    description: pkg.description ?? "",
                    // Left blank — the owner can move the start date, but starts
                    // from "no change requested" until they touch the field.
                    effectiveFrom: undefined,
                    effectiveUntil: undefined,
                    currentEffectiveFrom: pkg.effectiveFrom,
                    currentEffectiveUntil: pkg.effectiveUntil,
                    hasBooking: pkg.hasBooking,
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

        // Every option but the default (position 0, which falls back to the
        // service name — see ensureDefaultOption server-side) needs a name.
        // These used to be silently dropped from the payload instead of
        // surfacing an error, so a blank-named option would just vanish and
        // the form would close as if the save fully succeeded.
        //
        // Every brand-new option (including the default) now also needs an
        // explicit effective-from date — it's no longer silently defaulted
        // to today, the owner has to pick it.
        const fieldErrs: Record<string, string> = {};
        formInput.serviceOptions.forEach((pkg, i) => {
            if (i > 0 && !pkg.serviceOptionName.trim()) {
                fieldErrs[`serviceOptionName[${i}]`] = "Option name is required";
            }
            if (!pkg.serviceOptionId && !pkg.effectiveFrom) {
                fieldErrs[`serviceOptionEffectiveFrom[${i}]`] = "Effective from is required";
            }
        });
        if (Object.keys(fieldErrs).length > 0) {
            setFieldErrors(fieldErrs);
            setFormError("Please fix the errors above.");
            return;
        }

        setIsSubmitting(true);

        const payload: ServiceInput = {
            serviceName: formInput.serviceName,
            description: formInput.description || undefined,
            serviceOptions: formInput.serviceOptions.map(pkg => ({
                serviceOptionId: pkg.serviceOptionId,
                serviceOptionName: pkg.serviceOptionName,
                description: pkg.description || undefined,
                effectiveFrom: pkg.effectiveFrom || undefined,
                effectiveUntil: pkg.effectiveUntil || undefined,
                // effectiveUntil === "" means the owner explicitly cleared a
                // previously-saved end date — that can't be expressed by
                // effectiveUntil alone (blank looks the same as untouched),
                // so flag it explicitly.
                clearEffectiveUntil: pkg.effectiveUntil === "" && !!pkg.currentEffectiveUntil,
                serviceOptionItems: pkg.serviceOptionItems
                    .filter(i => i.serviceOptionItemName.trim())
                    .map(i => ({ serviceOptionItemName: i.serviceOptionItemName })),
            })),
        };

        try {
            const result = editingService
                ? await updateService(activeToken, editingService.serviceId, payload)
                : await createService(activeToken, payload);

            const parsed = parseGraphQLErrors(result, "Operation failed");
            if (parsed.hasErrors) {
                setFieldErrors(parsed.fieldErrors);
                setFormError(parsed.formError);
                return;
            }

            setShowForm(false);
            fetchServices();
        } catch {
            setFormError("Something went wrong. Please try again.");
        } finally {
            setIsSubmitting(false);
        }
    };

    // ── Delete helpers ─────────────────────────────────────────────────────────

    const serviceHasBooking = (svc: Service) => svc.serviceOptions.some(o => o.hasBooking);

    const openDelete = (svc: Service) => {
        if (serviceHasBooking(svc)) {
            setPageError("Deletion disabled - booking exists.");
            return;
        }
        setPageError("");
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
            <h1 className="mb-2 fs-1 text-start">Services</h1>
            <div className="d-flex justify-content-between align-items-center mb-4">
                <div className="d-flex gap-2">
                    <Button variant="primary" onClick={openAdd}>Add Service</Button>
                    <div className="position-relative">
                        <Button variant="outline-secondary" onClick={() => navigate("/service-slots")}>
                            Manage Service Slots
                        </Button>
                        {pendingCount > 0 && (
                            <Badge
                                bg="danger"
                                pill
                                className="position-absolute top-0 start-100 translate-middle"
                                title={`${pendingCount} pending request${pendingCount === 1 ? "" : "s"}`}
                            >
                                {pendingCount}
                            </Badge>
                        )}
                    </div>
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
                                            <Button
                                                variant={serviceHasBooking(svc) ? "outline-secondary" : "danger"}
                                                size="sm"
                                                onClick={() => openDelete(svc)}
                                            >
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
                                            {svc.serviceOptions.map(pkg => {
                                                const status = getOptionStatus(pkg.effectiveFrom, pkg.effectiveUntil);
                                                return (
                                                    <div key={pkg.serviceOptionId}>
                                                        <div className="fw-semibold">
                                                            {pkg.serviceOptionName}
                                                            <Badge bg={OPTION_STATUS_VARIANT[status]} className="ms-2">{OPTION_STATUS_LABEL[status]}</Badge>
                                                        </div>
                                                        <div className="text-muted small">
                                                            {pkg.effectiveFrom ?? "—"} — {pkg.effectiveUntil ?? "Present"}
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
                                                );
                                            })}
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
                warningNote="This will also delete all of this service's slots that have no existing booking."
                error={deleteError}
                isDeleting={isDeleting}
                onCancel={() => setShowDeleteConfirm(false)}
                onConfirm={handleDelete}
            />
        </Container>
    );
}
