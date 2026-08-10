import { useState, useEffect } from "react";
import { Alert, Col, Container, Row, Spinner } from "react-bootstrap";
import { useNavigate, useParams } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import {
    getPublicBusinesses, getPublicServices,
    type PublicBusiness, type PublicService, type PublicServiceOption,
} from "../services/PublicService";
import { findBookableOptionIds, isCurrentOption } from "../utils/serviceAvailability";
import { IconCalendar, IconChevronLeft, IconPin } from "../components/icons";
import BookingModal from "../modals/BookingModal";
import "../styles/BusinessServicesPage.css";

function ServiceOptionCard({ option, onBook }: { option: PublicServiceOption; onBook: () => void }) {
    return (
        <div className="bsp-option-card">
            <div className="bsp-option-top">
                <div className="bsp-option-name">{option.serviceOptionName}</div>
            </div>
            {option.description && <div className="bsp-option-description">{option.description}</div>}
            {option.serviceOptionItems.length > 0 && (
                <div className="bsp-option-items">
                    {option.serviceOptionItems.map(item => (
                        <span key={item.serviceOptionItemId} className="bsp-item-pill">
                            {item.serviceOptionItemName}
                        </span>
                    ))}
                </div>
            )}
            <button type="button" className="btn text-white bsp-book-btn" onClick={onBook}>
                <IconCalendar size={15} /> Book
            </button>
        </div>
    );
}

export default function BusinessServicesPage() {
    const { businessId } = useParams<{ businessId: string }>();
    const navigate = useNavigate();
    const { token, isLoggedIn } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [business, setBusiness] = useState<PublicBusiness | null>(null);
    const [services, setServices] = useState<PublicService[]>([]);
    // Which currently-active options actually have a bookable slot coming up
    // — an option can be "current" (not expired) but fully booked out or not
    // yet slotted at all, in which case there's nothing to book.
    const [bookableOptionIds, setBookableOptionIds] = useState<Set<string>>(new Set());
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState("");

    const [showBooking, setShowBooking] = useState(false);
    const [bookingServiceId, setBookingServiceId] = useState("");
    const [bookingServiceOptionId, setBookingServiceOptionId] = useState("");

    useEffect(() => {
        if (!businessId) return;
        const load = async () => {
            setIsLoading(true);
            setError("");
            try {
                const [bizRes, svcRes] = await Promise.all([
                    getPublicBusinesses(),
                    getPublicServices(businessId),
                ]);
                const found = bizRes.data?.publicBusinesses.find(b => b.businessId === businessId) ?? null;
                setBusiness(found);
                const svcs = svcRes.data?.publicServices ?? [];
                setServices(svcs);

                // Cards for options with nothing actually bookable just
                // waste the customer's time — check every still-current
                // option for a real upcoming slot.
                const currentOptions = svcs.flatMap(s => s.serviceOptions.filter(isCurrentOption));
                setBookableOptionIds(await findBookableOptionIds(businessId, currentOptions));
            } catch {
                setError("Failed to load services.");
            } finally {
                setIsLoading(false);
            }
        };
        load();
    }, [businessId]);

    const openBooking = (serviceId: string, serviceOptionId: string) => {
        if (!isLoggedIn) { navigate("/login"); return; }
        setBookingServiceId(serviceId);
        setBookingServiceOptionId(serviceOptionId);
        setShowBooking(true);
    };

    if (!businessId) return null;

    return (
        <Container className="py-4">
            <button
                type="button"
                className="btn btn-link ps-0 mb-3 text-decoration-none bsp-back"
                onClick={() => navigate(-1)}
            >
                <IconChevronLeft size={16} /> Back
            </button>

            {isLoading ? (
                <div className="text-center py-5"><Spinner /></div>
            ) : (
                <>
                    {business && (
                        <div className="bsp-header">
                            {business.imageUrl ? (
                                <img className="bsp-avatar mx-auto" src={business.imageUrl} alt={business.businessName} />
                            ) : (
                                <div className="bsp-avatar mx-auto">{business.businessName.charAt(0).toUpperCase()}</div>
                            )}
                            <h1 className="h3 bsp-name">{business.businessName}</h1>
                            {business.address && (
                                <div className="bsp-address"><IconPin size={14} />{business.address}</div>
                            )}
                            {business.description && <p className="bsp-description">{business.description}</p>}
                        </div>
                    )}

                    {error && <Alert variant="danger">{error}</Alert>}

                    {(() => {
                        // Services left with nothing currently bookable are
                        // skipped entirely rather than shown with an empty
                        // options section.
                        const visibleServices = services
                            .map(svc => ({
                                svc,
                                options: svc.serviceOptions.filter(o => isCurrentOption(o) && bookableOptionIds.has(o.serviceOptionId)),
                            }))
                            .filter(({ options }) => options.length > 0);

                        return visibleServices.length === 0 ? (
                            <p className="bsp-empty">No services available right now.</p>
                        ) : (
                            <div className="text-start">
                                {visibleServices.map(({ svc, options }) => (
                                    <div key={svc.serviceId} className="bsp-service-block">
                                        <div className="bsp-service-name text-center">{svc.serviceName}</div>
                                        {svc.description && <p className="bsp-service-description text-center">{svc.description}</p>}
                                        <Row className="g-3">
                                            {options.map(opt => (
                                                <Col key={opt.serviceOptionId} xs={12} md={6} lg={4}>
                                                    <ServiceOptionCard
                                                        option={opt}
                                                        onBook={() => openBooking(svc.serviceId, opt.serviceOptionId)}
                                                    />
                                                </Col>
                                            ))}
                                        </Row>
                                    </div>
                                ))}
                            </div>
                        );
                    })()}
                </>
            )}

            <BookingModal
                show={showBooking}
                onHide={() => setShowBooking(false)}
                businessId={businessId}
                token={activeToken}
                initialServiceId={bookingServiceId}
                initialServiceOptionId={bookingServiceOptionId}
            />
        </Container>
    );
}
