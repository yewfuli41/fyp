import { doGraphQL } from "../api/graphql";

export interface StatusCount {
    status: string;
    count: number;
}

export interface TypeCount {
    bookingType: string;
    count: number;
}

export interface BookingSummary {
    totalBookings: number;
    byStatus: StatusCount[];
    byType: TypeCount[];
    todayAcceptedCount: number;
    pendingCount: number;
}

export interface BookingTrendPoint {
    date: string;
    count: number;
}

export type TrendGranularity = "day" | "week" | "month";

export interface ServicePopularity {
    serviceId: string;
    serviceName: string;
    bookingCount: number;
}

export type SlotUtilizationGroupBy = "weekday" | "date" | "month" | "year";

// One point within a section — label is a weekday name ("monday"), a
// zero-padded day-of-month ("15"), a zero-padded month ("07"), or a year
// ("2026"), depending on the requested groupBy.
export interface UtilizationBreakdownPoint {
    label: string;
    totalSlots: number;
    bookedSlots: number;
}

// Groups related points under one subtitle — "date" grouping sections by
// month ("2026-07"), "month" grouping sections by year ("2026"). "weekday"
// and "year" groupings have no natural subtitle, so they come back as a
// single section with an empty subtitle.
export interface UtilizationSection {
    subtitle: string;
    points: UtilizationBreakdownPoint[];
}

// "weekday" stays scoped to the top-level from/until range. "date", "month",
// and "year" all span the business's whole history instead (its own
// creation date through until), since picking one of those means wanting
// the full calendar, not a window that happens to match the top-level filter.
export interface SlotUtilization {
    totalSlots: number;
    bookedSlots: number;
    utilizationRate: number;
    sections: UtilizationSection[];
}

// staffId is null for the "Owner-managed" bucket (unassigned slots).
export interface StaffUtilization {
    staffId: string | null;
    staffName: string;
    bookings: number;
    hoursBooked: number;
}

export interface CustomerRetention {
    newCustomers: number;
    returningCustomers: number;
    totalCustomers: number;
}

// Cancelled and rejected are combined here — both mean the same thing to
// the business: a booking that didn't happen.
export interface ServiceCancellation {
    serviceId: string;
    serviceName: string;
    count: number;
}

// customerCount + staffCount = the total cancelled/rejected for this staff
// (or "Owner-managed", when staffId is null). Rejections are always
// staff-initiated (a customer can't reject their own booking), so
// staffCount also absorbs every rejection.
export interface StaffCancellation {
    staffId: string | null;
    staffName: string;
    customerCount: number;
    staffCount: number;
}

export interface CancellationAnalysis {
    totalCancelled: number;
    totalRejected: number;
    totalCancelledOrRejected: number;
    cancelledOrRejectedRate: number;
    byService: ServiceCancellation[];
    byStaff: StaffCancellation[];
}

// One entry in the cancellation analysis's service filter — unlike
// displayServices, this includes soft-deleted services, since a deleted
// service's past cancellations/rejections still show up in byService.
// Grouped by name, same as byService: a service deleted and recreated under
// the same name is one option, and serviceIds is every service_id that ever
// used this name — pass all of them to getCancellationAnalysis when this
// option is selected. deleted is true only when none of them are active.
export interface ServiceFilterOption {
    serviceIds: string[];
    serviceName: string;
    deleted: boolean;
}

export const getBookingSummary = (token: string, from: string, until: string) => {
    const query = `
        query {
            bookingSummary(from: "${from}", until: "${until}") {
                totalBookings
                byStatus { status count }
                byType { bookingType count }
                todayAcceptedCount
                pendingCount
            }
        }
    `;
    return doGraphQL<{ bookingSummary: BookingSummary }>(query, token);
};

export const getBookingTrend = (token: string, from: string, until: string, granularity: TrendGranularity = "day") => {
    const query = `
        query {
            bookingTrend(from: "${from}", until: "${until}", granularity: ${granularity}) {
                date
                count
            }
        }
    `;
    return doGraphQL<{ bookingTrend: BookingTrendPoint[] }>(query, token);
};

export const getServicePopularity = (token: string, from: string, until: string) => {
    const query = `
        query {
            servicePopularity(from: "${from}", until: "${until}") {
                serviceId
                serviceName
                bookingCount
            }
        }
    `;
    return doGraphQL<{ servicePopularity: ServicePopularity[] }>(query, token);
};

export const getSlotUtilization = (token: string, from: string, until: string, groupBy: SlotUtilizationGroupBy = "weekday") => {
    const query = `
        query {
            slotUtilization(from: "${from}", until: "${until}", groupBy: ${groupBy}) {
                totalSlots
                bookedSlots
                utilizationRate
                sections { subtitle points { label totalSlots bookedSlots } }
            }
        }
    `;
    return doGraphQL<{ slotUtilization: SlotUtilization }>(query, token);
};

export const getStaffUtilization = (token: string, from: string, until: string) => {
    const query = `
        query {
            staffUtilization(from: "${from}", until: "${until}") {
                staffId
                staffName
                bookings
                hoursBooked
            }
        }
    `;
    return doGraphQL<{ staffUtilization: StaffUtilization[] }>(query, token);
};

export const getCustomerRetention = (token: string, from: string, until: string) => {
    const query = `
        query {
            customerRetention(from: "${from}", until: "${until}") {
                newCustomers
                returningCustomers
                totalCustomers
            }
        }
    `;
    return doGraphQL<{ customerRetention: CustomerRetention }>(query, token);
};

export const getCancellationAnalysis = (
    token: string, from: string, until: string, staffIds?: string[], serviceIds?: string[],
) => {
    const idsArg = (ids: string[]) => JSON.stringify(ids);
    const staffArg = staffIds?.length ? `, staffIds: ${idsArg(staffIds)}` : "";
    const serviceArg = serviceIds?.length ? `, serviceIds: ${idsArg(serviceIds)}` : "";
    const query = `
        query {
            cancellationAnalysis(from: "${from}", until: "${until}"${staffArg}${serviceArg}) {
                totalCancelled
                totalRejected
                totalCancelledOrRejected
                cancelledOrRejectedRate
                byService { serviceId serviceName count }
                byStaff { staffId staffName customerCount staffCount }
            }
        }
    `;
    return doGraphQL<{ cancellationAnalysis: CancellationAnalysis }>(query, token);
};

export const getCancellationServiceOptions = (token: string) => {
    const query = `
        query {
            cancellationServiceOptions {
                serviceIds
                serviceName
                deleted
            }
        }
    `;
    return doGraphQL<{ cancellationServiceOptions: ServiceFilterOption[] }>(query, token);
};

// The caller's own business's creation date — used to bound the dashboard's
// Years filter to years the business could actually have data in.
export const getBusinessCreatedAt = (token: string) => {
    const query = `query { businessCreatedAt }`;
    return doGraphQL<{ businessCreatedAt: string }>(query, token);
};
