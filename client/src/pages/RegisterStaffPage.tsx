import React, { useState, useEffect } from "react";
import { Alert, Button, Container, Form, Row, Col } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { userProfile } from "../services/ProfileService";
import { registerStaff, type StaffInput, type WorkingHour } from "../services/StaffService";
import { extractTime } from "../services/ServiceSlotService";
import { applyGraphQLErrors } from "../utils/graphqlErrors";
import { FIELD_LIMITS, shouldClearEmailError, shouldClearContactNumberError } from "../utils/fieldLimits";
import { DAYS_OF_WEEK, startTimeSlice, endTimeSlice, toTimeInputValue} from "../utils/time";

export default function RegisterStaffPage() {
    const navigate = useNavigate();
    const { token, login } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [staffName, setStaffName] = useState("");
    const [staffEmail, setStaffEmail] = useState("");
    const [staffContactNumber, setStaffContactNumber] = useState("");
    const [position, setPosition] = useState("")
    const [businessWorkingHours, setBusinessWorkingHours] = useState<WorkingHour[]>([])
    const [staffWorkingHours, setStaffWorkingHours] = useState<WorkingHour[]>([
        { day: "monday", startTime: "09:00:00", endTime: "18:00:00" }
    ]);

    const [isSubmitting, setIsSubmitting] = useState(false);
    const [formError, setFormError] = useState("");
    const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
    const [successMessage, setSuccessMessage] = useState("");
    useEffect(() => {
        if (!activeToken) return;

        const checkExistingBusiness = async () => {
            try {
                const result = await userProfile(activeToken);
                if (result.data?.userProfile?.businessProfile) {
                    const businessWorkingHours = result.data.userProfile.businessProfile.workingHours
                    setBusinessWorkingHours(businessWorkingHours)
                }
            } catch (err) {
                console.error("Failed to check existing business:", err);
            }
        };

        checkExistingBusiness();
    }, [activeToken, navigate]);

    // Days the business doesn't work at all can't be picked for a staff
    // working hour — only offer the days it actually operates on.
    const availableDays = DAYS_OF_WEEK.filter(day => businessWorkingHours.some(wh => wh.day === day));

    // businessWorkingHours loads asynchronously after mount, so the "monday"
    // default above may not actually be a day the business operates on —
    // once real data arrives, snap any mismatched rows to the first valid day.
    useEffect(() => {
        if (availableDays.length === 0) return;
        setStaffWorkingHours(prev => prev.map(wh => {
            const day = availableDays.includes(wh.day) ? wh.day : availableDays[0];
            const biz = businessWorkingHours.find(b => b.day === day);
            if (!biz) return { ...wh, day };
            const bizStart = extractTime(biz.startTime);
            const bizEnd = extractTime(biz.endTime);
            const opts = { businessStartTime: bizStart, businessEndTime: bizEnd };
            const startOpts = startTimeSlice(bizEnd, opts);
            const newStart = startOpts[0] ?? bizStart;
            const endOpts = endTimeSlice(newStart, opts);
            return {
                day,
                startTime: newStart + ":00",
                endTime: (endOpts[endOpts.length - 1] ?? bizEnd) + ":00",
            };
        }));
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [businessWorkingHours]);

    const handleAddWorkingHour = () => {
        const day = availableDays[0] ?? "monday";
        const biz = businessWorkingHours.find(wh => wh.day === day);
        let startTime = "09:00:00", endTime = "18:00:00";
        if (biz) {
            const bizStart = extractTime(biz.startTime);
            const bizEnd = extractTime(biz.endTime);
            const opts = { businessStartTime: bizStart, businessEndTime: bizEnd };
            const startOpts = startTimeSlice(bizEnd, opts);
            const newStart = startOpts[0] ?? bizStart;
            const endOpts = endTimeSlice(newStart, opts);
            startTime = newStart + ":00";
            endTime = (endOpts[endOpts.length - 1] ?? bizEnd) + ":00";
        }
        setStaffWorkingHours([...staffWorkingHours, { day, startTime, endTime }]);
    };

    const handleRemoveWorkingHour = (index: number) => {
        setStaffWorkingHours(staffWorkingHours.filter((_, i) => i !== index));
    };

    const handleWorkingHourChange = (index: number, field: keyof WorkingHour, value: string) => {
        const newWorkingHours = [...staffWorkingHours];
        if (field === "day") {
            const biz = businessWorkingHours.find(wh => wh.day === value);
            if (biz) {
                const bizStart = extractTime(biz.startTime);
                const bizEnd = extractTime(biz.endTime);
                const opts = { businessStartTime: bizStart, businessEndTime: bizEnd };
                const startOpts = startTimeSlice(bizEnd, opts);
                const newStart = startOpts[0] ?? bizStart;
                const endOpts = endTimeSlice(newStart, opts);
                const newEnd = endOpts[endOpts.length - 1] ?? bizEnd;
                newWorkingHours[index] = { day: value, startTime: newStart + ":00", endTime: newEnd + ":00" };
            } else {
                newWorkingHours[index] = { ...newWorkingHours[index], day: value };
            }
        } else {
            newWorkingHours[index] = { ...newWorkingHours[index], [field]: value };
        }
        setStaffWorkingHours(newWorkingHours);
        setFieldErrors(prev => ({ ...prev, [`workingHours[${index}]`]: "" }));
    };

    // Staff can only work within the hours the business itself is open on
    // that day — startTimeSlice/endTimeSlice already accept a business-hours
    // window, it just wasn't being passed in before.
    const businessHoursForDay = (day: string) => businessWorkingHours.find(wh => wh.day === day);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!activeToken) return;

        setIsSubmitting(true);
        setFormError("");
        setFieldErrors({});

        const input: StaffInput = {
            name: staffName,
            email: staffEmail,
            contactNumber: staffContactNumber,
            position,
            workingHours: staffWorkingHours,
        };

        try {
            const result = await registerStaff(activeToken, input);
            if (applyGraphQLErrors(result, {
                setFieldErrors,
                setFormError,
                fallbackMessage: "Failed to register staff",
            })) return;

            const profileResult = await userProfile(activeToken);
            const profile = profileResult.data?.userProfile;

            if (profile) {
                login(activeToken, {
                    userId: Number(profile.userId),
                    username: profile.username,
                    email: profile.email,
                    contactNumber: profile.contactNumber,
                    businessProfile: profile.businessProfile,
                    staffProfile: profile.staffProfile,
                });
            }
            setSuccessMessage("✓ Staff registered successfully")
            setStaffName("");
            setStaffEmail("");
            setStaffContactNumber("");
            setStaffWorkingHours([{ day: "monday", startTime: "09:00:00", endTime: "18:00:00" }]);
            setPosition("");
        } catch {
            setFormError("Something went wrong. Please try again.");
        } finally {
            setIsSubmitting(false);
        }
    };

    return (
        <Container className="py-5">
            <h1>Register Staff</h1>
            {formError && <Alert variant="danger">{formError}</Alert>}
            {successMessage && <Alert variant="success">{successMessage}</Alert>}
            <Form onSubmit={handleSubmit}>
                <Form.Group className="mb-3">
                    <Form.Label>Name</Form.Label>
                    <Form.Control
                        type="text"
                        value={staffName}
                        onChange={(e) => {
                            setStaffName(e.target.value);
                            setFieldErrors(prev => ({ ...prev, staffName: "" }));
                        }}
                        maxLength={FIELD_LIMITS.staffName}
                        isInvalid={!!fieldErrors.staffName}
                    />
                    <Form.Control.Feedback type="invalid">{fieldErrors.staffName}</Form.Control.Feedback>
                </Form.Group>

                <Form.Group className="mb-3">
                    <Form.Label>Email</Form.Label>
                    <Form.Control
                        type="text"
                        value={staffEmail}
                        onChange={(e) => {
                            setStaffEmail(e.target.value)
                            if (shouldClearEmailError(fieldErrors.staffEmail, e.target.value))
                                setFieldErrors(prev => ({ ...prev, staffEmail: "" }))
                        }}
                        maxLength={FIELD_LIMITS.email}
                        isInvalid={!!fieldErrors.staffEmail}
                    />
                    <Form.Control.Feedback type="invalid">{fieldErrors.staffEmail}</Form.Control.Feedback>
                </Form.Group>

                <Form.Group className="mb-3">
                    <Form.Label>Contact Number</Form.Label>
                    <Form.Control
                        type="text"
                        value={staffContactNumber}
                        onChange={(e) => {
                            setStaffContactNumber(e.target.value)
                            if (shouldClearContactNumberError(fieldErrors.staffContactNumber, e.target.value))
                                setFieldErrors(prev => ({ ...prev, staffContactNumber: "" }))
                        }}
                        maxLength={FIELD_LIMITS.staffContactNumber}
                        isInvalid={!!fieldErrors.staffContactNumber}
                    />
                    <Form.Control.Feedback type="invalid">{fieldErrors.staffContactNumber}</Form.Control.Feedback>
                </Form.Group>

                <Form.Group className="mb-3">
                    <Form.Label>Position</Form.Label>
                    <Form.Control
                        type="text"
                        value={position}
                        onChange={(e) => {
                            setPosition(e.target.value);
                            setFieldErrors(prev => ({ ...prev, position: "" }));
                        }}
                        maxLength={FIELD_LIMITS.position}
                        isInvalid={!!fieldErrors.position}
                    />
                    <Form.Control.Feedback type="invalid">{fieldErrors.position}</Form.Control.Feedback>
                </Form.Group>

                <h3 className="mt-4">Business Working Hours</h3>
                {businessWorkingHours.length === 0 ? (
                    <p className="text-muted">No business working hours set.</p>
                ) : (
                    <div className="mb-3">
                        {DAYS_OF_WEEK.filter(day => businessWorkingHours.some(wh => wh.day === day)).map(day => (
                            <div key={day}>
                                {day.charAt(0).toUpperCase() + day.slice(1)}: {businessWorkingHours
                                    .filter(wh => wh.day === day)
                                    .map(wh => `${extractTime(wh.startTime)} – ${extractTime(wh.endTime)}`)
                                    .join(", ")}
                            </div>
                        ))}
                    </div>
                )}

                <h3 className="mt-4">Working Hours</h3>
                {staffWorkingHours.map((wh, index) => {
                    const dayBusinessHours = businessHoursForDay(wh.day);
                    const timeOptions = dayBusinessHours && {
                        businessStartTime: extractTime(dayBusinessHours.startTime),
                        businessEndTime: extractTime(dayBusinessHours.endTime),
                    };
                    return (
                    <React.Fragment key={index}>
                        <Row key={index} className="mb-2 align-items-end">
                            <Col md={4}>
                                <Form.Group>
                                    <Form.Label>Day</Form.Label>
                                    <Form.Select
                                        value={wh.day}
                                        onChange={(e) => handleWorkingHourChange(index, "day", e.target.value)}
                                    >
                                        {availableDays.map(day => (
                                            <option key={day} value={day}>{day}</option>
                                        ))}
                                    </Form.Select>
                                </Form.Group>
                            </Col>
                            <Col md={3}>
                                <Form.Group>
                                    <Form.Label>Start Time</Form.Label>
                                    <Form.Select
                                        value={toTimeInputValue(wh.startTime)}
                                        onChange={(e) => handleWorkingHourChange(index, "startTime", e.target.value + ":00")}
                                        disabled={!dayBusinessHours}
                                    >
                                        {dayBusinessHours && startTimeSlice(toTimeInputValue(wh.endTime), timeOptions).map(time => (
                                            <option key={time} value={time}>{time}</option>
                                        ))}
                                    </Form.Select>
                                </Form.Group>
                            </Col>
                            <Col md={3}>
                                <Form.Group>
                                    <Form.Label>End Time</Form.Label>
                                    <Form.Select
                                        value={toTimeInputValue(wh.endTime)}
                                        onChange={(e) => handleWorkingHourChange(index, "endTime", e.target.value + ":00")}
                                        disabled={!dayBusinessHours}
                                    >
                                        {dayBusinessHours && endTimeSlice(toTimeInputValue(wh.startTime), timeOptions).map(time => (
                                            <option key={time} value={time}>{time}</option>
                                        ))}
                                    </Form.Select>
                                </Form.Group>
                            </Col>
                            <Col xs="auto">
                                <Button variant="danger" onClick={() => handleRemoveWorkingHour(index)}>Remove</Button>
                            </Col>
                        </Row>
                        {!dayBusinessHours && (
                            <div className="text-danger small mb-3">
                                The business is closed on {wh.day.charAt(0).toUpperCase() + wh.day.slice(1)} — choose a different day.
                            </div>
                        )}
                        {fieldErrors[`workingHours[${index}]`] && (
                            <div className="text-danger small mb-3">
                                {fieldErrors[`workingHours[${index}]`]}
                            </div>
                        )}
                    </React.Fragment>
                    );
                })}
                {fieldErrors.workingHours && <div className="text-danger mb-2">{fieldErrors.workingHours}</div>}

                <Button variant="link" onClick={handleAddWorkingHour} className="mb-4">
                    Add Working Hour
                </Button>

                <div className="d-flex gap-2 justify-content-end">
                    <Button variant="outline-secondary" onClick={() => navigate("/profile")}>
                        Cancel
                    </Button>
                    <Button variant="primary" type="submit" disabled={isSubmitting}>
                        {isSubmitting ? "Registering..." : "Register Staff"}
                    </Button>
                </div>
            </Form>
        </Container>
    );
}
