import { useState, useEffect, useMemo } from "react";
import { Button, Col, Container, Form, Row, Spinner } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import {
    getPublicBusinesses, getRecentlyBookedBusinesses,
    type PublicBusiness,
} from "../services/PublicService";
import { businessAvatarColor } from "../utils/businessColor";
import { IconCalendar, IconChevronRight, IconClock, IconMail, IconPhone, IconPin, IconSearch, IconUser } from "../components/icons";
import "../styles/BookLandingPage.css";

// A stable-per-visit sample of 3 — reshuffled only when the underlying list
// changes (i.e. once, after the initial load), not on every render.
function sampleThree<T>(items: T[]): T[] {
    if (items.length <= 3) return items;
    const pool = [...items];
    for (let i = pool.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [pool[i], pool[j]] = [pool[j], pool[i]];
    }
    return pool.slice(0, 3);
}

function BizCard({ biz, onBook }: { biz: PublicBusiness; onBook: () => void }) {
    return (
        <div className="biz-card">
            <div className="d-flex align-items-start gap-3 mb-3">
                {biz.imageUrl ? (
                    <img className="biz-avatar" src={biz.imageUrl} alt={biz.businessName} />
                ) : (
                    <div className="biz-avatar" style={{ background: businessAvatarColor(biz.businessId) }}>
                        {biz.businessName.charAt(0).toUpperCase()}
                    </div>
                )}
                <div className="flex-grow-1">
                    <div className="biz-name">{biz.businessName}</div>
                    {biz.address && (
                        <div className="biz-address"><IconPin size={13} />{biz.address}</div>
                    )}
                </div>
            </div>
            <div className="mt-auto text-end">
                <button className="btn btn-link biz-book p-0 d-inline-flex align-items-center gap-1" onClick={onBook}>
                    Book <IconChevronRight size={14} />
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
    const [businessesLoading, setBusinessesLoading] = useState(true);
    const [recent, setRecent] = useState<PublicBusiness[]>([]);
    const [recentLoading, setRecentLoading] = useState(false);

    // Loaded once for the type-ahead suggestions and the featured list.
    useEffect(() => {
        const load = async () => {
            try {
                const res = await getPublicBusinesses();
                setBusinesses(res.data?.publicBusinesses ?? []);
            } catch {
                setBusinesses([]);
            } finally {
                setBusinessesLoading(false);
            }
        };
        load();
    }, []);

    // Reshuffled only when the business list itself changes (i.e. once, right
    // after the initial load) — not a new random 3 on every re-render.
    const featured = useMemo(() => sampleThree(businesses), [businesses]);

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

    // Descriptions live on the services page, so every entry point routes
    // there instead of jumping straight into the booking modal.
    const goToBusiness = (id: string) => navigate(`/businesses/${id}/services`);

    const q = search.trim().toLowerCase();
    const suggestions = q
        ? businesses.filter(b => b.businessName.toLowerCase().includes(q)).slice(0, 6)
        : [];
    const showSuggestions = focused && suggestions.length > 0;

    return (
        <Container className="py-4">
            <div className="book-hero">
                <div className="book-hero-decor" aria-hidden="true">
                    <IconCalendar size={64} />
                    <IconClock size={48} />
                    <IconPin size={40} />
                    <IconUser size={52} />
                </div>
                <h1>BookIt</h1>
                <p className="book-tagline">Find local services and book with ease.</p>
                <Form className="book-search" onSubmit={handleBook}>
                    <div className="book-search-field">
                        <span className="book-search-icon"><IconSearch size={17} /></span>
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
                                        <div className="d-flex align-items-start gap-3">
                                            {b.imageUrl ? (
                                                <img className="book-suggestion-avatar" src={b.imageUrl} alt={b.businessName} />
                                            ) : (
                                                <div
                                                    className="book-suggestion-avatar"
                                                    style={{ background: businessAvatarColor(b.businessId) }}
                                                >
                                                    {b.businessName.charAt(0).toUpperCase()}
                                                </div>
                                            )}
                                            <div className="flex-grow-1" style={{ minWidth: 0 }}>
                                                <div className="book-suggestion-name">{b.businessName}</div>
                                                {b.address && (
                                                    <div className="book-suggestion-detail"><IconPin size={13} />{b.address}</div>
                                                )}
                                                <div className="book-suggestion-detail"><IconMail size={13} />{b.businessEmail}</div>
                                                <div className="book-suggestion-detail"><IconPhone size={13} />{b.businessContactNumber}</div>
                                            </div>
                                        </div>
                                    </li>
                                ))}
                            </ul>
                        )}
                    </div>
                    <Button variant="primary" type="submit">Search</Button>
                </Form>
            </div>

            {(businessesLoading || featured.length > 0) && (
                <div className="book-results">
                    <div className="book-section-title">Featured businesses</div>
                    {businessesLoading ? (
                        <div className="text-center py-4"><Spinner /></div>
                    ) : (
                        <Row className="g-3">
                            {featured.map(biz => (
                                <Col key={biz.businessId} xs={12} md={6} lg={4}>
                                    <BizCard biz={biz} onBook={() => goToBusiness(biz.businessId)} />
                                </Col>
                            ))}
                        </Row>
                    )}
                </div>
            )}

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
