import { useState, useEffect } from "react";
import { Alert, Button, Col, Container, Form, Row, Spinner } from "react-bootstrap";
import { useNavigate, useSearchParams } from "react-router-dom";
import { getPublicBusinesses, type PublicBusiness } from "../services/PublicService";

export default function BusinessListPage() {
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const [all, setAll] = useState<PublicBusiness[]>([]);
    const [search, setSearch] = useState(searchParams.get("search") ?? "");
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState("");

    useEffect(() => {
        const load = async () => {
            try {
                const res = await getPublicBusinesses();
                setAll(res.data?.publicBusinesses ?? []);
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
                        style={{ width: 240 }}
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
                filtered.map(biz => (
                    <div key={biz.businessId} className="mb-4">
                        <Row className="align-items-start">
                            {biz.imageUrl && (
                                <Col xs="auto">
                                    <img
                                        src={biz.imageUrl}
                                        alt={biz.businessName}
                                        style={{ width: 90, height: 70, objectFit: "cover", borderRadius: 6 }}
                                    />
                                </Col>
                            )}
                            <Col>
                                <h5 className="mb-1">{biz.businessName}</h5>
                                {biz.description && <p className="text-muted mb-1 small">{biz.description}</p>}
                                <div className="d-flex gap-3 flex-wrap text-muted small">
                                    {biz.address && <span>📍 {biz.address}</span>}
                                    {biz.businessContactNumber && <span>📞 {biz.businessContactNumber}</span>}
                                    {biz.businessEmail && <span>✉ {biz.businessEmail}</span>}
                                </div>
                            </Col>
                            <Col xs="auto" className="d-flex align-items-center">
                                <Button
                                    variant="primary"
                                    size="sm"
                                    onClick={() => navigate(`/businesses/${biz.businessId}/services`)}
                                >
                                    Book
                                </Button>
                            </Col>
                        </Row>
                        <hr />
                    </div>
                ))
            )}
        </Container>
    );
}
