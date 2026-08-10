import { Navigate, Outlet, useLocation } from "react-router-dom";
import {Spinner} from "react-bootstrap";
import { useAuth } from "../auth/AuthContext";

type ProtectedRouteProps = {
  roles?: string[];
  excludedRoles?: string[];
};

export default function ProtectedRoute({ roles = [], excludedRoles = [] }: ProtectedRouteProps) {
  const { isLoggedIn, user, hasRoles } = useAuth();
  const location = useLocation();

  if (isLoggedIn === null) {
    return (<Spinner animation="border" />);
  }

  if (isLoggedIn === false || !user ){
    return <Navigate to="/login" replace />;
  }

  if (user.staffProfile && user.mustResetPassword && location.pathname !== "/reset-password") {
    return <Navigate to="/reset-password" replace />;
  }

  if ((roles.length > 0 && !hasRoles(roles))) {
    return <Navigate to="/" replace />;
  }

  if (excludedRoles.length > 0 && hasRoles(excludedRoles)) {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
}
