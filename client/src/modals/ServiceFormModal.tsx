import type { Dispatch, SetStateAction } from "react";
import { Alert, Button, Form, Modal } from "react-bootstrap";
import type { ServiceInput, ServiceOptionInput } from "../services/ServiceService";
import { emptyOption, syncDefaultOption } from "../utils/serviceFormHelpers";
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

    const updateOption = (optionIdx: number, field: keyof ServiceOptionInput, value: string) => {
        setFormInput(prev => {
            const pkgs = [...prev.serviceOptions];
            pkgs[optionIdx] = { ...pkgs[optionIdx], [field]: value };
            return { ...prev, serviceOptions: pkgs };
        });
    };

    const addOption = () =>
        setFormInput(prev => ({
            ...prev,
            serviceOptions: [...prev.serviceOptions, emptyOption()],
        }));

    const removeOption = (optionIdx: number) =>
        setFormInput(prev => ({
            ...prev,
            serviceOptions: prev.serviceOptions.filter((_, i) => i !== optionIdx),
        }));

    const updateServiceOptionItem = (optionIdx: number, itemIdx: number, value: string) => {
        setFormInput(prev => {
            const pkgs = [...prev.serviceOptions];
            const items = [...pkgs[optionIdx].serviceOptionItems];
            items[itemIdx] = { serviceOptionItemName: value };
            pkgs[optionIdx] = { ...pkgs[optionIdx], serviceOptionItems: items };
            return { ...prev, serviceOptions: pkgs };
        });
    };

    const addServiceOptionItem = (optionIdx: number) =>
        setFormInput(prev => {
            const pkgs = [...prev.serviceOptions];
            pkgs[optionIdx] = {
                ...pkgs[optionIdx],
                serviceOptionItems: [...pkgs[optionIdx].serviceOptionItems, { serviceOptionItemName: "" }],
            };
            return { ...prev, serviceOptions: pkgs };
        });

    const removeServiceOptionItem = (optionIdx: number, itemIdx: number) =>
        setFormInput(prev => {
            const pkgs = [...prev.serviceOptions];
            pkgs[optionIdx] = {
                ...pkgs[optionIdx],
                serviceOptionItems: pkgs[optionIdx].serviceOptionItems.filter((_, i) => i !== itemIdx),
            };
            return { ...prev, serviceOptions: pkgs };
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
                                setFormInput(prev => ({
                                    ...prev,
                                    serviceName: newName,
                                    serviceOptions: syncDefaultOption(prev.serviceOptions, newName),
                                }));
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
                        <h6 className="mb-0">Service Options</h6>
                        <Button size="sm" variant="outline-secondary" onClick={addOption}>
                            + Add Option
                        </Button>
                    </div>
                    <p className="text-muted small mb-2">
                        A default option is always required. If left unnamed, it will use the service name.
                    </p>
                    {formInput.serviceOptions.map((pkg, optionIdx) => {
                        const isDefault = optionIdx === 0;
                        return (
                            <div key={optionIdx} className="border rounded p-3 mb-3 bg-light">
                                <div className="d-flex justify-content-between align-items-start mb-2">
                                    <strong>Option {optionIdx + 1}</strong>
                                    {isDefault ? (
                                        <small className="text-muted fst-italic">Default option</small>
                                    ) : (
                                        <Button
                                            size="sm"
                                            variant="outline-danger"
                                            onClick={() => removeOption(optionIdx)}
                                        >
                                            Remove
                                        </Button>
                                    )}
                                </div>

                                {fieldErrors[`serviceOptions[${optionIdx}]`] && (
                                    <div className="text-danger small mb-2">
                                        {fieldErrors[`serviceOptions[${optionIdx}]`]}
                                    </div>
                                )}

                                <Form.Group className="mb-2">
                                    <Form.Label>Option Name </Form.Label>
                                    <Form.Control
                                        type="text"
                                        value={pkg.serviceOptionName}
                                        onChange={e => updateOption(optionIdx, "serviceOptionName", e.target.value)}
                                        maxLength={FIELD_LIMITS.serviceOptionName}
                                        isInvalid={!!fieldErrors[`serviceOptionName[${optionIdx}]`]}
                                    />
                                    <Form.Control.Feedback type="invalid">{fieldErrors[`serviceOptionName[${optionIdx}]`]}</Form.Control.Feedback>
                                </Form.Group>

                                <Form.Group className="mb-3">
                                    <Form.Label>Option Description</Form.Label>
                                    <Form.Control
                                        type="text"
                                        value={pkg.description ?? ""}
                                        onChange={e => updateOption(optionIdx, "description", e.target.value)}
                                    />
                                </Form.Group>

                                <div className="d-flex justify-content-between align-items-center mb-2">
                                    <small className="text-muted fw-semibold">Service Option Items</small>
                                    <Button size="sm" variant="link" onClick={() => addServiceOptionItem(optionIdx)}>
                                        + Add Item
                                    </Button>
                                </div>

                                {pkg.serviceOptionItems.map((item, itemIdx) => (
                                    <div key={itemIdx} className="mb-2">
                                        <div className="d-flex gap-2">
                                            <Form.Control
                                                type="text"
                                                placeholder="Item name"
                                                value={item.serviceOptionItemName}
                                                onChange={e => updateServiceOptionItem(optionIdx, itemIdx, e.target.value)}
                                                maxLength={FIELD_LIMITS.serviceOptionItemName}
                                            />
                                            <Button
                                                size="sm"
                                                variant="outline-danger"
                                                onClick={() => removeServiceOptionItem(optionIdx, itemIdx)}
                                            >
                                                ✕
                                            </Button>
                                        </div>
                                        {fieldErrors[`serviceOptionItemName[${optionIdx}][${itemIdx}]`] && (
                                            <div className="text-danger small mt-1">
                                                {fieldErrors[`serviceOptionItemName[${optionIdx}][${itemIdx}]`]}
                                            </div>
                                        )}
                                    </div>
                                ))}
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
