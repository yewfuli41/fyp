// Tiny pub-sub so the navbar's pending booking/leave badges can refresh the
// moment an owner or staff member settles one (approve/reject a leave,
// accept/reject/cancel/reschedule a booking) — those actions happen on
// sibling pages, not in AppNavbar itself, so there's no prop path between
// them; this is the simplest way to notify across that boundary without
// wiring a new context through the whole app.
type Listener = () => void;
const listeners = new Set<Listener>();

export const onPendingCountsChanged = (cb: Listener): (() => void) => {
    listeners.add(cb);
    return () => listeners.delete(cb);
};

// Call after any action that could change the caller's pending booking or
// leave counts so the navbar badges update immediately instead of waiting
// for the next full page load/navigation.
export const notifyPendingCountsChanged = (): void => {
    listeners.forEach(cb => cb());
};
