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
            }
        }
    `

    const profileData = await doGraphQL<{userProfile: User}>(query, token);
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