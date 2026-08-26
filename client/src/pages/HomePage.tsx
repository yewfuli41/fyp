import { Container, Col, Row } from "react-bootstrap";
import { Link, Navigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { useViewMode } from "../view/ViewModeContext";
import BookLandingPage from "./BookLandingPage";
import BusinessDashboardPage from "./BusinessDashboardPage";
import StaffDashboardPage from "./StaffDashboardPage";
import { IconBell, IconCalendar, IconChart, IconClock, IconHistory, IconUsers } from "../components/icons";
import "../styles/HomePage.css";

// Grounded in what the app actually does — no invented features (payments,
// SMS, etc. aren't implemented) so this stays honest marketing copy. Each
// feature also carries its own accent colour (booking-theme.css) so the
// grid reads as six distinct categories at a glance instead of one blue
// block — buttons/links stay on --bk-primary, so colour here is purely
// for telling the cards apart, not implying they're interactive.
const FEATURES = [
    {
        icon: IconCalendar,
        title: "Smart Scheduling",
        description: "Recurring or one-off slots with automatic conflict prevention — no double-bookings.",
        color: "var(--bk-primary)",
    },
    {
        icon: IconClock,
        title: "Walk-In Friendly",
        description: "Record a walk-in customer straight into the same calendar as your online bookings.",
        color: "var(--bk-coral)",
    },
    {
        icon: IconUsers,
        title: "Staff Scheduling",
        description: "Set each staff member's working hours and see the whole team's availability at a glance.",
        color: "var(--bk-violet)",
    },
    {
        icon: IconBell,
        title: "Booking Requests",
        description: "Accept, reject, or reschedule requests the moment they come in — nothing slips through.",
        color: "var(--bk-rose)",
    },
    {
        icon: IconChart,
        title: "Built-In Analytics",
        description: "Booking trends, cancellations, and staff utilization, tracked automatically over time.",
        color: "var(--bk-teal)",
    },
    {
        icon: IconHistory,
        title: "Full History",
        description: "Every booking and leave application kept on record, filterable whenever you need it.",
        color: "var(--bk-amber)",
    },
];

export default function HomePage() {
    const { isLoggedIn, user, message, sessionExpired } = useAuth();
    const { viewMode } = useViewMode();

    // A returning user whose session lapsed on its own (expired token, or a
    // 401 mid-use — see sessionExpired in AuthProvider) is sent straight to
    // the login screen to pick up where they left off. A first-time visitor,
    // or someone who deliberately signed out, gets the welcome screen below
    // instead — neither of them was in the middle of anything.
    if (!isLoggedIn && sessionExpired) {
        return <Navigate to="/login" replace />;
    }

    // Customer mode belongs to "/" itself rather than a separate /book URL:
    // a plain customer is always in it, and an owner or staff member who
    // switches to it is asking for the same booking screen. Signed-out
    // visitors are deliberately excluded — the booking landing has nothing
    // to show them, so they keep the marketing screen at the bottom.
    if (isLoggedIn && viewMode === "customer") {
        return <BookLandingPage />;
    }

    // A business owner's landing page is the analytics dashboard (UC-12);
    // a staff member's is their own quick-glance dashboard — anyone signed
    // out falls through to the marketing screen at the bottom.
    if (isLoggedIn && user?.businessProfile) {
        return <BusinessDashboardPage />;
    }
    if (isLoggedIn && user?.staffProfile) {
        return <StaffDashboardPage />;
    }

    if (isLoggedIn && user) {
        return (
            <Container className="py-5 text-center">
                <h1>Welcome</h1>
                <p className="mb-3">
                    Signed in as <strong>{user.username}</strong> ({user.email})
                </p>
                {message && <p className="text-muted">{message}</p>}
                <Link to="/businesses" className="btn btn-primary">
                    Browse Businesses
                </Link>
            </Container>
        );
    }

    return (
        <Container className="py-4">
            <div className="home-hero">
                <span className="home-eyebrow">ALL-IN-ONE BOOKING SYSTEM FOR LOCAL BUSINESSES</span>
                <h1>Find & book<br /><span className="accent">local services in seconds.</span></h1>
                <p className="home-hero-subtitle">
                    Browse businesses, explore available services and time slots, and make your booking in one place.
                </p>
                <div className="home-hero-actions">
                    <Link to="/signup" className="btn btn-primary">Sign up free</Link>
                    <Link to="/businesses" className="btn btn-outline-secondary">Browse Businesses</Link>
                </div>
                <p className="home-hero-login mb-0">
                    Already have an account? <Link to="/login">Log in</Link>
                </p>
            </div>

            <h2 className="home-section-title">Everything you need to manage bookings in one place</h2>
            <p className="home-section-subtitle">Manage schedules, staff, customers, and more with one simple system.</p>
            <Row className="g-3">
                {FEATURES.map(f => (
                    <Col key={f.title} xs={12} sm={6} lg={4}>
                        <div className="home-feature-card">
                            <div className="home-feature-icon" style={{ background: f.color }}><f.icon size={22} /></div>
                            <div className="home-feature-title">{f.title}</div>
                            <div className="home-feature-description">{f.description}</div>
                        </div>
                    </Col>
                ))}
            </Row>

            <div className="home-cta">
                <h2>Own a business? Manage your bookings with ease.</h2>
                <p>Free to get started — set up your business in minutes.</p>
                <Link to="/register-business" className="btn">Register Business</Link>
            </div>
        </Container>
    );
}
