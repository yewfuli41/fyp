import {doGraphQL} from "../api/graphql"

export interface WorkingHour {
    day: string;
    startTime: string;
    endTime: string;
}

export interface StaffInput{
    name: string;
    email: string;
    contactNumber?: string;
    position?: string;
    workingHours: WorkingHour[];
}

export const registerStaff = async(token: string, staff:StaffInput) => {
    const query = `
        mutation {
            registerStaff(staff:{
                name: "${staff.name}",
                email: "${staff.email}",
                contactNumber: "${staff.contactNumber}",
                position: "${staff.position}",
            )
        }
    `

    return await doGraphQL(query, token);
}