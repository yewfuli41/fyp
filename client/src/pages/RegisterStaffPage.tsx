import { useState, useEffect, type FormEvent } from "react";
import { Alert, Button, Container, Form } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { userProfile } from "../services/ProfileService";
import { registerStaff, type StaffInput, type WorkingHour } from "../services/StaffService";
import { applyGraphQLErrors } from "../utils/graphqlErrors";
import { FIELD_LIMITS, shouldClearEmailError, shouldClearContactNumberError, shouldClearPasswordError } from "../utils/fieldLimits";
import WorkingHoursEditor from "../components/WorkingHoursEditor";
import PasswordInput from "../components/PasswordInput";

export default function RegisterStaffPage() {
    const navigate = useNavigate();
    const { token, login } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [staffName, setStaffName] = useState("");
    const [staffEmail, setStaffEmail] = useState("");
    const [staffContactNumber, setStaffContactNumber] = useState("");
    const [position, setPosition] = useState("")
    const [staffPassword, setStaffPassword] = useState("");
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

    const handleSubmit = async (e: FormEvent) => {
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
            password: staffPassword,
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
            setStaffPassword("");
        } catch {
            setFormError("Something went wrong. Please try again.");
        } finally {
            setIsSubmitting(false);
        }
    };

    return (
        <Container className="py-5">
            <h1>Register Staff</h1>
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

                <PasswordInput
                    controlId="staffPassword"
                    label="Temporary Password"
                    value={staffPassword}
                    onChange={(value) => {
                        setStaffPassword(value);
                        if (shouldClearPasswordError(fieldErrors.password, value))
                            setFieldErrors(prev => ({ ...prev, password: "" }));
                    }}
                    isInvalid={!!fieldErrors.password}
                    errorMessage={fieldErrors.password}
                    className="mb-1"
                />
                <p className="text-muted small mb-3">
                    Share this with the staff member directly — they'll be required to change it on first login.
                </p>

                <WorkingHoursEditor
                    businessWorkingHours={businessWorkingHours}
                    workingHours={staffWorkingHours}
                    setWorkingHours={setStaffWorkingHours}
                    fieldErrors={fieldErrors}
                    setFieldErrors={setFieldErrors}
                />

                {formError && <Alert variant="danger">{formError}</Alert>}

                <div className="d-flex gap-2 justify-content-end">
                    <Button variant="outline-secondary" onClick={() => navigate("/staff")}>
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
