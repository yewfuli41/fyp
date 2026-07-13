import { useState, useEffect } from "react";
import { Alert, Button, Container, Spinner } from "react-bootstrap";
import { useNavigate, useParams } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { getPublicServices, getPublicStaff, type PublicService, type PublicStaff } from "../services/PublicService";
import BookingModal from "../modals/BookingModal";

export default function BusinessServicesPage() {
    const { businessId } = useParams<{ businessId: string }>();
    const navigate = useNavigate();
    const { token } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [services, setServices] = useState<PublicService[]>([]);
    const [staff, setStaff] = useState<PublicStaff[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState("");

    const [showBooking, setShowBooking] = useState(false);
    const [selectedOptionId, setSelectedOptionId] = useState("");
    const [selectedOptionName, setSelectedOptionName] = useState("");

    useEffect(() => {
        if (!businessId) return;
        const load = async () => {
            setIsLoading(true);
            try {
                const [svcRes, staffRes] = await Promise.all([
                    getPublicServices(businessId),
                    getPublicStaff(businessId),
                ]);
                setServices(svcRes.data?.publicServices ?? []);
                setStaff(staffRes.data?.publicStaff ?? []);
            } catch {
                setError("Failed to load services.");
            } finally {
                setIsLoading(false);
            }
        };
        load();
    }, [businessId]);

    const openBooking = (optionId: string, optionName: string) => {
        setSelectedOptionId(optionId);
        setSelectedOptionName(optionName);
        setShowBooking(true);
    };

    return (
        <Container className="py-4">
            <Button variant="link" className="ps-0 mb-2" onClick={() => navigate("/businesses")}>
                ← Back
            </Button>
            <h2>Services</h2>
            <hr />
            {error && <Alert variant="danger">{error}</Alert>}
            {isLoading ? (
                <div className="text-center py-5"><Spinner /></div>
            ) : services.length === 0 ? (
                <p className="text-muted">No services available.</p>
            ) : (
                services.map(svc => (
                    <div key={svc.serviceId} className="mb-4">
                        <h4>{svc.serviceName}</h4>
                        {svc.description && <p className="text-muted">{svc.description}</p>}
                        <strong>Options offered:</strong>
                        <div className="mt-2">
                            {svc.serviceOptions.map(opt => (
                                <div
                                    key={opt.serviceOptionId}
                                    className="d-flex justify-content-between align-items-center border rounded px-3 py-2 mb-2"
                                >
                                    <span className="fw-semibold">{opt.serviceOptionName}</span>
                                    <Button
                                        size="sm"
                                        variant="primary"
                                        onClick={() => openBooking(opt.serviceOptionId, opt.serviceOptionName)}
                                    >
                                        Book
                                    </Button>
                                </div>
                            ))}
                        </div>
                        <hr />
                    </div>
                ))
            )}

            {businessId && (
                <BookingModal
                    show={showBooking}
                    onHide={() => setShowBooking(false)}
                    businessId={businessId}
                    serviceOptionId={selectedOptionId}
                    serviceOptionName={selectedOptionName}
                    staff={staff}
                    token={activeToken}
                />
            )}
        </Container>
    );
}
