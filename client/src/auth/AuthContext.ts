import { createContext, useContext } from "react";
import type { WorkingHour } from "../services/StaffService";

export type User = {
  userId: number;
  username: string;
  email: string;
  contactNumber?: string;
  mustResetPassword?: boolean;
  businessProfile?: BusinessProfile;
  staffProfile?: Staff;
};

export type BusinessProfile = {
  businessId: number;
}

export type Staff = {
  staffId: number;
  // Which business this staff member works for — shown on their dashboard,
  // since a staff account has no business profile of its own to read it from.
  businessName?: string;
  workingHours?: WorkingHour[];
}

export type AuthPayload = {
  token: string;
  user: User;
}

export type AuthContextValue = {
  isLoggedIn: boolean;
  user: User | null;
  token: string | null;
  message: string;
  // True when a previous session ended on its own (the token expired, or the
  // server rejected it) rather than by the user signing out. Distinguishes a
  // returning user who needs to re-authenticate — sent to the login page —
  // from a first-time visitor or someone who deliberately signed out, who
  // both land on the home page instead.
  sessionExpired: boolean;
  // Consumes the flag above. The lapsed-session bounce is a one-time handoff
  // ("you were signed in — sign in again"), not a durable state, so whoever
  // acts on it clears it. Without this the flag outlives its purpose and
  // every later visit to "/" is redirected to the login page forever.
  acknowledgeSessionExpired: () => void;
  login: (token: string, user: User) => void;
  logout: () => void;
  hasRoles: (roles: string[]) => boolean;
};

export const AuthContext = createContext<AuthContextValue | null>(null);

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);

  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }

  return context;
}
