import type { AuthPayload } from "../auth/AuthContext"
import {doGraphQL} from "../api/graphql"

export const logIn = async(
    email: string,
    password: string,
) => {
    const query = `
        mutation {
            logIn(user:{
                email: "${email}",
                password: "${password}",
            }){
                token
                user{
                    userId
                    username
                    email
                    contactNumber
                    mustResetPassword
                    businessProfile {
                        businessId
                    }

                    staffProfile {
                        staffId
                    }
                }
            }
        }
    `

    const logInData = await doGraphQL<{logIn: AuthPayload}>(query);
    return logInData
}
