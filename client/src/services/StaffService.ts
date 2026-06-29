import {doGraphQL} from "../api/graphql"

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
    workingHours {
        day
        startTime
        endTime
    }
`;

const toTimeScalar = (time: string): string => {
    const normalized = time.length === 5 ? `${time}:00` : time;
    return `1970-01-01T${normalized}Z`;
};

export const registerStaff = async(token: string, staff:StaffInput) => {
    const query = `
        mutation {
            registerStaff(staff:{
                name: "${staff.name}",
                email: "${staff.email}",
                contactNumber: "${staff.contactNumber}",
                position: "${staff.position}",
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