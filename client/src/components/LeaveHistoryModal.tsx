import { useState } from "react";
import { Modal } from "react-bootstrap";
import type { LeaveApplication } from "../services/LeaveService";
import type { Staff } from "../services/StaffService";
import LeaveFilterBar, { leaveMatchesFilters } from "./LeaveFilterBar";
import LeaveTable from "./LeaveTable";

interface Props {
    show: boolean;
    onHide: () => void;
    title: string;
    // Already narrowed to the past leaves of one status by the caller.
    leaves: LeaveApplication[];
    staffList: Staff[];
    showReason?: boolean;
    // Seeds the staff filter from whatever the main list was filtered to, so
    // opening History doesn't silently widen what the owner was looking at.
    initialStaffId?: string;
    emptyMessage: string;
    noMatchMessage: string;
}

export default function LeaveHistoryModal({
    show, onHide, title, leaves, staffList, showReason,
    initialStaffId = "", emptyMessage, noMatchMessage,
}: Props) {
    const [staffId, setStaffId] = useState(initialStaffId);
    const [date, setDate] = useState("");

    const filtered = leaves.filter(l => leaveMatchesFilters(l, staffId, date));

    return (
        <Modal
            show={show}
            onHide={onHide}
            size="lg"
            // Re-seed the staff filter each time it's opened rather than
            // keeping whatever was left from last time.
            onShow={() => { setStaffId(initialStaffId); setDate(""); }}
        >
            <Modal.Header closeButton><Modal.Title>{title}</Modal.Title></Modal.Header>
            <Modal.Body>
                <LeaveFilterBar
                    staffList={staffList}
                    staffId={staffId}
                    onStaffChange={setStaffId}
                    date={date}
                    onDateChange={setDate}
                />
                <LeaveTable
                    leaves={filtered}
                    showReason={showReason}
                    emptyMessage={leaves.length === 0 ? emptyMessage : noMatchMessage}
                />
            </Modal.Body>
        </Modal>
    );
}
