import type { User } from "../auth/AuthContext"
import {doGraphQL} from "../api/graphql"

export const userProfile= async(token: string) => { 
    const query = `
        query {
            userProfile{
                userId
                username
                email
                contactNumber
                businessProfile {
                    businessId
                    businessName
                    description
                    address
                    imageUrl
                    businessContactNumber
                    businessEmail
                    workingHours {
                        day
                        startTime
                        endTime
                    }
                }
                staffProfile {
                    staffId
                    workingHours {
                        day
                        startTime
                        endTime
                    }
                }
            }
        }
    `

    const profileData = await doGraphQL<{userProfile: any}>(query, token);
    return profileData
}

export const updateProfile = async (token: string, username: string, email: string, contactNumber: string) => {
    const query = `
        mutation {
            updateProfile(user: {
                username: "${username}",
                email: "${email}",
                contactNumber: "${contactNumber}"
            }) {
                userId
                username
                email
                contactNumber
            }
        }
    `

    const updateData = await doGraphQL<{updateProfile: User}>(query, token);
    return updateData
}