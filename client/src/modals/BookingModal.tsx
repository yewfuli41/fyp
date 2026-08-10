import { useState, useEffect } from "react";
import { Alert, Badge, Button, Form, Modal, Spinner } from "react-bootstrap";
import { extractTime } from "../services/ServiceSlotService";
import {
    getAvailableSlots, getPublicServices, getPublicStaff,
    type AvailableSlot, type PublicService, type PublicStaff,
} from "../services/PublicService";
import { createBooking } from "../services/BookingService";
import { parseGraphQLErrors } from "../utils/graphqlErrors";
import { NONE_COL, isOptionOfferedOn } from "../utils/serviceSlotHelpers";
import BookingDatePicker from "../components/BookingDatePicker";
import { IconCheck, IconClock, IconUser } from "../components/icons";
import "../styles/BookingModal.css";

function StepHeader({ n, title, hint }: { n: number; title: string; hint?: string }) {
    return (
        <div className="bk-step">
            <span className="bk-step-badge">{n}</span>
            <span className="bk-step-title">{title}</span>
            {hint && <span className="bk-step-hint">— {hint}</span>}
        </div>
    );
}

interface Props {
    show: boolean;
    onHide: () => void;
    businessId: string;
    token: string | null;
    // Jumps straight to a service (and, if still bookable, its option)
    // instead of making the customer pick step 1 again — used when arriving
    // from a card the customer already chose on the services page.
    initialServiceId?: string;
    initialServiceOptionId?: string;
}

interface SelectedSlot {
    slotOptionId: string;
    serviceSlotId: string;
    date: string;
    startTime: string;
    endTime: string;
    staffName?: string;
}

