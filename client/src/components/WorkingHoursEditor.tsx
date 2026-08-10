import React, { useEffect, type Dispatch, type SetStateAction } from "react";
import { Button, Col, Form, Row } from "react-bootstrap";
import { extractTime } from "../services/ServiceSlotService";
import type { WorkingHour } from "../services/StaffService";
import { DAYS_OF_WEEK, startTimeSlice, endTimeSlice, toTimeInputValue } from "../utils/time";

interface WorkingHoursEditorProps {
    // The business's own hours — every entry here bounds what can be picked
    // below (a day the business is closed on can't be chosen at all, and
    // start/end are clamped within that day's business hours).
    businessWorkingHours: WorkingHour[];
    workingHours: WorkingHour[];
    setWorkingHours: Dispatch<SetStateAction<WorkingHour[]>>;
    fieldErrors: Record<string, string>;
    setFieldErrors: Dispatch<SetStateAction<Record<string, string>>>;
}

// Shared by RegisterStaffPage (setting a new staff's initial hours) and
// StaffAvailabilityPage (editing an existing staff's hours) — a list of
// day/start/end rows, each bounded by the business's own hours for that day.
export default function WorkingHoursEditor({
    businessWorkingHours, workingHours, setWorkingHours, fieldErrors, setFieldErrors,
}: WorkingHoursEditorProps) {
    // Days the business doesn't work at all can't be picked for a staff
    // working hour — only offer the days it actually operates on.
    const availableDays = DAYS_OF_WEEK.filter(day => businessWorkingHours.some(wh => wh.day === day));

    // businessWorkingHours loads asynchronously after mount (RegisterStaffPage
    // renders the form immediately, before its background fetch resolves), so
    // a default row picked before it arrives may not actually be a day the
    // business operates on — once real data arrives, snap any such mismatched
    // row to the first valid day. A row whose day is ALREADY valid is left
    // completely untouched: it may be real, previously-saved data (editing an
    // existing staff's hours), and must not be silently overwritten with a
    // freshly computed default the moment this effect first runs.
    useEffect(() => {
        if (availableDays.length === 0) return;
        setWorkingHours(prev => prev.map(wh => {
            if (availableDays.includes(wh.day)) return wh;
            const day = availableDays[0];
            const biz = businessWorkingHours.find(b => b.day === day);
            if (!biz) return { ...wh, day };
            const bizStart = extractTime(biz.startTime);
            const bizEnd = extractTime(biz.endTime);
            const opts = { businessStartTime: bizStart, businessEndTime: bizEnd };
            const startOpts = startTimeSlice(bizEnd, opts);
            const newStart = startOpts[0] ?? bizStart;
            const endOpts = endTimeSlice(newStart, opts);
            return {
                day,
                startTime: newStart + ":00",
                endTime: (endOpts[endOpts.length - 1] ?? bizEnd) + ":00",
            };
        }));
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [businessWorkingHours]);

    const handleAddWorkingHour = () => {
        const day = availableDays[0] ?? "monday";
        const biz = businessWorkingHours.find(wh => wh.day === day);
        let startTime = "09:00:00", endTime = "18:00:00";
        if (biz) {
            const bizStart = extractTime(biz.startTime);
            const bizEnd = extractTime(biz.endTime);
            const opts = { businessStartTime: bizStart, businessEndTime: bizEnd };
            const startOpts = startTimeSlice(bizEnd, opts);
            const newStart = startOpts[0] ?? bizStart;
            const endOpts = endTimeSlice(newStart, opts);
            startTime = newStart + ":00";
            endTime = (endOpts[endOpts.length - 1] ?? bizEnd) + ":00";
        }
        setWorkingHours([...workingHours, { day, startTime, endTime }]);
    };

    const handleRemoveWorkingHour = (index: number) => {
        setWorkingHours(workingHours.filter((_, i) => i !== index));
    };

    const handleWorkingHourChange = (index: number, field: keyof WorkingHour, value: string) => {
        const newWorkingHours = [...workingHours];
        if (field === "day") {
            const biz = businessWorkingHours.find(wh => wh.day === value);
            if (biz) {
                const bizStart = extractTime(biz.startTime);
                const bizEnd = extractTime(biz.endTime);
                const opts = { businessStartTime: bizStart, businessEndTime: bizEnd };
                const startOpts = startTimeSlice(bizEnd, opts);
                const newStart = startOpts[0] ?? bizStart;
                const endOpts = endTimeSlice(newStart, opts);
                const newEnd = endOpts[endOpts.length - 1] ?? bizEnd;
                newWorkingHours[index] = { day: value, startTime: newStart + ":00", endTime: newEnd + ":00" };
            } else {
                newWorkingHours[index] = { ...newWorkingHours[index], day: value };
            }
        } else {
            newWorkingHours[index] = { ...newWorkingHours[index], [field]: value };
        }
        setWorkingHours(newWorkingHours);
        setFieldErrors(prev => ({ ...prev, [`workingHours[${index}]`]: "" }));
    };

    // Staff can only work within the hours the business itself is open on
    // that day — startTimeSlice/endTimeSlice already accept a business-hours
    // window, it just wasn't being passed in before.
    const businessHoursForDay = (day: string) => businessWorkingHours.find(wh => wh.day === day);

    return (
        <>
            <h3 className="mt-4">Business Working Hours</h3>
            {businessWorkingHours.length === 0 ? (
                <p className="text-muted">No business working hours set.</p>
            ) : (
                <div className="mb-3">
                    {DAYS_OF_WEEK.filter(day => businessWorkingHours.some(wh => wh.day === day)).map(day => (
                        <div key={day}>
                            {day.charAt(0).toUpperCase() + day.slice(1)}: {businessWorkingHours
                                .filter(wh => wh.day === day)
                                .map(wh => `${extractTime(wh.startTime)} – ${extractTime(wh.endTime)}`)
                                .join(", ")}
                        </div>
                    ))}
                </div>
            )}

            <h3 className="mt-4">Working Hours</h3>
            {workingHours.map((wh, index) => {
                const dayBusinessHours = businessHoursForDay(wh.day);
                const timeOptions = dayBusinessHours && {
                    businessStartTime: extractTime(dayBusinessHours.startTime),
                    businessEndTime: extractTime(dayBusinessHours.endTime),
                };
                return (
                    <React.Fragment key={index}>
                        <Row className="mb-2 align-items-end">
                            <Col md={4}>
                                <Form.Group>
                                    <Form.Label>Day</Form.Label>
                                    <Form.Select
                                        value={wh.day}
                                        onChange={(e) => handleWorkingHourChange(index, "day", e.target.value)}
                                    >
                                        {availableDays.map(day => (
                                            <option key={day} value={day}>{day}</option>
                                        ))}
                                    </Form.Select>
                                </Form.Group>
                            </Col>
                            <Col md={3}>
                                <Form.Group>
                                    <Form.Label>Start Time</Form.Label>
                                    <Form.Select
                                        value={toTimeInputValue(wh.startTime)}
                                        onChange={(e) => handleWorkingHourChange(index, "startTime", e.target.value + ":00")}
                                        disabled={!dayBusinessHours}
                                    >
                                        {dayBusinessHours && startTimeSlice(toTimeInputValue(wh.endTime), timeOptions).map(time => (
                                            <option key={time} value={time}>{time}</option>
                                        ))}
                                    </Form.Select>
                                </Form.Group>
                            </Col>
                            <Col md={3}>
                                <Form.Group>
                                    <Form.Label>End Time</Form.Label>
                                    <Form.Select
                                        value={toTimeInputValue(wh.endTime)}
                                        onChange={(e) => handleWorkingHourChange(index, "endTime", e.target.value + ":00")}
                                        disabled={!dayBusinessHours}
                                    >
                                        {dayBusinessHours && endTimeSlice(toTimeInputValue(wh.startTime), timeOptions).map(time => (
                                            <option key={time} value={time}>{time}</option>
                                        ))}
                                    </Form.Select>
                                </Form.Group>
                            </Col>
                            <Col xs="auto">
                                <Button variant="danger" onClick={() => handleRemoveWorkingHour(index)}>Remove</Button>
                            </Col>
                        </Row>
                        {!dayBusinessHours && (
                            <div className="text-danger small mb-3">
                                The business is closed on {wh.day.charAt(0).toUpperCase() + wh.day.slice(1)} — choose a different day.
                            </div>
                        )}
                        {fieldErrors[`workingHours[${index}]`] && (
                            <div className="text-danger small mb-3">
                                {fieldErrors[`workingHours[${index}]`]}
                            </div>
                        )}
                    </React.Fragment>
                );
            })}
            {fieldErrors.workingHours && <div className="text-danger mb-2">{fieldErrors.workingHours}</div>}

            <Button variant="link" onClick={handleAddWorkingHour} className="mb-4">
                Add Working Hour
            </Button>
        </>
    );
}
