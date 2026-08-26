import { doGraphQL } from "../api/graphql";

export interface PublicBusiness {
    businessId: string;
    businessName: string;
    description?: string;
    address: string;
    imageUrl?: string;
    businessContactNumber: string;
    businessEmail: string;
}

export interface PublicServiceOptionItem {
    serviceOptionItemId: string;
    serviceOptionItemName: string;
}

export interface PublicServiceOption {
    serviceOptionId: string;
    serviceOptionName: string;
    description?: string;
    serviceOptionItems: PublicServiceOptionItem[];
    effectiveFrom?: string;
    effectiveUntil?: string;
}

export interface PublicService {
    serviceId: string;
    serviceName: string;
    description?: string;
    serviceOptions: PublicServiceOption[];
}

export interface AvailableSlot {
    serviceSlotId: string;
    staffId?: string;
    staff?: { name: string };
    date: string;
    startTime: string;
    endTime: string;
    serviceSlotOptions: { slotOptionId: string; serviceOptionId: string }[];
}

export interface PublicStaff {
    staffId: string;
    name: string;
}

export const getPublicBusinesses = () => {
    const query = `
        query {
            publicBusinesses {
                businessId
                businessName
                description
                address
                imageUrl
                businessContactNumber
                businessEmail
            }
        }
    `;
    return doGraphQL<{ publicBusinesses: PublicBusiness[] }>(query);
};

export const getRecentlyBookedBusinesses = (token: string) => {
    const query = `
        query {
            recentlyBookedBusinesses {
                businessId
                businessName
                description
                address
                imageUrl
                businessContactNumber
                businessEmail
            }
        }
    `;
    return doGraphQL<{ recentlyBookedBusinesses: PublicBusiness[] }>(query, token);
};

export const getPublicServices = (businessId: string) => {
    const query = `
        query {
            publicServices(businessId: "${businessId}") {
                serviceId
                serviceName
                description
                serviceOptions {
                    serviceOptionId
                    serviceOptionName
                    description
                    effectiveFrom
                    effectiveUntil
                    serviceOptionItems {
                        serviceOptionItemId
                        serviceOptionItemName
                    }
                }
            }
        }
    `;
    return doGraphQL<{ publicServices: PublicService[] }>(query);
};

export const getPublicStaff = (businessId: string) => {
    const query = `
        query {
            publicStaff(businessId: "${businessId}") {
                staffId
                name
            }
        }
    `;
    return doGraphQL<{ publicStaff: PublicStaff[] }>(query);
};

// Hides one staff member's slots across a date range, so they're never
// offered as a reschedule target. Set while approving that staff's leave —
// the application is still pending then, so nothing else knows to keep the
// owner from moving a customer onto a day the staff is about to take off.
export interface StaffUnavailability {
    staffId: string;
    from: string;
    until: string;
}

const unavailableArgs = (u?: StaffUnavailability) =>
    u ? `, unavailableStaffId: "${u.staffId}", unavailableFrom: "${u.from}", unavailableUntil: "${u.until}"` : "";

export const getAvailableSlots = (
    businessId: string,
    serviceOptionId: string,
    date: string,
    staffId?: string,
    unassignedOnly?: boolean,
    unavailable?: StaffUnavailability,
) => {
    const staffArg = staffId ? `, staffId: "${staffId}"` : "";
    const unassignedArg = unassignedOnly ? `, unassignedOnly: true` : "";
    const query = `
        query {
            availableSlots(businessId: "${businessId}", serviceOptionId: "${serviceOptionId}", date: "${date}"${staffArg}${unassignedArg}${unavailableArgs(unavailable)}) {
                serviceSlotId
                staffId
                staff { name }
                date
                startTime
                endTime
                serviceSlotOptions { slotOptionId serviceOptionId }
            }
        }
    `;
    return doGraphQL<{ availableSlots: AvailableSlot[] }>(query);
};

export const getAvailableDates = (
    businessId: string,
    from: string,
    until: string,
    serviceId?: string,
    staffId?: string,
    serviceOptionId?: string,
    unassignedOnly?: boolean,
    unavailable?: StaffUnavailability,
) => {
    const serviceArg = serviceId ? `, serviceId: "${serviceId}"` : "";
    const optionArg = serviceOptionId ? `, serviceOptionId: "${serviceOptionId}"` : "";
    const staffArg = staffId ? `, staffId: "${staffId}"` : "";
    const unassignedArg = unassignedOnly ? `, unassignedOnly: true` : "";
    const query = `
        query {
            availableDates(businessId: "${businessId}", from: "${from}", until: "${until}"${serviceArg}${optionArg}${staffArg}${unassignedArg}${unavailableArgs(unavailable)})
        }
    `;
    return doGraphQL<{ availableDates: string[] }>(query);
};
