import React, { useState, useEffect } from "react";
import { Alert, Button, Container, Form, Row, Col } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { userProfile } from "../services/ProfileService";
import { registerBusinessProfile, type BusinessProfileInput, type WorkingHour } from "../services/BusinessService";
import { applyGraphQLErrors } from "../utils/graphqlErrors";
import { FIELD_LIMITS } from "../utils/fieldLimits";

const DAYS_OF_WEEK = [
    "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"
]

const TIME_INTERVAL = 30

const toTimeInputValue = (time: string): string => {
    if (!time) return "";
    const parts = time.split(":");
    if (parts.length < 2) return "";
    return `${parts[0].padStart(2, "0")}:${parts[1].padStart(2, "0")}`;
};

const generateTimeArray = (minInterval: number): string[] => {
    const array:string[] = []
    for (let i = 0; i <24 ; i++){
        const hour = i.toString().padStart(2, "0")
        for (let j = 0; j < (60/minInterval); j++){
            const minutes = (minInterval*j).toString().padStart(2, "0")
            array.push(hour+":"+minutes)
        }
    }
    return array;
}

const startTimeSlice = (endTime: string, timeArray: string[]): string[]=> {
    const index = timeArray.indexOf(endTime)
    if (index === -1)
        return timeArray
    return timeArray.slice(0, index)
}

const endTimeSlice = (startTime: string, timeArray: string[]): string[]=> {
    const index = timeArray.indexOf(startTime)
    if (index === -1)
        return timeArray
    return timeArray.slice(index+1)
}

export default function RegisterBusinessPage() {
    const navigate = useNavigate();
    const { token, login } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [businessName, setBusinessName] = useState("");
    const [description, setDescription] = useState("");
    const [address, setAddress] = useState("");
    const [imageUrl, setImageUrl] = useState("");
    const [businessContactNumber, setBusinessContactNumber] = useState("");
    const [businessEmail, setBusinessEmail] = useState("");
    const [workingHours, setWorkingHours] = useState<WorkingHour[]>([
        { day: "monday", startTime: "09:00:00", endTime: "18:00:00" }
    ]);

    const [isSubmitting, setIsSubmitting] = useState(false);
    const [formError, setFormError] = useState("");
    const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
    const timeArray = generateTimeArray(TIME_INTERVAL)
    useEffect(() => {
        if (!activeToken) return;

        const checkExistingBusiness = async () => {
            try {
                const result = await userProfile(activeToken);
                if (result.data?.userProfile?.businessProfile) {
                    navigate("/edit-business");
                }
            } catch (err) {
                console.error("Failed to check existing business:", err);
            }
        };

        checkExistingBusiness();
    }, [activeToken, navigate]);

    const handleImageChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (file) {
            const reader = new FileReader();
            reader.onloadend = () => {
                setImageUrl(reader.result as string);
            };
            reader.readAsDataURL(file);
        }
    };

    const handleAddWorkingHour = () => {
        setWorkingHours([...workingHours, { day: "monday", startTime: "09:00:00", endTime: "18:00:00" }]);
    };

    const handleRemoveWorkingHour = (index: number) => {
        setWorkingHours(workingHours.filter((_, i) => i !== index));
    };

    const handleWorkingHourChange = (index: number, field: keyof WorkingHour, value: string) => {
        const newWorkingHours = [...workingHours];
        newWorkingHours[index] = { ...newWorkingHours[index], [field]: value };
        setWorkingHours(newWorkingHours);
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!activeToken) return;

        setIsSubmitting(true);
        setFormError("");
        setFieldErrors({});

        const input: BusinessProfileInput = {
            businessName,
            description: description || undefined,
            address,
            imageUrl: imageUrl || undefined,
            businessContactNumber,
            businessEmail,
            workingHours
        };

        try {
            const result = await registerBusinessProfile(activeToken, input);
            if (applyGraphQLErrors(result, {
                setFieldErrors,
                setFormError,
                fallbackMessage: "Failed to register business profile",
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
            navigate("/profile");
        } catch {
            setFormError("Something went wrong. Please try again.");
        } finally {
            setIsSubmitting(false);
        }
    };

    return (
        <Container className="py-5">
            <h1>Register Business Profile</h1>
            {formError && <Alert variant="danger">{formError}</Alert>}

            <Form onSubmit={handleSubmit}>
                <Form.Group className="mb-3">
                    <Form.Label>Business Name</Form.Label>
                    <Form.Control
                        type="text"
                        value={businessName}
                        onChange={(e) => setBusinessName(e.target.value)}
                        maxLength={FIELD_LIMITS.businessName}
                        isInvalid={!!fieldErrors.businessName}
                    />
                    <Form.Control.Feedback type="invalid">{fieldErrors.businessName}</Form.Control.Feedback>
                </Form.Group>

                <Form.Group className="mb-3">
                    <Form.Label>Description</Form.Label>
                    <Form.Control
                        as="textarea"
                        rows={3}
                        value={description}
                        onChange={(e) => setDescription(e.target.value)}
                    />
                </Form.Group>

                <Form.Group className="mb-3">
                    <Form.Label>Address</Form.Label>
                    <Form.Control
                        type="text"
                        value={address}
                        onChange={(e) => setAddress(e.target.value)}
                        isInvalid={!!fieldErrors.address}
                    />
                    <Form.Control.Feedback type="invalid">{fieldErrors.address}</Form.Control.Feedback>
                </Form.Group>

                <Form.Group className="mb-3">
                    <Form.Label>Business Image</Form.Label>
                    <Form.Control
                        type="file"
                        accept="image/*"
                        onChange={handleImageChange}
                    />
                    {imageUrl && (
                        <div className="mt-2">
                            <img src={imageUrl} alt="Preview" style={{ maxWidth: "200px", maxHeight: "200px" }} />
                        </div>
                    )}
                </Form.Group>

                <Form.Group className="mb-3">
                    <Form.Label>Business Contact Number</Form.Label>
                    <Form.Control
                        type="text"
                        value={businessContactNumber}
                        onChange={(e) => setBusinessContactNumber(e.target.value)}
                        maxLength={FIELD_LIMITS.businessContactNumber}
                        isInvalid={!!fieldErrors.businessContactNumber}
                    />
                    <Form.Control.Feedback type="invalid">{fieldErrors.businessContactNumber}</Form.Control.Feedback>
                </Form.Group>

                <Form.Group className="mb-3">
                    <Form.Label>Business Email</Form.Label>
                    <Form.Control
                        type="email"
                        value={businessEmail}
                        onChange={(e) => setBusinessEmail(e.target.value)}
                        maxLength={FIELD_LIMITS.businessEmail}
                        isInvalid={!!fieldErrors.businessEmail}
                    />
                    <Form.Control.Feedback type="invalid">{fieldErrors.businessEmail}</Form.Control.Feedback>
                </Form.Group>

                <h3 className="mt-4">Working Hours</h3>
                {workingHours.map((wh, index) => (
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
                                    {startTimeSlice(toTimeInputValue(wh.endTime), timeArray).map(time => (
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
                                    {endTimeSlice(toTimeInputValue(wh.startTime), timeArray).map(time => (
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
                        {isSubmitting ? "Registering..." : "Register Business"}
                    </Button>
                </div>
            </Form>
        </Container>
    );
}
