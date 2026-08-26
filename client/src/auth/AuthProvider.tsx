import React, { useState, useEffect} from "react";
import { AuthContext, type AuthContextValue, type User } from "./AuthContext";
import { isTokenExpired } from "../utils/token";

// Set when a session ends on its own — the stored token was already past its
// expiry, or the server rejected it mid-use. Persisted (rather than kept in
// React state alone) so the "you were signed in, please sign in again" landing
// survives a reload. Cleared by an explicit sign-out and by a fresh sign-in,
// which is what keeps a deliberate logout landing on the home page.
const SESSION_EXPIRED_KEY = "sessionExpired";

// A token left in localStorage past its expiry is no better than no token at
// all — clear it (and the user it belongs to) up front so the very first
// render already knows the session is over, rather than showing a signed-in
// UI until some request comes back 401.
const readStoredToken = (): string | null => {
  const token = localStorage.getItem("token");
  if (token && isTokenExpired(token)) {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    localStorage.setItem(SESSION_EXPIRED_KEY, "1");
    sessionStorage.setItem("authMessage", "Session expired. Please log in again.");
    return null;
  }
  return token;
};

export function AuthProvider({
  children,
}: {
  children: React.ReactNode;
}) {

  useEffect(() => {
  // A 401 mid-session is the same situation as finding an already-expired
  // token on load: the session lapsed rather than being ended deliberately,
  // so re-flag it after logout() (which clears the flag) has run.
  const handleUnauthorized = () => {
    sessionStorage.setItem("authMessage", "Session expired. Please log in again.");
    logout();
    localStorage.setItem(SESSION_EXPIRED_KEY, "1");
    setSessionExpired(true);
  };

  window.addEventListener("unauthorized", handleUnauthorized);

  return () => {
    window.removeEventListener("unauthorized", handleUnauthorized);
  };
}, []);
  // Declared before `user` on purpose: readStoredToken clears both entries
  // when the stored token has expired, so the user initializer below must
  // run after it to see that cleared state.
  const [token, setToken] = useState<string | null>(readStoredToken);
  const [user, setUser] = useState<User | null>(() => {
    const savedUser = localStorage.getItem("user");
    try {
      return savedUser ? JSON.parse(savedUser) : null;
    } catch {
      return null;
    }
  });
  // Read after readStoredToken, which is what may have just set the flag.
  const [sessionExpired, setSessionExpired] = useState(
    () => localStorage.getItem(SESSION_EXPIRED_KEY) === "1",
  );
  const [message, setMessage] = useState("");

  const login = (token: string, user: User) => {
    localStorage.setItem("token", token);
    localStorage.setItem("user", JSON.stringify(user));
    // Every fresh sign-in lands in Customer Mode (AppNavbar's getInitialMode
    // reads this) regardless of role or whatever mode was last active in a
    // previous session — a page reload of an already-open session doesn't
    // call login() at all, so this never resets the mode mid-session.
    localStorage.setItem("viewMode", "customer");
    localStorage.removeItem(SESSION_EXPIRED_KEY);

    setToken(token);
    setUser(user);
    setSessionExpired(false);
    setMessage("Logged in successfully!");
  };

  // Signing out is deliberate, so it clears the lapsed-session flag — the
  // user lands back on the home page rather than being pushed to log in.
  const logout = () => {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    localStorage.removeItem(SESSION_EXPIRED_KEY);

    setToken(null);
    setUser(null);
    setSessionExpired(false);
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
        sessionExpired,
        login,
        logout,
        hasRoles,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