export default function BookingModal({
    show, onHide, businessId, token, initialServiceId, initialServiceOptionId,
}: Props) {
    const today = new Date().toISOString().slice(0, 10);
    // Arrived from a specific option's "Book" button on the services page —
    // service and option are already decided, so steps 1 and 3 (picking
    // them) are skipped entirely rather than just pre-filled.
    const locked = Boolean(initialServiceId && initialServiceOptionId);

    const [services, setServices] = useState<PublicService[]>([]);
    const [staff, setStaff] = useState<PublicStaff[]>([]);
    const [isLoadingMeta, setIsLoadingMeta] = useState(false);

    const [serviceId, setServiceId] = useState("");
    const [serviceOptionId, setServiceOptionId] = useState("");

    const [date, setDate] = useState(today);
    const [staffId, setStaffId] = useState("");
    const [slots, setSlots] = useState<AvailableSlot[]>([]);
    // Multiple slots (across different dates) can be selected before booking.
    const [selected, setSelected] = useState<SelectedSlot[]>([]);
    // Slots per offered option for the current date+staff, used to hide
    // options with nothing bookable from step 3 — keyed by serviceOptionId.
    const [optionSlots, setOptionSlots] = useState<Record<string, AvailableSlot[]>>({});
    const [checkingOptions, setCheckingOptions] = useState(false);
    // A free-text note attached to every booking created in this submission.
    const [description, setDescription] = useState("");
    const [formError, setFormError] = useState("");
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [success, setSuccess] = useState(0);

    useEffect(() => {
        if (!show) return;
        setServices([]);
        setStaff([]);
        setServiceId("");
        setServiceOptionId("");
        setDate(today);
        setStaffId("");
        setSlots([]);
        setOptionSlots({});
        setSelected([]);
        setDescription("");
        setFormError("");
        setSuccess(0);

        setIsLoadingMeta(true);
        Promise.all([getPublicServices(businessId), getPublicStaff(businessId)])
            .then(async ([svcRes, staffRes]) => {
                const svcs = svcRes.data?.publicServices ?? [];
                setServices(svcs);
                setStaff(staffRes.data?.publicStaff ?? []);

                const service = initialServiceId && svcs.find(s => s.serviceId === initialServiceId);
                if (!service) return;
                setServiceId(service.serviceId);

                if (initialServiceOptionId) {
                    // Locked to this exact option — no falling back to a
                    // different one just because today has no slots; the
                    // customer can change the date instead.
                    const option = service.serviceOptions.find(o => o.serviceOptionId === initialServiceOptionId);
                    if (option) {
                        setServiceOptionId(option.serviceOptionId);
                        const map = await checkOptionAvailability([option], today, "");
                        setSlots(map[option.serviceOptionId] ?? []);
                    }
                    return;
                }

                const offered = service.serviceOptions.filter(o => isOptionOfferedOn(o, today));
                const map = await checkOptionAvailability(offered, today, "");
                const firstBookable = offered.find(o => (map[o.serviceOptionId]?.length ?? 0) > 0);
                if (firstBookable) {
                    setServiceOptionId(firstBookable.serviceOptionId);
                    setSlots(map[firstBookable.serviceOptionId]);
                }
            })
            .catch(() => setFormError("Failed to load services."))
            .finally(() => setIsLoadingMeta(false));
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [show, businessId, initialServiceId, initialServiceOptionId]);

    const selectedService = services.find(s => s.serviceId === serviceId);
    const selectedOption = selectedService?.serviceOptions.find(o => o.serviceOptionId === serviceOptionId);
    // Which options are even offerable on the currently chosen date — an
    // option's own effective-from/until window can exclude it.
    const offeredOptions = selectedService?.serviceOptions.filter(o => isOptionOfferedOn(o, date)) ?? [];
    // Of those, only the ones that actually have a bookable slot get shown —
    // no point offering a tier the customer can't do anything with.
    const bookableOptions = offeredOptions.filter(o => (optionSlots[o.serviceOptionId]?.length ?? 0) > 0);

    // Checks every offered option's slots for the given date+staff in
    // parallel, so step 3 can hide options with nothing bookable. Returns the
    // map (rather than relying on the state update) so callers can act on it
    // immediately without waiting for a re-render. `sid` is the raw Select
    // Staff value — NONE_COL means "Owner-managed", filtering to slots with
    // no staff assigned rather than to a specific staff member.
    const checkOptionAvailability = async (
        opts: PublicService["serviceOptions"], d: string, sid: string,
    ): Promise<Record<string, AvailableSlot[]>> => {
        if (opts.length === 0) {
            setOptionSlots({});
            return {};
        }
        const unassignedOnly = sid === NONE_COL;
        const staffFilter = unassignedOnly ? undefined : (sid || undefined);
        setCheckingOptions(true);
        try {
            const entries = await Promise.all(opts.map(async opt => {
                const res = await getAvailableSlots(businessId, opt.serviceOptionId, d, staffFilter, unassignedOnly);
                return [opt.serviceOptionId, res.data?.availableSlots ?? []] as const;
            }));
            const map = Object.fromEntries(entries);
            setOptionSlots(map);
            return map;
        } catch {
            setFormError("Failed to load available slots.");
            setOptionSlots({});
            return {};
        } finally {
            setCheckingOptions(false);
        }
    };

    const handleSelectService = async (id: string) => {
        setServiceId(id);
        setSelected([]); // a different service starts a fresh selection
        setServiceOptionId("");
        setSlots([]);
        const service = services.find(s => s.serviceId === id);
        const offered = service?.serviceOptions.filter(o => isOptionOfferedOn(o, date)) ?? [];
        const map = await checkOptionAvailability(offered, date, staffId);
        // Default to the first bookable option, same as the Add/Manage
        // Service Slot forms do for their own default selection.
        const firstBookable = offered.find(o => (map[o.serviceOptionId]?.length ?? 0) > 0);
        if (firstBookable) {
            setServiceOptionId(firstBookable.serviceOptionId);
            setSlots(map[firstBookable.serviceOptionId]);
        }
    };

    const handleSelectServiceOption = (id: string) => {
        setServiceOptionId(id);
        setSelected([]);
        setSlots(optionSlots[id] ?? []); // already fetched while checking availability
    };

    const handleDateChange = async (d: string) => {
        setDate(d);
        if (!serviceId) return;
        const service = services.find(s => s.serviceId === serviceId);

        if (locked) {
            // Stay on the locked option regardless of the new date's
            // availability — never swap to a different option underneath the
            // customer.
            const option = service?.serviceOptions.find(o => o.serviceOptionId === serviceOptionId);
            if (!option) { setSlots([]); return; }
            const map = await checkOptionAvailability([option], d, staffId);
            setSlots(map[option.serviceOptionId] ?? []);
            return;
        }

        const offered = service?.serviceOptions.filter(o => isOptionOfferedOn(o, d)) ?? [];
        const map = await checkOptionAvailability(offered, d, staffId);
        // The chosen option may still be bookable on the new date — keep it
        // (and the current `selected` picks) rather than resetting.
        if (serviceOptionId && (map[serviceOptionId]?.length ?? 0) > 0) {
            setSlots(map[serviceOptionId]);
            return;
        }
        // Otherwise fall back to the first bookable option on the new date.
        setSelected([]);
        const firstBookable = offered.find(o => (map[o.serviceOptionId]?.length ?? 0) > 0);
        if (firstBookable) {
            setServiceOptionId(firstBookable.serviceOptionId);
            setSlots(map[firstBookable.serviceOptionId]);
        } else {
            setServiceOptionId("");
            setSlots([]);
        }
    };

    const handleStaffChange = async (sid: string) => {
        setStaffId(sid);
        if (locked) {
            const service = services.find(s => s.serviceId === serviceId);
            const option = service?.serviceOptions.find(o => o.serviceOptionId === serviceOptionId);
            if (!option) return;
            const map = await checkOptionAvailability([option], date, sid);
            setSlots(map[option.serviceOptionId] ?? []);
            return;
        }
        const map = await checkOptionAvailability(offeredOptions, date, sid);
        if (serviceOptionId) setSlots(map[serviceOptionId] ?? []);
    };

    const isSelected = (serviceSlotId: string) => selected.some(s => s.serviceSlotId === serviceSlotId);

    const toggleSlot = (slot: AvailableSlot) => {
        const slotOptionId = slot.serviceSlotOptions[0]?.slotOptionId ?? "";
        if (!slotOptionId) return;
        setSelected(prev =>
            prev.some(s => s.serviceSlotId === slot.serviceSlotId)
                ? prev.filter(s => s.serviceSlotId !== slot.serviceSlotId)
                : [...prev, {
                    slotOptionId,
                    serviceSlotId: slot.serviceSlotId,
                    date: slot.date,
                    startTime: slot.startTime,
                    endTime: slot.endTime,
                    staffName: slot.staff?.name,
                }],
        );
    };

    const removeSelected = (serviceSlotId: string) =>
        setSelected(prev => prev.filter(s => s.serviceSlotId !== serviceSlotId));

    const handleBook = async () => {
        if (!token) { setFormError("Please log in to make a booking."); return; }
        if (selected.length === 0) { setFormError("Please select at least one time slot."); return; }
        setIsSubmitting(true);
        setFormError("");
        const trimmedDescription = description.trim() || undefined;
        try {
            const results = await Promise.all(selected.map(async s => {
                try {
                    const res = await createBooking(token, s.slotOptionId, trimmedDescription);
                    const parsed = parseGraphQLErrors(res);
                    const message = Object.values(parsed.fieldErrors)[0] || parsed.formError || undefined;
                    return { s, ok: !parsed.hasErrors, message };
                } catch {
                    return { s, ok: false, message: undefined };
                }
            }));
            const failed = results.filter(r => !r.ok).map(r => r.s);
            const bookedIds = results.filter(r => r.ok).map(r => r.s.serviceSlotId);
            setSuccess(results.length - failed.length);
            setSelected(failed); // keep only the ones that couldn't be booked
            if (failed.length > 0) {
                const messages = [...new Set(
                    results.filter(r => !r.ok).map(r => r.message).filter((m): m is string => !!m)
                )];
                setFormError(
                    messages.length > 0
                        ? messages.join(" ")
                        : `${failed.length} slot(s) could not be booked (they may have just been taken).`
                );
            }
            // Drop the just-booked slots from the visible list immediately, then
            // recheck every offered option's availability so a now-empty option
            // both refreshes its slot list and drops out of step 3's options
            // (a booked slot is no longer "available" so it can't be picked again).
            setSlots(prev => prev.filter(s => !bookedIds.includes(s.serviceSlotId)));
            const optionsToRefresh = locked ? (selectedOption ? [selectedOption] : []) : offeredOptions;
            const map = await checkOptionAvailability(optionsToRefresh, date, staffId);
            if (serviceOptionId) setSlots(map[serviceOptionId] ?? []);
        } catch {
            setFormError("Something went wrong. Please try again.");
        } finally {
            setIsSubmitting(false);
        }
    };

    const optionLabel = (opt: PublicService["serviceOptions"][number]) =>
        opt.serviceOptionItems.length > 0
            ? `${opt.serviceOptionName} - ${opt.serviceOptionItems.map(i => i.serviceOptionItemName).join(", ")}`
            : opt.serviceOptionName;

    const fmtDate = (iso: string) => {
        const d = new Date(`${iso}T00:00:00`);
        return isNaN(d.getTime()) ? iso : d.toLocaleDateString("en-GB", { day: "numeric", month: "short" });
    };

    const allBooked = success > 0 && selected.length === 0;

    // Numbers the visible steps in document order — locked mode skips the
    // service/option pickers, so later steps (date, staff, slots) shift down
    // without needing separate hardcoded numbering per mode.
    let stepCount = 0;
    const step = () => ++stepCount;

    return (
        <Modal show={show} onHide={onHide} centered className="bk-modal">
            <Modal.Header closeButton>
                <Modal.Title>
                    {locked && selectedService && selectedOption
                        ? `${selectedService.serviceName} — ${selectedOption.serviceOptionName}`
                        : "Book a Service"}
                </Modal.Title>
            </Modal.Header>
            <Modal.Body>
                {allBooked ? (
                    <Alert variant="success" className="d-flex align-items-center gap-2 mb-0">
                        <IconCheck size={18} />
                        {success} booking{success === 1 ? "" : "s"} submitted! Pending confirmation.
                    </Alert>
                ) : isLoadingMeta ? (
                    <div className="text-center py-3"><Spinner size="sm" /> Loading services…</div>
                ) : (
                    <>
                        {success > 0 && (
                            <Alert variant="success" className="py-2 d-flex align-items-center gap-2">
                                <IconCheck size={16} />
                                {success} booking{success === 1 ? "" : "s"} submitted.
                            </Alert>
                        )}

                        {!locked && (
                            <Form.Group className="mb-4">
                                <StepHeader n={step()} title="Select Service" />
                                {services.length === 0 ? (
                                    <p className="text-muted mb-0">No services available.</p>
                                ) : (
                                    <Form.Select value={serviceId} onChange={e => handleSelectService(e.target.value)}>
                                        <option value="">Select a service</option>
                                        {services.map(s => (
                                            <option key={s.serviceId} value={s.serviceId}>{s.serviceName}</option>
                                        ))}
                                    </Form.Select>
                                )}
                            </Form.Group>
                        )}

                        {selectedService && (
                            <Form.Group className="mb-4">
                                <StepHeader n={step()} title="Select Date" />
                                <BookingDatePicker
                                    businessId={businessId}
                                    serviceId={serviceId}
                                    // Only meaningful once locked — before that,
                                    // no option has been chosen yet (that's
                                    // still step 3), so the calendar reflects
                                    // the whole service instead.
                                    serviceOptionId={locked ? serviceOptionId : undefined}
                                    value={date}
                                    onChange={handleDateChange}
                                />
                            </Form.Group>
                        )}

                        {selectedService && !locked && (
                            <Form.Group className="mb-4">
                                <StepHeader n={step()} title="Select Service Option" />
                                {checkingOptions ? (
                                    <div className="text-center py-3"><Spinner size="sm" /> Checking availability…</div>
                                ) : bookableOptions.length === 0 ? (
                                    <p className="text-muted mb-0">No options with available slots on this date.</p>
                                ) : (
                                    <div className="d-flex flex-column gap-2">
                                        {bookableOptions.map(opt => (
                                            <Form.Check
                                                key={opt.serviceOptionId}
                                                type="radio"
                                                name="serviceOption"
                                                id={`service-option-${opt.serviceOptionId}`}
                                                label={optionLabel(opt)}
                                                checked={serviceOptionId === opt.serviceOptionId}
                                                onChange={() => handleSelectServiceOption(opt.serviceOptionId)}
                                                className="bk-option-choice"
                                            />
                                        ))}
                                    </div>
                                )}
                            </Form.Group>
                        )}

                        {serviceOptionId && (
                            <>
                                {staff.length > 0 && (
                                    <Form.Group className="mb-4">
                                        <StepHeader n={step()} title="Select Staff" hint="optional" />
                                        <Form.Select value={staffId} onChange={e => handleStaffChange(e.target.value)}>
                                            <option value="">All staff</option>
                                            <option value={NONE_COL}>Owner-managed</option>
                                            {staff.map(s => (
                                                <option key={s.staffId} value={s.staffId}>{s.name}</option>
                                            ))}
                                        </Form.Select>
                                    </Form.Group>
                                )}

                                <StepHeader n={step()} title="Select Time Slot(s)" hint="pick slots across several dates" />
                                {checkingOptions ? (
                                    <div className="text-center py-3"><Spinner size="sm" /> Loading slots…</div>
                                ) : slots.length === 0 ? (
                                    <p className="text-muted">No slots available for this date.</p>
                                ) : (
                                    <div className="bk-slot-grid">
                                        {slots.map(slot => {
                                            const picked = isSelected(slot.serviceSlotId);
                                            return (
                                                <button
                                                    key={slot.serviceSlotId}
                                                    type="button"
                                                    className={`bk-slot-chip${picked ? " is-picked" : ""}`}
                                                    onClick={() => toggleSlot(slot)}
                                                >
                                                    <span className="bk-slot-time">
                                                        <IconClock size={13} />
                                                        {extractTime(slot.startTime)} – {extractTime(slot.endTime)}
                                                        {picked && <IconCheck size={13} className="ms-auto" />}
                                                    </span>
                                                    <span className="bk-slot-staff">
                                                        <IconUser size={11} className="me-1" />
                                                        {slot.staff?.name || "Owner-managed"}
                                                    </span>
                                                </button>
                                            );
                                        })}
                                    </div>
                                )}

                                {selected.length > 0 && (
                                    <div className="mt-3">
                                        <div className="fw-semibold mb-1">Selected ({selected.length})</div>
                                        <div className="d-flex flex-wrap gap-2">
                                            {selected.map(s => (
                                                <Badge
                                                    key={s.serviceSlotId}
                                                    className="bk-selected-chip d-inline-flex align-items-center gap-1"
                                                >
                                                    {fmtDate(s.date)} · {extractTime(s.startTime)}
                                                    <span
                                                        role="button"
                                                        aria-label="Remove"
                                                        className="bk-remove"
                                                        onClick={() => removeSelected(s.serviceSlotId)}
                                                        style={{ cursor: "pointer" }}
                                                    >
                                                        ✕
                                                    </span>
                                                </Badge>
                                            ))}
                                        </div>
                                    </div>
                                )}

                                <Form.Group className="mt-4">
                                    <Form.Label className="fw-semibold mb-1">
                                        Note for the business <span className="text-muted fw-normal">(optional)</span>
                                    </Form.Label>
                                    <Form.Control
                                        as="textarea"
                                        rows={2}
                                        value={description}
                                        onChange={e => setDescription(e.target.value)}
                                    />
                                </Form.Group>
                            </>
                        )}

                        {formError && <Alert variant="danger" className="mb-0 mt-3">{formError}</Alert>}
                    </>
                )}
            </Modal.Body>
            <Modal.Footer>
                <Button variant="outline-secondary" onClick={onHide}>
                    {allBooked ? "Close" : "Cancel"}
                </Button>
                {!allBooked && (
                    <Button variant="primary" onClick={handleBook} disabled={isSubmitting || selected.length === 0}>
                        {isSubmitting ? "Booking…" : `Book${selected.length > 1 ? ` (${selected.length})` : ""}`}
                    </Button>
                )}
            </Modal.Footer>
        </Modal>
    );
}
