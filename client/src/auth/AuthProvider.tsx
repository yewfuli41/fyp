import React, { useState, useEffect} from "react";
import { useNavigate } from "react-router-dom";
import { AuthContext, type AuthContextValue, type User } from "./AuthContext";

export function AuthProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const navigate = useNavigate();

  useEffect(() => {
  const handleUnauthorized = () => {
    sessionStorage.setItem("authMessage", "Session expired. Please log in again.");
    logout();
    navigate("/login", { replace: true });
  };

  window.addEventListener("unauthorized", handleUnauthorized);

  return () => {
    window.removeEventListener("unauthorized", handleUnauthorized);
  };
}, []);
  const [user, setUser] = useState<User | null>(() => {
    const savedUser = localStorage.getItem("user");
    try {
      return savedUser ? JSON.parse(savedUser) : null;
    } catch {
      return null;
    }
  });
  const [token, setToken] = useState<string | null>(() => localStorage.getItem("token"));
  const [message, setMessage] = useState("");

  const login = (token: string, user: User) => {
    localStorage.setItem("token", token);
    localStorage.setItem("user", JSON.stringify(user));
    // Every fresh sign-in lands in Customer Mode (AppNavbar's getInitialMode
    // reads this) regardless of role or whatever mode was last active in a
    // previous session — a page reload of an already-open session doesn't
    // call login() at all, so this never resets the mode mid-session.
    localStorage.setItem("viewMode", "customer");

    setToken(token);
    setUser(user);
    setMessage("Logged in successfully!");
  };

  const logout = () => {
    localStorage.removeItem("token");
    localStorage.removeItem("user");

    setToken(null);
    setUser(null);
    setMessage("Logged out successfully!");
  };

  const hasRoles: AuthContextValue["hasRoles"] = 
    (requiredRoles) => {
      if (!user) return false;
      const roles: string[] = ["CUSTOMER"]
      if(user.businessProfile)
        roles.push("OWNER")
      if(user.staffProfile)
        roles.push("STAFF")
      return requiredRoles.some(role => roles.includes(role));
  };

  return (
    <AuthContext.Provider
      value={{
        isLoggedIn: !!token,
        user,
        token,
        message,
        login,
        logout,
        hasRoles,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
