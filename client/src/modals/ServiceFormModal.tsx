import type { Dispatch, SetStateAction } from "react";
import { Alert, Button, Form, Modal } from "react-bootstrap";
import type { ServiceInput, ServicePackageInput } from "../services/ServiceService";
import { emptyPackage, syncDefaultItems, syncDefaultPackage } from "../utils/serviceFormHelpers";
import { FIELD_LIMITS } from "../utils/fieldLimits";

interface ServiceFormModalProps {
    show: boolean;
    onHide: () => void;
    isEditing: boolean;
    formInput: ServiceInput;
    setFormInput: Dispatch<SetStateAction<ServiceInput>>;
    formError: string;
    fieldErrors: Record<string, string>;
    setFieldErrors: Dispatch<SetStateAction<Record<string, string>>>;
    isSubmitting: boolean;
    onSubmit: (e: React.FormEvent) => void;
}

// Add / Edit Service modal, including its package and package-item editors.
export default function ServiceFormModal({
    show, onHide, isEditing, formInput, setFormInput, formError, fieldErrors, setFieldErrors, isSubmitting, onSubmit,
}: ServiceFormModalProps) {
    // ── Package / item helpers ─────────────────────────────────────────────────

    const updatePackage = (pkgIdx: number, field: keyof ServicePackageInput, value: string) => {
        setFormInput(prev => {
            const pkgs = [...prev.servicePackages];
            pkgs[pkgIdx] = { ...pkgs[pkgIdx], [field]: value };
            return { ...prev, servicePackages: pkgs };
        });
    };

    const addPackage = () =>
        setFormInput(prev => ({
            ...prev,
            servicePackages: [...prev.servicePackages, emptyPackage(prev.serviceName)],
        }));

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

    return (
        <Modal show={show} onHide={onHide} size="lg" backdrop="static">
            <Modal.Header closeButton>
                <Modal.Title>{isEditing ? "Edit Service" : "Add Service"}</Modal.Title>
            </Modal.Header>
            <Form onSubmit={onSubmit}>
                <Modal.Body>
                    {formError && <Alert variant="danger">{formError}</Alert>}

                    <Form.Group className="mb-3">
                        <Form.Label>Service Name <span className="text-danger">*</span></Form.Label>
                        <Form.Control
                            type="text"
                            value={formInput.serviceName}
                            onChange={e => {
                                const newName = e.target.value;
                                setFormInput(prev => {
                                    const withDefaultPkg = syncDefaultPackage(prev.servicePackages, newName);
                                    const withDefaultItems = syncDefaultItems(withDefaultPkg, newName);
                                    return { ...prev, serviceName: newName, servicePackages: withDefaultItems };
                                });
                                setFieldErrors(prev => ({ ...prev, serviceName: "" }));
                            }}
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

                    <div className="d-flex justify-content-between align-items-center mb-1">
                        <h6 className="mb-0">Service Packages</h6>
                        <Button size="sm" variant="outline-secondary" onClick={addPackage}>
                            + Add Package
                        </Button>
                    </div>
                    <p className="text-muted small mb-2">
                       A default package with the same name as the service is created automatically, allowing customers to book the service without selecting a specific package.
                    </p>

                    {formInput.servicePackages.map((pkg, pkgIdx) => {
                        const isDefault = pkgIdx === 0;
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

                                {fieldErrors[`servicePackages[${pkgIdx}]`] && (
                                    <div className="text-danger small mb-2">
                                        {fieldErrors[`servicePackages[${pkgIdx}]`]}
                                    </div>
                                )}

                                <Form.Group className="mb-2">
                                    <Form.Label>Package Name </Form.Label>
                                    <Form.Control
                                        type="text"
                                        value={pkg.servicePackageName}
                                        onChange={e => updatePackage(pkgIdx, "servicePackageName", e.target.value)}
                                        maxLength={FIELD_LIMITS.servicePackageName}
                                        isInvalid={!!fieldErrors[`servicePackageName[${pkgIdx}]`]}
                                    />
                                    <Form.Control.Feedback type="invalid">{fieldErrors[`servicePackageName[${pkgIdx}]`]}</Form.Control.Feedback>
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

                                {pkg.packageItems.map((item, itemIdx) => {
                                    const isDefaultItem = itemIdx === 0;
                                    return (
                                    <div key={itemIdx} className="mb-2">
                                        <div className="d-flex gap-2">
                                            <Form.Control
                                                type="text"
                                                placeholder="Item name"
                                                value={item.packageItemName}
                                                onChange={e => updateItem(pkgIdx, itemIdx, e.target.value)}
                                                maxLength={FIELD_LIMITS.packageItemName}
                                                disabled={isDefaultItem}
                                            />
                                            {pkg.packageItems.length > 1 && !isDefaultItem && (
                                                <Button
                                                    size="sm"
                                                    variant="outline-danger"
                                                    onClick={() => removeItem(pkgIdx, itemIdx)}
                                                >
                                                    ✕
                                                </Button>
                                            )}
                                        </div>
                                        {fieldErrors[`packageItemName[${pkgIdx}][${itemIdx}]`] && (
                                            <div className="text-danger small mt-1">
                                                {fieldErrors[`packageItemName[${pkgIdx}][${itemIdx}]`]}
                                            </div>
                                        )}
                                    </div>
                                    );
                                })}
                            </div>
                        );
                    })}
                </Modal.Body>
                <Modal.Footer>
                    <Button variant="outline-secondary" onClick={onHide}>
                        Cancel
                    </Button>
                    <Button variant="primary" type="submit" disabled={isSubmitting}>
                        {isSubmitting ? "Saving..." : isEditing ? "Save Changes" : "Create Service"}
                    </Button>
                </Modal.Footer>
            </Form>
        </Modal>
    );
}
