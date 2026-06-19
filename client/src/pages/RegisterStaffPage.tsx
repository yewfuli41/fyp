import React, { useState, useEffect } from "react";
import { Alert, Button, Container, Form, Row, Col } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { userProfile } from "../services/ProfileService";
import { registerStaff, type StaffInput, type WorkingHour } from "../services/StaffService";
import { applyGraphQLErrors } from "../utils/graphqlErrors";
import { FIELD_LIMITS } from "../utils/fieldLimits";
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

    const handleAddWorkingHour = () => {
        setStaffWorkingHours([...staffWorkingHours, { day: "monday", startTime: "09:00:00", endTime: "18:00:00" }]);
    };

    const handleRemoveWorkingHour = (index: number) => {
        setStaffWorkingHours(staffWorkingHours.filter((_, i) => i !== index));
    };

    const handleWorkingHourChange = (index: number, field: keyof WorkingHour, value: string) => {
        const newWorkingHours = [...staffWorkingHours];
        newWorkingHours[index] = { ...newWorkingHours[index], [field]: value };
        setStaffWorkingHours(newWorkingHours);
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!activeToken) return;

        setIsSubmitting(true);
        setFormError("");
        setFieldErrors({});

        const input: StaffInput = {
            name: staffName,
            email: staffEmail,
            contactNumber: staffContactNumber || undefined,
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
                        onChange={(e) => setStaffName(e.target.value)}
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
                        onChange={(e) => setStaffEmail(e.target.value)}
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
                        onChange={(e) => setStaffContactNumber(e.target.value)}
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
                        onChange={(e) => setPosition(e.target.value)}
                        maxLength={FIELD_LIMITS.position}
                        isInvalid={!!fieldErrors.position}
                    />
                    <Form.Control.Feedback type="invalid">{fieldErrors.position}</Form.Control.Feedback>
                </Form.Group>

                <h3 className="mt-4">Working Hours</h3>
                {staffWorkingHours.map((wh, index) => (
                    <React.Fragment key={index}>
                        <Row key={index} className="mb-2 align-items-end">
                            <Col md={4}>
                                <Form.Group>
                                    <Form.Label>Day</Form.Label>
                                    <Form.Select
                                        value={wh.day}
                                        onChange={(e) => handleWorkingHourChange(index, "day", e.target.value)}
                                    >
                                        {DAYS_OF_WEEK.map(day => (
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
                                    >
                                        {startTimeSlice(toTimeInputValue(wh.endTime)).map(time => (
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
                                    >
                                        {endTimeSlice(toTimeInputValue(wh.startTime)).map(time => (
                                            <option key={time} value={time}>{time}</option>
                                        ))}
                                    </Form.Select>
                                </Form.Group>
                            </Col>
                            <Col xs="auto">
                                <Button variant="danger" onClick={() => handleRemoveWorkingHour(index)}>Remove</Button>
                            </Col>
                        </Row>
                        {fieldErrors[`workingHours[${index}]`] && (
                            <div className="text-danger small mb-3">
                                {fieldErrors[`workingHours[${index}]`]}
                            </div>
                        )}
                    </React.Fragment>
                ))}
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
