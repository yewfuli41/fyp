import type { AuthPayload } from "../auth/AuthContext"
import {doGraphQL} from "../api/graphql"

export const signUp = async(
    username: string,
    email: string,
    contactNumber: string,
    password: string,
) => {
    const query = `
        mutation {
            signUp(user:{
                username: "${username}",
                email: "${email}",
                contactNumber: "${contactNumber}",
                password: "${password}",
            }){
                token
                user{
                    userId
                    username
                    email
                    contactNumber
                }
            }
        }
    `

    const signUpData = await doGraphQL<{signUp: AuthPayload}>(query);
    return signUpData
}