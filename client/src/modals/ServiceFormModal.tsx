import type { Dispatch, SetStateAction } from "react";
import { Alert, Badge, Button, Form, Modal } from "react-bootstrap";
import type { ServiceInput, ServiceOptionInput } from "../services/ServiceService";
import { emptyOption, syncDefaultOption, getOptionStatus, OPTION_STATUS_LABEL, OPTION_STATUS_VARIANT } from "../utils/serviceFormHelpers";
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
            const next = { ...pkgs[optionIdx], [field]: value };
            // A brand-new option's 'effective until' needs an 'effective from' —
            // default from to today if the owner sets an until without one. An
            // existing option's start stays untouched (blank = "no change")
            // unless the owner explicitly moves it themselves.
            if (field === "effectiveUntil" && value && !next.effectiveFrom && !next.serviceOptionId) {
                next.effectiveFrom = new Date().toISOString().slice(0, 10);
            }
            pkgs[optionIdx] = next;
            return { ...prev, serviceOptions: pkgs };
        });
        if (field === "effectiveFrom" || field === "effectiveUntil") {
            setFieldErrors(prev => ({ ...prev, [`serviceOptions[${optionIdx}]`]: "" }));
        }
    };

    const addOption = () =>
        setFormInput(prev => ({
            ...prev,
            serviceOptions: [...prev.serviceOptions, emptyOption()],
        }));

    // Deletes the option outright. A booked option can never actually be
    // removed server-side (see serviceService.go) — checked here immediately,
    // right on this option's own card, rather than letting the owner discover
    // it only after a full form submit (by which point the option has
    // already vanished from the list with nothing left to attach the error to
    // — that later, list-level case is still handled by the bare
    // fieldErrors.serviceOptions alert above, for the rare race where a
    // booking lands between opening this form and hitting Save).
    const removeOption = (optionIdx: number) => {
        const pkg = formInput.serviceOptions[optionIdx];
        if (pkg.hasBooking) {
            setFieldErrors(prev => ({
                ...prev,
                [`serviceOptions[${optionIdx}]`]: `Can't delete "${pkg.serviceOptionName}" — it has a booking.`,
            }));
            return;
        }
        setFormInput(prev => ({
            ...prev,
            serviceOptions: prev.serviceOptions.filter((_, i) => i !== optionIdx),
        }));
    };

    // Moves an option to the front of the list, making it the default (the
    // default is always array position 0 — see serviceService.go). The
    // default can't be deleted or given an end date, so a service always has
    // one available option.
    const setAsDefault = (optionIdx: number) => {
        const pkg = formInput.serviceOptions[optionIdx];
        // Untouched (undefined) → fall back to the option's already-saved end
        // date. Explicitly cleared ("") → the owner just removed it, so it no
        // longer counts, even though currentEffectiveUntil still shows the
        // stale pre-edit snapshot.
        const hasScheduledEnd = pkg?.effectiveUntil !== undefined
            ? !!pkg.effectiveUntil
            : !!pkg?.currentEffectiveUntil;
        if (hasScheduledEnd) {
            setFieldErrors(errors => ({
                ...errors,
                [`serviceOptions[${optionIdx}]`]: "This option has an effective-until date set. Clear it (or delete this option and add a replacement without one), then set it as default.",
            }));
            return;
        }

        setFormInput(prev => {
            if (optionIdx === 0) return prev;
            const pkgs = [...prev.serviceOptions];
            const [chosen] = pkgs.splice(optionIdx, 1);
            pkgs.unshift(chosen);
            return { ...prev, serviceOptions: pkgs };
        });
    };

    const updateServiceOptionItem = (optionIdx: number, itemIdx: number, value: string) => {
        setFormInput(prev => {
            const pkgs = [...prev.serviceOptions];
            const items = [...pkgs[optionIdx].serviceOptionItems];
            items[itemIdx] = { ...items[itemIdx], serviceOptionItemName: value };
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

                    {/* Deleting an option removes it from this list immediately (see
                        removeOption below), so if the server rejects the deletion —
                        the option turned out to have a booking — there's no longer a
                        row to attach the error to. The backend reports it under the
                        bare "serviceOptions" field (no index) for exactly this case. */}
                    {fieldErrors.serviceOptions && (
                        <Alert variant="danger" className="py-2 small mb-2">{fieldErrors.serviceOptions}</Alert>
                    )}

                    {formInput.serviceOptions.map((pkg, optionIdx) => {
                        const isDefault = optionIdx === 0;
                        const isExisting = !!pkg.serviceOptionId;

                        const today = new Date().toISOString().slice(0, 10);
                        // An existing option's window can be resized, so prefill
                        // from/until with their real current values when untouched.
                        const fromValue = isExisting
                            ? (pkg.effectiveFrom ?? pkg.currentEffectiveFrom ?? "")
                            : (pkg.effectiveFrom ?? "");
                        const untilValue = isExisting
                            ? (pkg.effectiveUntil ?? pkg.currentEffectiveUntil ?? "")
                            : (pkg.effectiveUntil ?? "");
                        // A saved option's real start date is usually well in the
                        // past — don't set min=today in that case, or the browser
                        // treats the displayed value as out-of-range (new picks are
                        // still guided to today+ once the owner actually changes it,
                        // and the backend enforces "not in the past" either way).
                        const fromMin = fromValue && fromValue < today ? undefined : today;
                        const status = getOptionStatus(fromValue, untilValue);

                        return (
                            <div key={optionIdx} className="border rounded p-3 mb-3 bg-light">
                                <div className="d-flex justify-content-between align-items-start mb-2">
                                    <strong>
                                        Option {optionIdx + 1}
                                        <Badge bg={OPTION_STATUS_VARIANT[status]} className="ms-2">{OPTION_STATUS_LABEL[status]}</Badge>
                                    </strong>
                                    <div className="d-flex gap-2 align-items-center">
                                        {!isDefault && (
                                            <Button size="sm" variant="outline-primary" onClick={() => setAsDefault(optionIdx)}>
                                                Set as default
                                            </Button>
                                        )}
                                        {isDefault ? (
                                            <small className="text-muted fst-italic">Default option</small>
                                        ) : (
                                            <Button
                                                size="sm"
                                                variant={pkg.hasBooking ? "outline-secondary" : "outline-danger"}
                                                onClick={() => removeOption(optionIdx)}
                                            >
                                                Delete
                                            </Button>
                                        )}
                                    </div>
                                </div>

                                <div className="d-flex gap-3 mb-2">
                                    <Form.Group className="flex-fill">
                                        <Form.Label className="small mb-1">
                                            Effective from {!isExisting && <span className="text-danger">*</span>}
                                        </Form.Label>
                                        <Form.Control
                                            type="date"
                                            min={fromMin}
                                            value={fromValue}
                                            onChange={e => updateOption(optionIdx, "effectiveFrom", e.target.value)}
                                            isInvalid={!isExisting && !!fieldErrors[`serviceOptionEffectiveFrom[${optionIdx}]`]}
                                        />
                                        <Form.Control.Feedback type="invalid">
                                            {fieldErrors[`serviceOptionEffectiveFrom[${optionIdx}]`]}
                                        </Form.Control.Feedback>
                                    </Form.Group>
                                    {!isDefault && (
                                        <Form.Group className="flex-fill">
                                            <Form.Label className="small mb-1">Effective until <span className="text-muted">(optional)</span></Form.Label>
                                            <Form.Control
                                                type="date"
                                                min={fromValue || today}
                                                value={untilValue}
                                                onChange={e => updateOption(optionIdx, "effectiveUntil", e.target.value)}
                                            />
                                        </Form.Group>
                                    )}
                                </div>

                                {isEditing && isExisting && (
                                    <small className="text-muted d-block mb-2">
                                        {isDefault
                                            ? <>This is the default option — it can't be deleted or given an end date, but you can adjust when it starts (today or later). Set another option as default first if you need to retire this one.</>
                                            : <></>}
                                    </small>
                                )}

                                {!isExisting && (
                                    <small className="text-muted d-block mb-2">
                                        {isDefault
                                            ? <>Pick when this new default option starts being offered. As the default, it can't have an end date.</>
                                            : <>Pick when this new option starts being offered — effective until is still optional; leave it blank to make the option open-ended.</>}
                                    </small>
                                )}

                                {fieldErrors[`serviceOptions[${optionIdx}]`] && (
                                    <div className="text-danger small mb-2">
                                        {fieldErrors[`serviceOptions[${optionIdx}]`]}
                                    </div>
                                )}

                                {isExisting && (
                                    <div className="small text-muted mb-2">
                                        This option is saved — its name and items can't be edited here. Delete it and add a new option instead to change them.
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
                                        disabled={isExisting}
                                    />
                                    <Form.Control.Feedback type="invalid">{fieldErrors[`serviceOptionName[${optionIdx}]`]}</Form.Control.Feedback>
                                </Form.Group>

                                <Form.Group className="mb-3">
                                    <Form.Label>Option Description</Form.Label>
                                    <Form.Control
                                        type="text"
                                        value={pkg.description ?? ""}
                                        onChange={e => updateOption(optionIdx, "description", e.target.value)}
                                        disabled={isExisting}
                                    />
                                </Form.Group>

                                <div className="d-flex justify-content-between align-items-center mb-2">
                                    <small className="text-muted fw-semibold">Service Option Items</small>
                                    {!isExisting && (
                                        <Button size="sm" variant="link" onClick={() => addServiceOptionItem(optionIdx)}>
                                            + Add Item
                                        </Button>
                                    )}
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
                                                disabled={isExisting}
                                            />
                                            {!isExisting && (
                                                <Button
                                                    size="sm"
                                                    variant="outline-danger"
                                                    onClick={() => removeServiceOptionItem(optionIdx, itemIdx)}
                                                >
                                                    ✕
                                                </Button>
                                            )}
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

                    {formError && <Alert variant="danger" className="mb-0">{formError}</Alert>}
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
