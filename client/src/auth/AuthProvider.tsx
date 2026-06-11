import React, { useState, useEffect} from "react";
import { AuthContext, type AuthContextValue, type User } from "./AuthContext";

export function AuthProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  useEffect(() => {
  const handleUnauthorized = () => {
    sessionStorage.setItem("authMessage", "Session expired. Please log in again.");
    logout();
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
      let role = "CUSTOMER"
      if(user.businessProfile)
        role = "OWNER"
      else if(user.staffProfile)
        role = "STAFF"
      return requiredRoles.includes(role)
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
