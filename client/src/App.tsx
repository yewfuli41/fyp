import { Container } from "react-bootstrap";
import { Navigate, Route, Routes, Link } from "react-router-dom";
import { useAuth } from "./auth/AuthContext";
import LogInPage from "./pages/LogInPage";
import SignUpPage from "./pages/SignUpPage";
import ProfilePage from "./pages/ProfilePage";
import EditBusinessPage from "./pages/EditBusinessPage";
import RegisterBusinessPage from "./pages/RegisterBusinessPage";
import ServicePage from "./pages/ServicePage";
import RegisterStaffPage from "./pages/RegisterStaffPage"
import ProtectedRoute from "./components/ProtectedRoute";

function HomePage() {
  const { isLoggedIn, user, message, logout } = useAuth();

  return (
    <Container className="py-5 text-center">
      <h1>Welcome</h1>
      {isLoggedIn && user ? (
        <>
          <p className="mb-3">
            Signed in as <strong>{user.username}</strong> ({user.email})
          </p>
          {message && <p className="text-muted">{message}</p>}
          <div className="d-flex justify-content-center gap-2">
            <Link to="/profile" className="btn btn-primary">
              View Profile
            </Link>
            <button type="button" className="btn btn-outline-secondary" onClick={logout}>
              Log out
            </button>
          </div>
        </>
      ) : (
        <p className="text-muted">Please log in or sign up to continue.</p>
      )}
    </Container>
  );
}

function App() {
  return (
    <Routes>
      <Route path="/" element={<HomePage />} />
      <Route path="/login" element={<LogInPage />} />
      <Route path="/signup" element={<SignUpPage />} />
      <Route element={<ProtectedRoute />}>
        <Route path="/profile" element={<ProfilePage />} />
        <Route path="/register-business" element={<RegisterBusinessPage />} />
      </Route>
      <Route element={<ProtectedRoute roles={["OWNER"]} />}>
        <Route path="/edit-business" element={<EditBusinessPage />} />
        <Route path="/services" element={<ServicePage />} />
        <Route path="/register-staff" element={<RegisterStaffPage/>}/>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}

export default App;
