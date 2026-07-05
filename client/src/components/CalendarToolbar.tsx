import type { Dispatch, RefObject, SetStateAction } from "react";
import { Button, Form } from "react-bootstrap";
import type { Staff } from "../services/StaffService";
import type { Service } from "../services/ServiceService";
import { NONE_COL } from "../utils/serviceSlotHelpers";
import calendarIcon from "../assets/calendar-icon-vector_942802-557.avif";

interface CalendarToolbarProps {
    onBack: () => void;
    onAddClick: () => void;
    dateLabel: string;
    isoDate: string;
    dateInputRef: RefObject<HTMLInputElement | null>;
    onPrevDate: () => void;
    onNextDate: () => void;
    onDateChange: (isoDate: string) => void;
    // Omit these three (e.g. a staff member viewing only their own calendar,
    // where there's nothing to filter by staff) to hide the Select Staff filter.
    staffFilter?: string;
    setStaffFilter?: Dispatch<SetStateAction<string>>;
    staff?: Staff[];
    serviceFilter: string;
    setServiceFilter: Dispatch<SetStateAction<string>>;
    services: Service[];
}

// Purely presentational: back/add buttons, date navigation + picker, and the
// staff/service filters. All state and handlers are owned by CalendarPage.
export default function CalendarToolbar({
    onBack, onAddClick, dateLabel, isoDate, dateInputRef, onPrevDate, onNextDate, onDateChange,
    staffFilter, setStaffFilter, staff, serviceFilter, setServiceFilter, services,
}: CalendarToolbarProps) {
    return (
        <>
            <Button variant="link" className="px-0 mb-2 text-decoration-none" onClick={onBack}>
                &larr; Back to Services
            </Button>

            <div className="d-flex flex-wrap justify-content-between align-items-center mb-4 gap-3">
                <h1 className="fs-2 fw-bold mb-0">Calendar View</h1>
                <Button variant="primary" onClick={onAddClick}>Add Service Slots</Button>
            </div>

            <div className="d-flex flex-wrap align-items-end gap-4 mb-4">
                <div className="d-flex align-items-center gap-2">
                    <Button variant="outline-secondary" size="sm" onClick={onPrevDate}>&lt;</Button>
                    <div className="fw-semibold calendar-date-label">{dateLabel}</div>
                    <Button variant="outline-secondary" size="sm" onClick={onNextDate}>&gt;</Button>
                    <Button
                        variant="outline-secondary"
                        size="sm"
                        aria-label="Pick a date"
                        onClick={() => dateInputRef.current?.showPicker?.()}
                    >
                        <img src={calendarIcon} alt="" className="calendar-date-picker-icon" />
                    </Button>
                    <Form.Control
                        ref={dateInputRef}
                        type="date"
                        className="calendar-date-picker-hidden"
                        value={isoDate}
                        onChange={e => {
                            if (!e.target.value) return;
                            onDateChange(e.target.value);
                        }}
                    />
                </div>
                {staffFilter !== undefined && setStaffFilter && staff && (
                    <div>
                        <Form.Label className="fw-semibold mb-1 small">Select Staff</Form.Label>
                        <Form.Select value={staffFilter} onChange={e => setStaffFilter(e.target.value)} className="calendar-filter-select">
                            <option value="">All Staffs</option>
                            <option value={NONE_COL}>Unassigned (owner-managed)</option>
                            {staff.map(s => <option key={s.staffId} value={s.staffId}>{s.name}</option>)}
                        </Form.Select>
                    </div>
                )}
                <div>
                    <Form.Label className="fw-semibold mb-1 small">Select Service</Form.Label>
                    <Form.Select value={serviceFilter} onChange={e => setServiceFilter(e.target.value)} className="calendar-filter-select">
                        <option value="">All Services</option>
                        {services.map(s => <option key={s.serviceId} value={s.serviceId}>{s.serviceName}</option>)}
                    </Form.Select>
                </div>
            </div>
        </>
    );
}
