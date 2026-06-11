import { Navigate, Outlet } from "react-router-dom";
import {Spinner} from "react-bootstrap";
import { useAuth } from "../auth/AuthContext";

type ProtectedRouteProps = {
  roles?: string[];
};

export default function ProtectedRoute({ roles = [] }: ProtectedRouteProps) {
  const { isLoggedIn, user, hasRoles } = useAuth();

  if (isLoggedIn === null) {
    return (<Spinner animation="border" />);
  }

  if (isLoggedIn === false || !user ){
    return <Navigate to="/login" replace />;
  }

  if ((roles.length > 0 && !hasRoles(roles))) {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
}