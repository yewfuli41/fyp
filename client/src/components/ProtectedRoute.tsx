import { Navigate, Outlet } from "react-router-dom";
import {Spinner} from "react-bootstrap";
import { useAuth } from "../auth/AuthContext";

type ProtectedRouteProps = {
  role?: string;
};

export default function ProtectedRoute({ role = "" }: ProtectedRouteProps) {
  const { isLoggedIn, user, hasRoles } = useAuth();
console.log("user", user);
console.log(
  "businessProfile",
  user?.businessProfile
);
console.log(
  "has owner role",
  hasRoles("OWNER")
);
  if (isLoggedIn === null) {
    return (<Spinner animation="border" />);
  }

  if (isLoggedIn === false || !user || (!hasRoles(role))) {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
}