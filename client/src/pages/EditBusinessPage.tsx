import React, { useState, useEffect } from "react";
import { Alert, Button, Container, Form, Spinner, Row, Col } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import saveIcon from "../assets/save.jpg";
import { useAuth } from "../auth/AuthContext";
import { userProfile } from "../services/ProfileService";
import { updateBusinessProfile, type BusinessProfileInput, type WorkingHour } from "../services/BusinessService";
import { applyGraphQLErrors } from "../utils/graphqlErrors";
import { FIELD_LIMITS, shouldClearEmailError, shouldClearContactNumberError } from "../utils/fieldLimits";
import { DAYS_OF_WEEK, startTimeSlice, endTimeSlice, toTimeInputValue} from "../utils/time";

export default function EditBusinessPage() {
    const navigate = useNavigate();
    const { token } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [businessName, setBusinessName] = useState("");
    const [description, setDescription] = useState("");
    const [address, setAddress] = useState("");
    const [imageUrl, setImageUrl] = useState("");
    const [businessContactNumber, setBusinessContactNumber] = useState("");
    const [businessEmail, setBusinessEmail] = useState("");
    const [workingHours, setWorkingHours] = useState<WorkingHour[]>([]);

    const [isLoading, setIsLoading] = useState(true);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [formError, setFormError] = useState("");
    const [formSuccess, setFormSuccess] = useState("");
    const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

    useEffect(() => {
        if (!activeToken) return;

        const fetchBusinessData = async () => {
            try {
                const result = await userProfile(activeToken);
                if (result.data?.userProfile?.businessProfile) {
                    const bp = result.data.userProfile.businessProfile;
                    setBusinessName(bp.businessName);
                    setDescription(bp.description || "");
                    setAddress(bp.address || "");
                    setImageUrl(bp.imageUrl || "");
                    setBusinessContactNumber(bp.businessContactNumber || "");
                    setBusinessEmail(bp.businessEmail || "");
                    setWorkingHours(bp.workingHours.map((wh: any) => ({
                        day: wh.day.toLowerCase(),
                        startTime: wh.startTime.split("T")[1]?.slice(0, 8) ?? "09:00:00",
                        endTime: wh.endTime.split("T")[1]?.slice(0, 8) ?? "18:00:00"
                    })));
                } else {
                    navigate("/register-business");
                }
            } catch {
                setFormError("Failed to load business data.");
            } finally {
                setIsLoading(false);
            }
        };

        fetchBusinessData();
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
        setFieldErrors(prev => ({ ...prev, [`workingHours[${index}]`]: "" }));
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!activeToken) return;

        setIsSubmitting(true);
        setFormError("");
        setFormSuccess("");
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
            const result = await updateBusinessProfile(activeToken, input);
            if (applyGraphQLErrors(result, {
                setFieldErrors,
                setFormError,
                fallbackMessage: "Failed to update business profile",
            })) return;

            setFormSuccess("Business profile updated successfully!");
        } catch {
            setFormError("Something went wrong. Please try again.");
        } finally {
            setIsSubmitting(false);
        }
    };

    if (isLoading) {
        return (
            <Container className="d-flex justify-content-center align-items-center" style={{ minHeight: "50vh" }}>
                <Spinner animation="border" variant="primary" />
            </Container>
        );
    }

    return (
        <Container className="py-5">
            <div className="d-flex justify-content-between align-items-center mb-3">
                <Button variant="link" onClick={() => navigate(-1)}>
                    <span style={{ fontSize: "1.3rem" }}></span>
                </Button>
                <h1 className="mb-0">Edit Business Profile</h1>
                <Button variant="link" type="submit" form="edit-business-form" disabled={isSubmitting} className="p-0">
                    <img src={saveIcon} alt="Save" style={{ width: "55px", height: "55px", objectFit: "contain" }} />
                    <span style={{ fontSize: "1.3rem" }}>Save</span>
                </Button>
            </div>

            {formError && <Alert variant="danger">{formError}</Alert>}
            {formSuccess && <Alert variant="success">{formSuccess}</Alert>}

            <Form id="edit-business-form" onSubmit={handleSubmit}>
                <Form.Group className="mb-3">
                    <Form.Label>Business Name</Form.Label>
                    <Form.Control
                        type="text"
                        value={businessName}
                        onChange={(e) => {
                            setBusinessName(e.target.value);
                            setFieldErrors(prev => ({ ...prev, businessName: "" }));
                        }}
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
                        onChange={(e) => {
                            setAddress(e.target.value);
                            setFieldErrors(prev => ({ ...prev, address: "" }));
                        }}
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
                        onChange={(e) => {
                            setBusinessContactNumber(e.target.value)
                            if (shouldClearContactNumberError(fieldErrors.businessContactNumber, e.target.value))
                                setFieldErrors(prev => ({ ...prev, businessContactNumber: "" }))
                        }}
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
                        onChange={(e) => {
                            setBusinessEmail(e.target.value)
                            if (shouldClearEmailError(fieldErrors.businessEmail, e.target.value))
                                setFieldErrors(prev => ({ ...prev, businessEmail: "" }))
                        }}
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

            </Form>
        </Container>
    );
}
