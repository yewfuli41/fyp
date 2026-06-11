import { doGraphQL } from "../api/graphql"

export interface WorkingHour {
    day: string;
    startTime: string;
    endTime: string;
}

export interface BusinessProfileInput {
    businessName: string;
    description?: string;
    address: string;
    imageUrl?: string;
    businessContactNumber: string;
    businessEmail: string;
    workingHours: WorkingHour[];
}

// The backend exposes working-hour start/end as the GraphQL `Time` scalar,
// which expects an RFC3339 timestamp. Working hours are a time-of-day only, so
// anchor them to a fixed date.
const toTimeScalar = (time: string): string => {
    const normalized = time.length === 5 ? `${time}:00` : time; // HH:MM -> HH:MM:SS
    return `1970-01-01T${normalized}Z`;
};

const buildBusinessProfileInput = (business: BusinessProfileInput): string => `{
                businessName: "${business.businessName}",
                description: ${business.description ? `"${business.description}"` : "null"},
                address: "${business.address}",
                imageUrl: ${business.imageUrl ? `"${business.imageUrl}"` : "null"},
                businessContactNumber: "${business.businessContactNumber}",
                businessEmail: "${business.businessEmail}",
                workingHours: [
                    ${business.workingHours.map(wh => `{
                        day: ${wh.day.toLowerCase()},
                        startTime: "${toTimeScalar(wh.startTime)}",
                        endTime: "${toTimeScalar(wh.endTime)}"
                    }`).join(",")}
                ]
            }`;

export const registerBusinessProfile = async (token: string, business: BusinessProfileInput) => {
    const query = `
        mutation {
            registerBusinessProfile(business: ${buildBusinessProfileInput(business)}) {
                businessId
                businessName
            }
        }
    `

    return await doGraphQL(query, token);
}

export const updateBusinessProfile = async (token: string, business: BusinessProfileInput) => {
    const query = `
        mutation {
            updateBusinessProfile(business: ${buildBusinessProfileInput(business)}) {
                businessId
                businessName
            }
        }
    `

    return await doGraphQL(query, token);
}
