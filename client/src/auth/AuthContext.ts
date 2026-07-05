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
