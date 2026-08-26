import { Alert, Button, Table } from "react-bootstrap";
import type { LeaveApplication } from "../services/LeaveService";
import { IconDownload } from "./icons";

export const leaveDateRange = (leave: LeaveApplication): string =>
    leave.startDate === leave.endDate ? leave.startDate : `${leave.startDate} – ${leave.endDate}`;

interface Props {
    leaves: LeaveApplication[];
    emptyMessage: string;
    // Rejected leaves carry the owner's rejection reason; approved ones have
    // nothing to show there, so the column is dropped rather than left blank.
    showReason?: boolean;
    // Approved leaves can still be reversed (rejecting an approved leave is
    // the intended undo — there's no separate cancel), so they get an action.
    onReject?: (leave: LeaveApplication) => void;
    busyLeaveId?: string | null;
}

export default function LeaveTable({
    leaves, emptyMessage, showReason, onReject, busyLeaveId,
}: Props) {
    if (leaves.length === 0) {
        return <Alert variant="info" className="mb-0">{emptyMessage}</Alert>;
    }
    return (
        <div style={{ maxHeight: 420, overflowY: "auto" }}>
            <Table bordered hover responsive className="align-middle mb-0">
                <thead>
                    <tr>
                        <th>Staff</th>
                        <th>Dates</th>
                        {showReason && <th>Reason</th>}
                        {onReject && <th style={{ width: 120 }}>Actions</th>}
                    </tr>
                </thead>
                <tbody>
                    {leaves.map(leave => (
                        <tr key={leave.leaveId}>
                            <td>{leave.staffName}</td>
                            <td>
                                <div>{leaveDateRange(leave)}</div>
                                {leave.fileUrl && (
                                    <a
                                        href={leave.fileUrl}
                                        download={`leave-${leave.leaveId}-attachment`}
                                        className="d-inline-flex align-items-center gap-1 small mt-1"
                                    >
                                        <IconDownload size={12} /> Download
                                    </a>
                                )}
                            </td>
                            {showReason && (
                                <td className="text-secondary">
                                    {leave.remark?.trim()
                                        ? leave.remark
                                        : <span className="fst-italic text-muted">No reason given</span>}
                                </td>
                            )}
                            {onReject && (
                                <td>
                                    <Button
                                        variant="outline-danger" size="sm"
                                        disabled={busyLeaveId === leave.leaveId}
                                        onClick={() => onReject(leave)}
                                    >
                                        Reject
                                    </Button>
                                </td>
                            )}
                        </tr>
                    ))}
                </tbody>
            </Table>
        </div>
    );
}
