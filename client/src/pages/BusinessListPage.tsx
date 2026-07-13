import { useState, useEffect } from "react";
import { Alert, Button, Col, Container, Form, Row, Spinner } from "react-bootstrap";
import { useNavigate, useSearchParams } from "react-router-dom";
import { getPublicBusinesses, getPublicServices, type PublicBusiness, type PublicService } from "../services/PublicService";
import "../styles/BookLandingPage.css";

export default function BusinessListPage() {
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const [all, setAll] = useState<PublicBusiness[]>([]);
    const [servicesByBusiness, setServicesByBusiness] = useState<Record<string, PublicService[]>>({});
    const [search, setSearch] = useState(searchParams.get("search") ?? "");
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState("");

    useEffect(() => {
        const load = async () => {
            try {
                const res = await getPublicBusinesses();
                const businesses = res.data?.publicBusinesses ?? [];
                setAll(businesses);

                const servicesEntries = await Promise.all(
                    businesses.map(async biz => {
                        try {
                            const svcRes = await getPublicServices(biz.businessId);
                            return [biz.businessId, svcRes.data?.publicServices ?? []] as const;
                        } catch {
                            return [biz.businessId, []] as const;
                        }
                    })
                );
                setServicesByBusiness(Object.fromEntries(servicesEntries));
            } catch {
                setError("Failed to load businesses.");
            } finally {
                setIsLoading(false);
            }
        };
        load();
    }, []);

    const filtered = search.trim()
        ? all.filter(b =>
            b.businessName.toLowerCase().includes(search.toLowerCase()) ||
            b.description?.toLowerCase().includes(search.toLowerCase()) ||
            b.address?.toLowerCase().includes(search.toLowerCase())
        )
        : all;

    const handleBook = (businessId: string) => {
        navigate(`/businesses/${businessId}/services`);
    };

    return (
        <Container className="py-4">
            <Row className="align-items-center mb-3">
                <Col>
                    <h2 className="mb-0">Businesses</h2>
                </Col>
                <Col xs="auto">
                    <Form.Control
                        placeholder="Search businesses…"
                        value={search}
                        onChange={e => setSearch(e.target.value)}
                        style={{ width: 400 }}
                    />
                </Col>
            </Row>
            <hr />
            {error && <Alert variant="danger">{error}</Alert>}
            {isLoading ? (
                <div className="text-center py-5"><Spinner /></div>
            ) : filtered.length === 0 ? (
                <p className="text-muted">No businesses found.</p>
            ) : (
                <Row className="g-3">
                    {filtered.map(biz => {
                        const services = servicesByBusiness[biz.businessId] ?? [];
                        return (
                            <Col key={biz.businessId} xs={12} md={6} lg={4}>
                                <div className="biz-card text-start">
                                    <div className="d-flex align-items-center gap-3 mb-3">
                                        {biz.imageUrl ? (
                                            <img className="biz-avatar" src={biz.imageUrl} alt={biz.businessName} />
                                        ) : (
                                            <div className="biz-avatar">{biz.businessName.charAt(0).toUpperCase()}</div>
                                        )}
                                        <div className="biz-name">{biz.businessName}</div>
                                    </div>
                                    {biz.description && <p className="text-muted small mb-3">{biz.description}</p>}
                                    <div className="d-flex flex-column gap-2 text-muted small mb-3 biz-contact">
                                        {biz.address && <span>📍 {biz.address}</span>}
                                        {biz.businessContactNumber && <span>📞 {biz.businessContactNumber}</span>}
                                        {biz.businessEmail && <span>✉ {biz.businessEmail}</span>}
                                    </div>
                                    <div className="biz-services mb-3">
                                        <div className="biz-services-label">Services</div>
                                        {services.length === 0 ? (
                                            <span className="text-muted small">No services listed.</span>
                                        ) : (
                                            <div className="d-flex flex-wrap gap-2">
                                                {services.map(svc => (
                                                    <span key={svc.serviceId} className="biz-service-pill">
                                                        {svc.serviceName}
                                                    </span>
                                                ))}
                                            </div>
                                        )}
                                    </div>
                                    <div className="mt-auto text-end">
                                        <Button variant="primary" size="sm" onClick={() => handleBook(biz.businessId)}>
                                            Book
                                        </Button>
                                    </div>
                                </div>
                            </Col>
                        );
                    })}
                </Row>
            )}
        </Container>
    );
}
