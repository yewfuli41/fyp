import { type BookingDetail } from "../services/BookingService";
import { extractTime } from "../services/ServiceSlotService";

// Spells out exactly which booking a destructive confirmation is about.
// The prompt sentence alone ("Reject the request from Mr Yew?") isn't enough
// when a customer has several requests in the same list — rejecting can't be
// undone, so the owner should be able to check the date, time and service
// before committing rather than trusting they clicked the right row.
//
// Deliberately read-only text, not a form: there is nothing here to edit, and
// a confirmation step that looks editable invites the owner to try.

// Full date including the year — unlike the request list's "Mon, 22 Jun",
// which can lean on surrounding context that a modal doesn't have.
const fullDate = (iso: string) => {
    const d = new Date(`${iso}T00:00:00`);
    return isNaN(d.getTime())
        ? iso
        : d.toLocaleDateString("en-GB", { day: "numeric", month: "short", year: "numeric" });
};

const FIELD_LABEL_WIDTH = "5.5rem";

export default function BookingSummary({ booking }: { booking: BookingDetail }) {
    const rows: [string, string][] = [
        ["Date", fullDate(booking.date)],
        ["Time", `${extractTime(booking.startTime)} – ${extractTime(booking.endTime)}`],
        ["Service", booking.serviceName],
        ["Option", booking.optionName],
        // Owner-managed slots have no staff assigned (service_slots.staff_id
        // is nullable), so this row would otherwise render an empty value.
        ...(booking.staffName ? ([["Staff", booking.staffName]] as [string, string][]) : []),
        ["Customer", booking.customerName],
    ];

    return (
        <dl className="mb-0 mt-3 small">
            {rows.map(([label, value]) => (
                <div key={label} className="d-flex gap-2 mb-1">
                    <dt
                        className="fw-semibold text-secondary flex-shrink-0 mb-0"
                        style={{ width: FIELD_LABEL_WIDTH }}
                    >
                        {label}
                    </dt>
                    <dd className="mb-0">{value}</dd>
                </div>
            ))}
        </dl>
    );
}
