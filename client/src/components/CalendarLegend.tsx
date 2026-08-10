// Explains the status colours used on slot blocks in CalendarGrid — keep the
// four entries below in sync with the status classes defined in CalendarPage.css.
export default function CalendarLegend() {
    return (
        <div className="calendar-legend">
            <span className="calendar-legend-item">
                <i className="calendar-legend-dot calendar-slot-accepted" /> Accepted
            </span>
            <span className="calendar-legend-item">
                <i className="calendar-legend-dot calendar-slot-reschedule-pending" /> Rescheduled
            </span>
            <span className="calendar-legend-item">
                <i className="calendar-legend-dot calendar-slot-pending" /> Pending
            </span>
            <span className="calendar-legend-item">
                <i className="calendar-legend-dot calendar-slot-available" /> Available
            </span>
        </div>
    );
}
