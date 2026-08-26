import { doGraphQL } from "../api/graphql";
import { BOOKING_FIELDS, type BookingDetail } from "./BookingService";

export type LeaveStatus = "PENDING" | "APPROVED" | "REJECTED";

export interface LeaveApplication {
    leaveId: string;
    staffId: string;
    staffName: string;
    position?: string | null;
    startDate: string;
    endDate: string;
    justification?: string | null;
    fileUrl?: string | null;
    status: LeaveStatus;
    remark?: string | null;
    decidedAt?: string | null;
    createdAt: string;
    affectedBookings: BookingDetail[];
}

export interface ApplyLeaveInput {
    startDate: string; // "YYYY-MM-DD"
    endDate: string;   // "YYYY-MM-DD"
    justification?: string;
    fileUrl?: string; // base64 data URI of a supporting document
}

const LEAVE_FIELDS = `
    leaveId
    staffId
    staffName
    position
    startDate
    endDate
    justification
    fileUrl
    status
    remark
    decidedAt
    createdAt
    affectedBookings {
        ${BOOKING_FIELDS}
    }
`;

export const applyLeave = async (token: string, input: ApplyLeaveInput) => {
    const query = `
        mutation {
            applyLeave(input: {
                startDate: "${input.startDate}",
                endDate: "${input.endDate}",
                justification: ${input.justification ? JSON.stringify(input.justification) : "null"},
                fileUrl: ${input.fileUrl ? JSON.stringify(input.fileUrl) : "null"}
            }) {
                ${LEAVE_FIELDS}
            }
        }
    `;
    return await doGraphQL<{ applyLeave: LeaveApplication }>(query, token);
};

export const myLeaveApplications = async (token: string) => {
    const query = `
        query {
            myLeaveApplications {
                ${LEAVE_FIELDS}
            }
        }
    `;
    return await doGraphQL<{ myLeaveApplications: LeaveApplication[] }>(query, token);
};

// Only the reason can be edited, and only while the application is still
// pending — the date range is fixed once submitted.
export const updateLeaveApplication = async (token: string, leaveId: string, justification: string) => {
    const query = `
        mutation {
            updateLeaveApplication(leaveId: "${leaveId}", justification: ${JSON.stringify(justification)}) {
                ${LEAVE_FIELDS}
            }
        }
    `;
    return await doGraphQL<{ updateLeaveApplication: LeaveApplication }>(query, token);
};

export const deleteLeaveApplication = async (token: string, leaveId: string) => {
    const query = `
        mutation {
            deleteLeaveApplication(leaveId: "${leaveId}")
        }
    `;
    return await doGraphQL<{ deleteLeaveApplication: boolean }>(query, token);
};

export const businessLeaveApplications = async (token: string) => {
    const query = `
        query {
            businessLeaveApplications {
                ${LEAVE_FIELDS}
            }
        }
    `;
    return await doGraphQL<{ businessLeaveApplications: LeaveApplication[] }>(query, token);
};

// Approving never blocks on picking replacements — a slot with no booking is
// freed back to owner-managed immediately, while a booked slot is simply
// left as-is until the owner reschedules that booking separately (see
// rescheduleBooking in BookingService — it already requires the customer to
// accept the new time before it's final).
export interface LeaveReschedule {
    bookingId: string;
    newSlotOptionId: string;
}

// The owner settles every affected booking in the UI first and the whole set
// is sent here with the approval, so the server can move them in the same
// transaction. Nothing is written — and no customer is emailed — if the owner
// abandons the flow part-way, leaving the leave pending and every booking put.
export const approveLeaveApplication = async (
    token: string,
    leaveId: string,
    reschedules: LeaveReschedule[] = [],
) => {
    const reschedulesArg = reschedules.length
        ? `, reschedules: [${reschedules
            .map(r => `{bookingId: "${r.bookingId}", newSlotOptionId: "${r.newSlotOptionId}"}`)
            .join(", ")}]`
        : "";
    const query = `
        mutation {
            approveLeaveApplication(leaveId: "${leaveId}"${reschedulesArg}) {
                ${LEAVE_FIELDS}
            }
        }
    `;
    return await doGraphQL<{ approveLeaveApplication: LeaveApplication }>(query, token);
};

// Also used to reverse an already-approved leave (moving it to rejected,
// with a required remark) — there is no separate cancel mutation.
export const rejectLeaveApplication = async (token: string, leaveId: string, remark: string) => {
    const query = `
        mutation {
            rejectLeaveApplication(leaveId: "${leaveId}", remark: ${JSON.stringify(remark)}) {
                ${LEAVE_FIELDS}
            }
        }
    `;
    return await doGraphQL<{ rejectLeaveApplication: LeaveApplication }>(query, token);
};
