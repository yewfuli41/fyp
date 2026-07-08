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

export interface PublicServiceOption {
    serviceOptionId: string;
    serviceOptionName: string;
    description?: string;
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

export const getAvailableSlots = (
    businessId: string,
    serviceOptionId: string,
    date: string,
    staffId?: string,
) => {
    const staffArg = staffId ? `, staffId: "${staffId}"` : "";
    const query = `
        query {
            availableSlots(businessId: "${businessId}", serviceOptionId: "${serviceOptionId}", date: "${date}"${staffArg}) {
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
