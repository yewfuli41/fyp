import type { RefObject } from "react";
import { Button, Form } from "react-bootstrap";
import type { Staff } from "../services/StaffService";
import type { Service } from "../services/ServiceService";
import { NONE_COL } from "../utils/serviceSlotHelpers";
import MultiSelectDropdown from "./MultiSelectDropdown";
import calendarIcon from "../assets/calendar-icon-vector_942802-557.avif";
import "../styles/toolbarButtons.css";

interface CalendarToolbarProps {
    onAddClick: () => void;
    onWalkInClick: () => void;
    dateLabel: string;
    isoDate: string;
    dateInputRef: RefObject<HTMLInputElement | null>;
    onPrevDate: () => void;
    onNextDate: () => void;
    onDateChange: (isoDate: string) => void;
    // Omit these three (e.g. a staff member viewing only their own calendar,
    // where there's nothing to filter by staff) to hide the Select Staff filter.
    staffFilters?: string[];
    setStaffFilters?: (ids: string[]) => void;
    staff?: Staff[];
    serviceFilters: string[];
    setServiceFilters: (ids: string[]) => void;
    services: Service[];
}

// Purely presentational: back/add buttons, date navigation + picker, and the
// staff/service filters. All state and handlers are owned by CalendarPage.
export default function CalendarToolbar({
    onAddClick, onWalkInClick, dateLabel, isoDate, dateInputRef, onPrevDate, onNextDate, onDateChange,
    staffFilters, setStaffFilters, staff, serviceFilters, setServiceFilters, services,
}: CalendarToolbarProps) {
    return (
        <>
            <div className="d-flex flex-wrap justify-content-between align-items-center mb-4 gap-3">
                <h1 className="fs-2 fw-bold mb-0">Calendar View</h1>
                <div className="d-flex gap-2">
                    <Button className="toolbar-btn toolbar-btn-soft" variant="outline-primary" onClick={onWalkInClick}>Record Walk-In</Button>
                    <Button className="toolbar-btn" variant="primary" onClick={onAddClick}>Add Service Slots</Button>
                </div>
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
                {staffFilters !== undefined && setStaffFilters && staff && (
                    <div>
                        <Form.Label className="fw-semibold mb-1 small d-block">Select Staff</Form.Label>
                        <MultiSelectDropdown
                            label="Staff"
                            selected={staffFilters}
                            onChange={setStaffFilters}
                            options={[
                                { value: NONE_COL, label: "Owner-managed" },
                                ...staff.map(s => ({ value: s.staffId, label: s.name })),
                            ]}
                        />
                    </div>
                )}
                <div>
                    <Form.Label className="fw-semibold mb-1 small d-block">Select Service(s)</Form.Label>
                    <MultiSelectDropdown
                        label="Service"
                        selected={serviceFilters}
                        onChange={setServiceFilters}
                        options={services.map(s => ({ value: s.serviceId, label: s.serviceName }))}
                    />
                </div>
            </div>
        </>
    );
}
