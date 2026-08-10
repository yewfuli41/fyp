import {doGraphQL} from "../api/graphql"
import { toTimeScalar, type ServiceSlot } from "./ServiceSlotService";

export interface WorkingHour {
    day: string;
    startTime: string;
    endTime: string;
}

export interface StaffInput{
    name: string;
    email: string;
    contactNumber: string;
    position: string;
    // Temporary password the owner sets for a brand-new staff account — the
    // staff member is required to change it on their first login.
    password: string;
    workingHours: WorkingHour[];
}

export interface Staff {
    staffId: string;
    name: string;
    email: string;
    mustResetPassword: boolean;
    contactNumber: string;
    position?: string;
    workingHours?: WorkingHour[];
    // true once this staff member has an active (pending/accepted/rescheduled)
    // booking on any of their slots — exactly what blocks deleting them. Only
    // populated by getBusinessStaff (used for the delete button); other Staff
    // call sites omit it.
    hasBooking?: boolean;
}

export interface UpdateStaffInput {
    name: string;
    email: string;
    contactNumber: string;
    position: string;
}

const STAFF_FIELDS = `
    staffId
    name
    email
    mustResetPassword
    contactNumber
    position
    hasBooking
    workingHours {
        day
        startTime
        endTime
    }
`;

export const registerStaff = async(token: string, staff:StaffInput) => {
    const query = `
        mutation {
            registerStaff(staff:{
                name: "${staff.name}",
                email: "${staff.email}",
                contactNumber: "${staff.contactNumber}",
                position: "${staff.position}",
                password: ${JSON.stringify(staff.password)},
                workingHours: [
                    ${staff.workingHours.map(wh => `{
                        day: ${wh.day.toLowerCase()},
                        startTime: "${toTimeScalar(wh.startTime)}",
                        endTime: "${toTimeScalar(wh.endTime)}"
                    }`).join(",")}
                ]
            })
        }
    `

    return await doGraphQL(query, token);
}

export const getBusinessStaff = async (token: string) => {
    const query = `
        query {
            displayStaff {
                ${STAFF_FIELDS}
            }
        }
    `;
    return await doGraphQL<{ displayStaff: Staff[] }>(query, token);
};

export const updateStaff = async (token: string, staffId: string, staff: UpdateStaffInput) => {
    const query = `
        mutation {
            updateStaff(staffId: "${staffId}", staff: {
                name: ${JSON.stringify(staff.name)},
                email: ${JSON.stringify(staff.email)},
                contactNumber: ${JSON.stringify(staff.contactNumber)},
                position: ${JSON.stringify(staff.position)}
            }) {
                ${STAFF_FIELDS}
            }
        }
    `;
    return await doGraphQL<{ updateStaff: Staff }>(query, token);
};

export const deleteStaff = async (token: string, staffId: string) => {
    const query = `
        mutation {
            deleteStaff(staffId: "${staffId}")
        }
    `;
    return await doGraphQL<{ deleteStaff: boolean }>(query, token);
};

// Picks a replacement staff (or unassigns, when staffId is "") for one
// service slot affected by a leave approval or a working-hours edit.
export interface SlotReassignment {
    serviceSlotId: string;
    staffId: string; // "" => unassign (owner-managed)
}

const buildWorkingHoursBody = (workingHours: WorkingHour[]): string =>
    `[${workingHours.map(wh => `{
        day: ${wh.day.toLowerCase()},
        startTime: "${toTimeScalar(wh.startTime)}",
        endTime: "${toTimeScalar(wh.endTime)}"
    }`).join(",")}]`;

export const buildReassignmentsBody = (reassignments: SlotReassignment[]): string =>
    `[${reassignments.map(r => `{
        serviceSlotId: "${r.serviceSlotId}",
        staffId: ${r.staffId ? `"${r.staffId}"` : "null"}
    }`).join(",")}]`;

// Precheck before updateStaffWorkingHours — the staff's currently-booked
// slots that would fall outside the proposed hours, so the caller can show a
// replacement-staff picker before committing the change.
export const staffHoursConflicts = async (token: string, staffId: string, workingHours: WorkingHour[]) => {
    const query = `
        query {
            staffHoursConflicts(staffId: "${staffId}", workingHours: ${buildWorkingHoursBody(workingHours)}) {
                serviceSlotId
                date
                startTime
                endTime
                staff { staffId name }
                hasBooking
            }
        }
    `;
    return await doGraphQL<{ staffHoursConflicts: ServiceSlot[] }>(query, token);
};

export const updateStaffWorkingHours = async (
    token: string, staffId: string, workingHours: WorkingHour[], reassignments: SlotReassignment[],
) => {
    const query = `
        mutation {
            updateStaffWorkingHours(
                staffId: "${staffId}",
                workingHours: ${buildWorkingHoursBody(workingHours)},
                reassignments: ${buildReassignmentsBody(reassignments)}
            ) {
                ${STAFF_FIELDS}
            }
        }
    `;
    return await doGraphQL<{ updateStaffWorkingHours: Staff }>(query, token);
};