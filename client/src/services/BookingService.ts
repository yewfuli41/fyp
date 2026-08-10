import { doGraphQL } from "../api/graphql";
import { toTimeScalar } from "./ServiceSlotService";

export type BookingStatus =
    | "PENDING" | "ACCEPTED" | "RESCHEDULED" | "CANCELLED" | "REJECTED" | "PAST";

export interface BookingDetail {
    bookingId: string;
    status: BookingStatus;
    bookingType: string;
    serviceSlotId: string;
    slotOptionId: string;
    serviceOptionId: string;
    date: string;
    startTime: string;
    endTime: string;
    serviceName: string;
    optionName: string;
    staffName?: string | null;
    customerName: string;
    customerEmail: string;
    businessId: string;
    businessName: string;
    description?: string | null;
    createdAt: string;
}

export const BOOKING_FIELDS = `
    bookingId
    status
    bookingType
    serviceSlotId
    slotOptionId
    serviceOptionId
    date
    startTime
    endTime
    serviceName
    optionName
    staffName
    customerName
    customerEmail
    businessId
    businessName
    description
    createdAt
`;

export const createBooking = (token: string, slotOptionId: string, description?: string) => {
    const descArg = description ? `, description: ${JSON.stringify(description)}` : "";
    const query = `
        mutation {
            createBooking(slotOptionId: "${slotOptionId}"${descArg}) {
                bookingId
                status
            }
        }
    `;
    return doGraphQL<{ createBooking: { bookingId: string; status: string } }>(query, token);
};

export const getBusinessBookings = (token: string) => {
    const query = `query { businessBookings { ${BOOKING_FIELDS} } }`;
    return doGraphQL<{ businessBookings: BookingDetail[] }>(query, token);
};

export const getMyAppointments = (token: string) => {
    const query = `query { myAppointments { ${BOOKING_FIELDS} } }`;
    return doGraphQL<{ myAppointments: BookingDetail[] }>(query, token);
};

const singleBookingMutation = (name: string, bookingId: string) => `
    mutation { ${name}(bookingId: "${bookingId}") { ${BOOKING_FIELDS} } }
`;

export const acceptBooking = (token: string, bookingId: string) =>
    doGraphQL<{ acceptBooking: BookingDetail }>(singleBookingMutation("acceptBooking", bookingId), token);

export const rejectBooking = (token: string, bookingId: string) =>
    doGraphQL<{ rejectBooking: BookingDetail }>(singleBookingMutation("rejectBooking", bookingId), token);

export const cancelBooking = (token: string, bookingId: string) =>
    doGraphQL<{ cancelBooking: BookingDetail }>(singleBookingMutation("cancelBooking", bookingId), token);

export const acceptReschedule = (token: string, bookingId: string) =>
    doGraphQL<{ acceptReschedule: BookingDetail }>(singleBookingMutation("acceptReschedule", bookingId), token);

export const rescheduleBooking = (token: string, bookingId: string, newSlotOptionId: string) => {
    const query = `
        mutation {
            rescheduleBooking(bookingId: "${bookingId}", newSlotOptionId: "${newSlotOptionId}") {
                ${BOOKING_FIELDS}
            }
        }
    `;
    return doGraphQL<{ rescheduleBooking: BookingDetail }>(query, token);
};

export interface WalkInInput {
    serviceOptionId: string;
    staffId?: string;  // "" or omitted => owner-managed
    date: string;      // "YYYY-MM-DD"
    startTime: string; // "HH:MM"
    endTime: string;   // "HH:MM"
}

export const recordWalkIn = (token: string, input: WalkInInput) => {
    const staffLine = input.staffId ? `staffId: "${input.staffId}",` : "";
    const query = `
        mutation {
            recordWalkIn(input: {
                serviceOptionId: "${input.serviceOptionId}",
                ${staffLine}
                date: "${input.date}",
                startTime: "${toTimeScalar(input.startTime)}",
                endTime: "${toTimeScalar(input.endTime)}"
            }) {
                ${BOOKING_FIELDS}
            }
        }
    `;
    return doGraphQL<{ recordWalkIn: BookingDetail }>(query, token);
};

// Edits just the note on an existing booking — date/time/option stay fixed.
export const updateBookingDescription = (token: string, bookingId: string, description: string) => {
    const query = `
        mutation {
            updateBookingDescription(bookingId: "${bookingId}", description: ${JSON.stringify(description)}) {
                ${BOOKING_FIELDS}
            }
        }
    `;
    return doGraphQL<{ updateBookingDescription: BookingDetail }>(query, token);
};
