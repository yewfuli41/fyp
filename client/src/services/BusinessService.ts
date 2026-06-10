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

export const registerBusinessProfile = async (token: string, business: BusinessProfileInput) => {
    const query = `
        mutation {
            registerBusinessProfile(business: {
                businessName: "${business.businessName}",
                description: ${business.description ? `"${business.description}"` : "null"},
                address: "${business.address}",
                imageUrl: ${business.imageUrl ? `"${business.imageUrl}"` : "null"},
                businessContactNumber: "${business.businessContactNumber}",
                businessEmail: "${business.businessEmail}",
                workingHours: [
                    ${business.workingHours.map(wh => `{
                        day: ${wh.day.toLowerCase()},
                        startTime: "${wh.startTime}",
                        endTime: "${wh.endTime}"
                    }`).join(",")}
                ]
            }) {
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
            updateBusinessProfile(business: {
                businessName: "${business.businessName}",
                description: ${business.description ? `"${business.description}"` : "null"},
                address: "${business.address}",
                imageUrl: ${business.imageUrl ? `"${business.imageUrl}"` : "null"},
                businessContactNumber: "${business.businessContactNumber}",
                businessEmail: "${business.businessEmail}",
                workingHours: [
                    ${business.workingHours.map(wh => `{
                        day: ${wh.day.toLowerCase()},
                        startTime: "${wh.startTime}",
                        endTime: "${wh.endTime}"
                    }`).join(",")}
                ]
            }) {
                businessId
                businessName
            }
        }
    `

    return await doGraphQL(query, token);
}
