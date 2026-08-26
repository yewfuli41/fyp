import { createContext, useContext } from "react";

// Which "hat" a signed-in user is currently wearing. An owner or staff member
// can put on the customer hat to make a booking of their own; a plain customer
// only ever has this one.
export type ViewMode = "business" | "staff" | "customer";

export type ViewModeContextValue = {
  viewMode: ViewMode;
  // Only the modes this user is actually entitled to — a plain customer gets
  // a single entry, which is what hides the switcher for them.
  availableModes: { value: ViewMode; label: string }[];
  switchMode: (mode: ViewMode) => void;
};

export const ViewModeContext = createContext<ViewModeContextValue | null>(null);

export function useViewMode(): ViewModeContextValue {
  const context = useContext(ViewModeContext);

  if (!context) {
    throw new Error("useViewMode must be used within ViewModeProvider");
  }

  return context;
}
