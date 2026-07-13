import { useState, useEffect } from "react";
import { Button, Col, Container, Form, Row, Spinner } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { getPublicBusinesses, getRecentlyBookedBusinesses, type PublicBusiness } from "../services/PublicService";
import "../styles/BookLandingPage.css";

function BizCard({ biz, onBook }: { biz: PublicBusiness; onBook: () => void }) {
    return (
        <div className="biz-card">
            <div className="d-flex align-items-start gap-3 mb-3">
                {biz.imageUrl ? (
                    <img className="biz-avatar" src={biz.imageUrl} alt={biz.businessName} />
                ) : (
                    <div className="biz-avatar">{biz.businessName.charAt(0).toUpperCase()}</div>
                )}
                <div className="flex-grow-1">
                    <div className="biz-name">{biz.businessName}</div>
                    {biz.address && <div className="biz-address">📍 {biz.address}</div>}
                </div>
            </div>
            <div className="mt-auto text-end">
                <button className="btn btn-link biz-book p-0" onClick={onBook}>
                    Book →
                </button>
            </div>
        </div>
    );
}

export default function BookLandingPage() {
    const navigate = useNavigate();
    const { isLoggedIn, token } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [search, setSearch] = useState("");
    const [focused, setFocused] = useState(false);
    const [businesses, setBusinesses] = useState<PublicBusiness[]>([]);
    const [recent, setRecent] = useState<PublicBusiness[]>([]);
    const [recentLoading, setRecentLoading] = useState(false);

    // Loaded once for the type-ahead suggestions.
    useEffect(() => {
        getPublicBusinesses()
            .then(res => setBusinesses(res.data?.publicBusinesses ?? []))
            .catch(() => setBusinesses([]));
    }, []);

    useEffect(() => {
        if (!isLoggedIn || !activeToken) {
            setRecent([]);
            navigate("/");
            return;
        }
        setRecentLoading(true);
        getRecentlyBookedBusinesses(activeToken)
            .then(res => setRecent(res.data?.recentlyBookedBusinesses ?? []))
            .catch(() => setRecent([]))
            .finally(() => setRecentLoading(false));
    }, [isLoggedIn, activeToken]);

    // Searching from the home page hands off to the full business list, filtered.
    const handleBook = (e: React.FormEvent) => {
        e.preventDefault();
        const q = search.trim();
        navigate(q ? `/businesses?search=${encodeURIComponent(q)}` : "/businesses");
    };

    const goToBusiness = (id: string) => navigate(`/businesses/${id}/services`);

    const q = search.trim().toLowerCase();
    const suggestions = q
        ? businesses.filter(b => b.businessName.toLowerCase().includes(q)).slice(0, 6)
        : [];
    const showSuggestions = focused && suggestions.length > 0;

    return (
        <Container className="py-4">
            <div className="book-hero">
                <h1>BookIt</h1>
                <p className="book-tagline">Book local services in seconds.</p>
                <Form className="book-search" onSubmit={handleBook}>
                    <div className="book-search-field">
                        <Form.Control
                            placeholder="Search Business"
                            value={search}
                            onChange={e => setSearch(e.target.value)}
                            onFocus={() => setFocused(true)}
                            onBlur={() => setFocused(false)}
                            autoComplete="off"
                        />
                        {showSuggestions && (
                            <ul className="book-autocomplete">
                                {suggestions.map(b => (
                                    <li
                                        key={b.businessId}
                                        // onMouseDown fires before the input's blur, so the click registers.
                                        onMouseDown={() => goToBusiness(b.businessId)}
                                    >
                                        {b.businessName}
                                    </li>
                                ))}
                            </ul>
                        )}
                    </div>
                    <Button variant="primary" type="submit">Search</Button>
                </Form>
            </div>

            {isLoggedIn && (
                <div className="book-results">
                    <div className="book-section-title">Recently booked</div>
                    {recentLoading ? (
                        <div className="text-center py-4"><Spinner /></div>
                    ) : recent.length === 0 ? (
                        <p className="text-muted mb-0">
                            You haven't booked any businesses yet.{" "}
                            <button className="btn btn-link p-0 align-baseline" onClick={() => navigate("/businesses")}>
                                Browse businesses →
                            </button>
                        </p>
                    ) : (
                        <Row className="g-3">
                            {recent.map(biz => (
                                <Col key={biz.businessId} xs={12} md={6} lg={4}>
                                    <BizCard biz={biz} onBook={() => goToBusiness(biz.businessId)} />
                                </Col>
                            ))}
                        </Row>
                    )}
                </div>
            )}
        </Container>
    );
}
