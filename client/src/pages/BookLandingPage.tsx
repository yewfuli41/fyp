import { useState, useEffect, useMemo } from "react";
import { Button, Col, Container, Form, Row, Spinner } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import {
    getPublicBusinesses, getPublicServices, getRecentlyBookedBusinesses,
    type PublicBusiness, type PublicService,
} from "../services/PublicService";
import { findBookableOptionIds, isCurrentOption } from "../utils/serviceAvailability";
import { IconCalendar, IconChevronRight, IconClock, IconPin, IconSearch, IconUser } from "../components/icons";
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
                    <div className="biz-avatar">{biz.businessName.charAt(0).toUpperCase()}</div>
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
    const [servicesByBusiness, setServicesByBusiness] = useState<Record<string, PublicService[]>>({});
    // Which options are still actually bookable — same "current + has a real
    // upcoming slot" check the services-browsing page uses, so a suggestion
    // never advertises an option with nothing to book.
    const [bookableOptionIds, setBookableOptionIds] = useState<Set<string>>(new Set());
    const [recent, setRecent] = useState<PublicBusiness[]>([]);
    const [recentLoading, setRecentLoading] = useState(false);

    // Loaded once for the type-ahead suggestions (name, description, and
    // each business's currently bookable options).
    useEffect(() => {
        const load = async () => {
            try {
                const res = await getPublicBusinesses();
                const list = res.data?.publicBusinesses ?? [];
                setBusinesses(list);
                setBusinessesLoading(false);

                const servicesEntries = await Promise.all(
                    list.map(async biz => {
                        try {
                            const svcRes = await getPublicServices(biz.businessId);
                            return [biz.businessId, svcRes.data?.publicServices ?? []] as const;
                        } catch {
                            return [biz.businessId, []] as const;
                        }
                    })
                );
                setServicesByBusiness(Object.fromEntries(servicesEntries));

                const bookableEntries = await Promise.all(
                    servicesEntries.map(([businessId, services]) => {
                        const currentOptions = services.flatMap(s => s.serviceOptions.filter(isCurrentOption));
                        return findBookableOptionIds(businessId, currentOptions);
                    })
                );
                setBookableOptionIds(new Set(bookableEntries.flatMap(set => [...set])));
            } catch {
                setBusinesses([]);
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

    // Flattened, currently-bookable options for one business — capped for
    // the compact dropdown row rather than every option it offers.
    const bookableOptionsFor = (businessId: string) =>
        (servicesByBusiness[businessId] ?? [])
            .flatMap(s => s.serviceOptions)
            .filter(o => isCurrentOption(o) && bookableOptionIds.has(o.serviceOptionId));

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
                                {suggestions.map(b => {
                                    const options = bookableOptionsFor(b.businessId);
                                    return (
                                        <li
                                            key={b.businessId}
                                            // onMouseDown fires before the input's blur, so the click registers.
                                            onMouseDown={() => goToBusiness(b.businessId)}
                                        >
                                            <div className="book-suggestion-name">{b.businessName}</div>
                                            {b.description && (
                                                <div className="book-suggestion-desc">{b.description}</div>
                                            )}
                                            {options.length > 0 && (
                                                <div className="book-suggestion-options">
                                                    {options.slice(0, 4).map(o => (
                                                        <span key={o.serviceOptionId} className="book-suggestion-option-pill">
                                                            {o.serviceOptionName}
                                                        </span>
                                                    ))}
                                                    {options.length > 4 && (
                                                        <span className="text-muted small align-self-center">
                                                            +{options.length - 4} more
                                                        </span>
                                                    )}
                                                </div>
                                            )}
                                        </li>
                                    );
                                })}
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
