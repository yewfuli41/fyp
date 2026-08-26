import React, { useState, useEffect, useMemo } from "react";
import { useAuth } from "../auth/AuthContext";
import { ViewModeContext, type ViewMode } from "./ViewModeContext";

const VIEW_MODE_KEY = "viewMode";

// A stored mode is only honoured while the account still qualifies for it —
// a staff member who is removed, or an owner whose business is gone, would
// otherwise stay stuck on a set of nav links that lead nowhere.
function getInitialMode(isOwner: boolean, isStaff: boolean): ViewMode {
  const saved = localStorage.getItem(VIEW_MODE_KEY) as ViewMode | null;
  if (saved === "business" && isOwner) return "business";
  if (saved === "staff" && isStaff) return "staff";
  if (saved === "customer") return "customer";
  if (isOwner) return "business";
  if (isStaff) return "staff";
  return "customer";
}

// Lifted out of AppNavbar so the home page can answer the same question the
// navbar does: what should "/" show this user? The navbar owns the switcher,
// but it is no longer the only thing that needs the answer.
export function ViewModeProvider({ children }: { children: React.ReactNode }) {
  const { hasRoles } = useAuth();
  const isOwner = hasRoles(["OWNER"]);
  const isStaff = hasRoles(["STAFF"]);

  const [viewMode, setViewMode] = useState<ViewMode>(() => getInitialMode(isOwner, isStaff));

  // Signing in or out changes which modes exist, so the current one is
  // re-derived rather than left pointing at the previous user's role.
  useEffect(() => {
    setViewMode(getInitialMode(isOwner, isStaff));
  }, [isOwner, isStaff]);

  const switchMode = (mode: ViewMode) => {
    setViewMode(mode);
    localStorage.setItem(VIEW_MODE_KEY, mode);
  };

  const availableModes = useMemo(
    () => [
      ...(isOwner ? [{ value: "business" as ViewMode, label: "Business Mode" }] : []),
      ...(isStaff ? [{ value: "staff" as ViewMode, label: "Staff Mode" }] : []),
      { value: "customer" as ViewMode, label: "Customer Mode" },
    ],
    [isOwner, isStaff]
  );

  return (
    <ViewModeContext.Provider value={{ viewMode, availableModes, switchMode }}>
      {children}
    </ViewModeContext.Provider>
  );
}
