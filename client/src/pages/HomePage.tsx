import { Container } from "react-bootstrap";
import { Link } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";

export default function HomePage() {
    const { isLoggedIn, user, message } = useAuth();

    return (
        <Container className="py-5 text-center">
            <h1>Welcome</h1>
            {isLoggedIn && user ? (
                <>
                    <p className="mb-3">
                        Signed in as <strong>{user.username}</strong> ({user.email})
                    </p>
                    {message && <p className="text-muted">{message}</p>}
                    <Link to="/businesses" className="btn btn-primary">
                        Browse Businesses
                    </Link>
                </>
            ) : (
                <>
                    <p className="text-muted mb-4">Find and book services from local businesses.</p>
                    <div className="d-flex justify-content-center gap-2">
                        <Link to="/businesses" className="btn btn-primary">Browse Businesses</Link>
                        <Link to="/login" className="btn btn-outline-secondary">Log in</Link>
                    </div>
                </>
            )}
        </Container>
    );
}
