import { useState, useEffect, useCallback } from "react";
import { Badge, Container, Dropdown, Form, Nav, Navbar, Button } from "react-bootstrap";
import { NavLink, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { getBusinessBookings } from "../services/BookingService";
import { businessLeaveApplications, myLeaveApplications } from "../services/LeaveService";
import { onPendingCountsChanged } from "../utils/pendingCounts";
import { todayISO } from "../utils/serviceSlotHelpers";
import "../styles/AppNavbar.css";

type ViewMode = "business" | "staff" | "customer";

function getInitialMode(isOwner: boolean, isStaff: boolean): ViewMode {
    const saved = localStorage.getItem("viewMode") as ViewMode | null;
    if (saved === "business" && isOwner) return "business";
    if (saved === "staff" && isStaff) return "staff";
    if (saved === "customer") return "customer";
    if (isOwner) return "business";
    if (isStaff) return "staff";
    return "customer";
}

// Small badge overlay for a nav link that needs the owner/staff's attention
// — same pill-in-the-corner treatment ServicePage uses on its "Manage
// Service Slots" button, just generalized for any nav item.
function NavCountBadge({ count }: { count: number }) {
    if (count === 0) return null;
    return (
        <Badge
            bg="danger"
            pill
            className="position-absolute top-0 start-100 translate-middle"
            style={{ fontSize: "0.6rem" }}
            title={`${count} pending`}
        >
            {count}
        </Badge>
    );
}

export default function AppNavbar() {
    const { isLoggedIn, user, hasRoles, logout, token } = useAuth();
    const navigate = useNavigate();
    const isOwner = hasRoles(["OWNER"]);
    const isStaff = hasRoles(["STAFF"]);
    const activeToken = token ?? localStorage.getItem("token");

    const [viewMode, setViewMode] = useState<ViewMode>(() => getInitialMode(isOwner, isStaff));
    const [pendingBookingCount, setPendingBookingCount] = useState(0);
    const [pendingLeaveCount, setPendingLeaveCount] = useState(0);
    const [decidedMyLeaveCount, setDecidedMyLeaveCount] = useState(0);

    const switchMode = (mode: ViewMode) => {
        setViewMode(mode);
        localStorage.setItem("viewMode", mode);
        if (mode === "business" || mode === "staff") navigate("/");
        else navigate("/book");
    };

    const availableModes: { value: ViewMode; label: string }[] = [
        ...(isOwner ? [{ value: "business" as ViewMode, label: "Business Mode" }] : []),
        ...(isStaff ? [{ value: "staff" as ViewMode, label: "Staff Mode" }] : []),
        { value: "customer", label: "Customer Mode" },
    ];

    const showModeSwitcher = isLoggedIn && availableModes.length > 1;

    useEffect(() => {
    setViewMode(getInitialMode(isOwner, isStaff));
}, [isOwner, isStaff]);

    // Pending-count badges — bookings scope to the caller either way (all of
    // them for an owner, just their own for a staff member), so both modes
    // get a badge on their calendar link. Leave applications go the other
    // way: an owner sees applications still awaiting their own decision,
    // while a staff member sees their own applications that HAVE been
    // decided (decidedAt set) — they already know what's still pending
    // since they submitted it themselves; what's new is the owner's outcome.
    const refreshPendingCounts = useCallback(() => {
        if (!isLoggedIn || !activeToken || (!isOwner && !isStaff)) {
            setPendingBookingCount(0);
            setPendingLeaveCount(0);
            setDecidedMyLeaveCount(0);
            return;
        }
        getBusinessBookings(activeToken)
            .then(res => {
                const bookings = res.data?.businessBookings ?? [];
                setPendingBookingCount(bookings.filter(b => b.status === "PENDING").length);
            })
            .catch(() => setPendingBookingCount(0));

        if (isOwner) {
            businessLeaveApplications(activeToken)
                .then(res => {
                    const leaves = res.data?.businessLeaveApplications ?? [];
                    setPendingLeaveCount(leaves.filter(l => l.status === "PENDING").length);
                })
                .catch(() => setPendingLeaveCount(0));
        } else {
            setPendingLeaveCount(0);
        }

        if (isStaff) {
            myLeaveApplications(activeToken)
                .then(res => {
                    const leaves = res.data?.myLeaveApplications ?? [];
                    // Skip leave periods that are already over — nothing left
                    // to act on or care about there.
                    setDecidedMyLeaveCount(leaves.filter(l => l.decidedAt && l.endDate >= todayISO()).length);
                })
                .catch(() => setDecidedMyLeaveCount(0));
        } else {
            setDecidedMyLeaveCount(0);
        }
    }, [isLoggedIn, isOwner, isStaff, activeToken]);

    useEffect(() => {
        refreshPendingCounts();
    }, [refreshPendingCounts]);

    // Any page that settles a pending booking or leave (approve/reject a
    // leave, accept/reject/cancel/reschedule a booking) calls
    // notifyPendingCountsChanged() — refresh right away instead of waiting
    // for the next navigation to happen to re-run the effect above.
    useEffect(() => onPendingCountsChanged(refreshPendingCounts), [refreshPendingCounts]);

    return (
        <Navbar bg="white" expand="md" sticky="top" className="border-bottom shadow-sm">
            <Container>
                {/* Logo */}
                <Navbar.Brand
                    as={NavLink}
                    to={!isLoggedIn ? "/" : viewMode === "customer" ? "/book" : "/"}
                    className="fw-bold fs-4 px-3 py-1 border border-2 border-dark rounded text-dark me-4"
                    style={{ letterSpacing: "-0.5px" }}
                >
                    BookIt
                </Navbar.Brand>

                <Navbar.Toggle aria-controls="main-nav" />
                <Navbar.Collapse id="main-nav">
                    {/* Left nav links — role/mode dependent */}
                    <Nav variant="pills" className="me-auto gap-1">
                        {(!isLoggedIn || viewMode === "customer") && (
                            <Nav.Link as={NavLink} to="/businesses">Businesses</Nav.Link>
                        )}
                        {isLoggedIn && viewMode === "customer" && (
                            <Nav.Link as={NavLink} to="/appointments">My Appointments</Nav.Link>
                        )}
                        {viewMode === "business" && isOwner && (
                            <>
                                <Nav.Link as={NavLink} to="/">Dashboard</Nav.Link>
                                <Nav.Link as={NavLink} to="/services">Services</Nav.Link>
                                <span className="position-relative">
                                    <Nav.Link as={NavLink} to="/service-slots">Calendar</Nav.Link>
                                    <NavCountBadge count={pendingBookingCount} />
                                </span>
                                <Nav.Link as={NavLink} to="/staff">Staff</Nav.Link>
                                <span className="position-relative">
                                    <Nav.Link as={NavLink} to="/staff-availability">Staff Availability</Nav.Link>
                                    <NavCountBadge count={pendingLeaveCount} />
                                </span>
                            </>
                        )}
                        {viewMode === "staff" && isStaff && (
                            <>
                                <Nav.Link as={NavLink} to="/">Dashboard</Nav.Link>
                                <span className="position-relative">
                                    <Nav.Link as={NavLink} to="/my-calendar">My Calendar</Nav.Link>
                                    <NavCountBadge count={pendingBookingCount} />
                                </span>
                                <span className="position-relative">
                                    <Nav.Link as={NavLink} to="/leave">My Leave</Nav.Link>
                                    <NavCountBadge count={decidedMyLeaveCount} />
                                </span>
                            </>
                        )}
                    </Nav>

                    {/* Right side */}
                    <Nav className="align-items-center gap-2">
                        {/* Role switcher */}
                        {showModeSwitcher && (
                            <Form.Select
                                size="sm"
                                value={viewMode}
                                onChange={e => switchMode(e.target.value as ViewMode)}
                                style={{ width: "auto" }}
                            >
                                {availableModes.map(m => (
                                    <option key={m.value} value={m.value}>{m.label}</option>
                                ))}
                            </Form.Select>
                        )}

                        {/* Profile / auth */}
                        {isLoggedIn ? (
                            <Dropdown align="end">
                                <Dropdown.Toggle
                                    variant="link"
                                    className="text-decoration-none text-primary fw-semibold p-0 d-flex align-items-center gap-2"
                                >
                                    <span
                                        className="rounded border border-secondary d-inline-flex align-items-center justify-content-center bg-light overflow-hidden"
                                        style={{ width: 36, height: 36 }}
                                    >
                                        {user?.username?.[0]?.toUpperCase() ?? "?"}
                                    </span>
                                    {user?.username?.toUpperCase()}
                                </Dropdown.Toggle>
                                <Dropdown.Menu>
                                    <Dropdown.Item as={NavLink} to="/profile">Profile</Dropdown.Item>
                                    {isOwner ? (
                                        <Dropdown.Item as={NavLink} to="/edit-business">Business Profile</Dropdown.Item>
                                    ) : (
                                        <Dropdown.Item as={NavLink} to="/register-business">Register Business Profile</Dropdown.Item>
                                    )}
                                    <Dropdown.Divider />
                                    <Dropdown.Item onClick={logout} className="text-danger">Log out</Dropdown.Item>
                                </Dropdown.Menu>
                            </Dropdown>
                        ) : (
                            <>
                                <Nav.Link as={NavLink} to="/login">Log in</Nav.Link>
                                <NavLink to="/signup" className="text-decoration-none">
                                    <Button variant="primary">
                                        Sign up
                                    </Button>
                                </NavLink>
                            </>
                        )}
                    </Nav>
                </Navbar.Collapse>
            </Container>
        </Navbar>
    );
}
