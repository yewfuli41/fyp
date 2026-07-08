import { doGraphQL } from "../api/graphql";

export const createBooking = (token: string, slotOptionId: string) => {
    const query = `
        mutation {
            createBooking(slotOptionId: "${slotOptionId}") {
                bookingId
                status
            }
        }
    `;
    return doGraphQL<{ createBooking: { bookingId: string; status: string } }>(query, token);
};
