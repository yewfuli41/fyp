import { Button, Form } from "react-bootstrap";
import type { Staff } from "../services/StaffService";

// Staff + single-date filter shared by every leave list on the page (approved,
// rejected, and both of their history modals). A leave matches the date when
// that day falls anywhere inside its [startDate, endDate] range, not just when
// it starts on it.
export function leaveMatchesFilters(
    leave: { staffId: string; startDate: string; endDate: string },
    staffId: string,
    date: string,
): boolean {
    if (staffId && leave.staffId !== staffId) return false;
    if (date && (leave.startDate > date || leave.endDate < date)) return false;
    return true;
}

interface Props {
    staffList: Staff[];
    staffId: string;
    onStaffChange: (staffId: string) => void;
    date: string;
    onDateChange: (date: string) => void;
    id?: string;
}

export default function LeaveFilterBar({
    staffList, staffId, onStaffChange, date, onDateChange, id,
}: Props) {
    return (
        <div id={id} className="d-flex flex-wrap gap-2 mb-3">
            <Form.Select
                aria-label="Staff"
                style={{ maxWidth: 220 }}
                value={staffId}
                onChange={e => onStaffChange(e.target.value)}
            >
                <option value="">All staff</option>
                {staffList.map(s => (
                    <option key={s.staffId} value={s.staffId}>{s.name}</option>
                ))}
            </Form.Select>
            <Form.Control
                type="date"
                aria-label="Date"
                style={{ maxWidth: 170 }}
                value={date}
                onChange={e => onDateChange(e.target.value)}
            />
            {(staffId || date) && (
                <Button
                    variant="outline-secondary"
                    onClick={() => { onStaffChange(""); onDateChange(""); }}
                >
                    Clear filters
                </Button>
            )}
        </div>
    );
}
