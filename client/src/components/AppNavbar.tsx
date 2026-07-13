import { useState } from "react";
import { Container, Dropdown, Form, Nav, Navbar, Button } from "react-bootstrap";
import { NavLink, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";

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

export default function AppNavbar() {
    const { isLoggedIn, user, hasRoles, logout } = useAuth();
    const navigate = useNavigate();
    const isOwner = hasRoles(["OWNER"]);
    const isStaff = hasRoles(["STAFF"]);

    const [viewMode, setViewMode] = useState<ViewMode>(() => getInitialMode(isOwner, isStaff));

    const switchMode = (mode: ViewMode) => {
        setViewMode(mode);
        localStorage.setItem("viewMode", mode);
        if (mode === "business") navigate("/services");
        else if (mode === "staff") navigate("/my-calendar");
        else navigate("/businesses");
    };

    const availableModes: { value: ViewMode; label: string }[] = [
        ...(isOwner ? [{ value: "business" as ViewMode, label: "Business Mode" }] : []),
        ...(isStaff ? [{ value: "staff" as ViewMode, label: "Staff Mode" }] : []),
        { value: "customer", label: "Customer Mode" },
    ];

    const showModeSwitcher = isLoggedIn && availableModes.length > 1;

    return (
        <Navbar bg="white" expand="md" sticky="top" className="border-bottom shadow-sm">
            <Container>
                {/* Logo */}
                <Navbar.Brand
                    as={NavLink}
                    to={isLoggedIn?"/book":"/"}
                    className="fw-bold fs-4 px-3 py-1 border border-2 border-dark rounded text-dark me-4"
                    style={{ letterSpacing: "-0.5px" }}
                >
                    BookIt
                </Navbar.Brand>

                <Navbar.Toggle aria-controls="main-nav" />
                <Navbar.Collapse id="main-nav">
                    {/* Left nav links — role/mode dependent */}
                    <Nav className="me-auto">
                        {(!isLoggedIn || viewMode === "customer") && (
                            <Nav.Link as={NavLink} to="/businesses">Businesses</Nav.Link>
                        )}
                        {viewMode === "business" && isOwner && (
                            <>
                                <Nav.Link as={NavLink} to="/">Dashboard</Nav.Link>
                                <Nav.Link as={NavLink} to="/services">Services</Nav.Link>
                                <Nav.Link as={NavLink} to="/staff">Staff</Nav.Link>
                            </>
                        )}
                        {viewMode === "staff" && isStaff && (
                            <>
                                <Nav.Link as={NavLink} to="/">Dashboard</Nav.Link>
                                <Nav.Link as={NavLink} to="/my-calendar">My Calendar</Nav.Link>
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

                        {/* Bell */}
                        {isLoggedIn && (
                            <Nav.Link className="px-1" style={{ fontSize: "1.25rem", lineHeight: 1 }}>
                                🔔
                            </Nav.Link>
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
