import { doGraphQL } from "../api/graphql";

export interface SlotPackage {
    slotPackageId: string;
    servicePackage: {
        servicePackageId: string;
        servicePackageName: string;
        service: { serviceId: string; serviceName: string };
    };
}

export interface ServiceSlot {
    serviceSlotId: string;
    date: string;
    startTime: string; // RFC3339, e.g. "0000-01-01T09:00:00Z"
    endTime: string;
    staff?: { staffId: string; name: string } | null;
    serviceSlotPackages: SlotPackage[];
    hasBooking: boolean;
}

export interface ServiceSlotInput {
    staffId: string;        // "" => unassigned (owner-managed)
    date?: string;          // "YYYY-MM-DD" single date, OR ...
    daysOfWeek: string[];   // ... recurring weekdays e.g. ["monday"]
    startTime: string;      // "HH:MM"
    endTime: string;        // "HH:MM"
    servicePackageIds: string[];
}

// The Time scalar is RFC3339; the backend only cares about the clock part.
export const toTimeScalar = (hhmm: string): string => {
    const normalized = hhmm.length === 5 ? `${hhmm}:00` : hhmm;
    return `1970-01-01T${normalized}Z`;
};

// Extract "HH:MM" from an RFC3339 time string returned by the server.
export const extractTime = (rfc: string): string => {
    const match = rfc.match(/T(\d{2}:\d{2})/);
    return match ? match[1] : rfc;
};

const SLOT_FIELDS = `
    serviceSlotId
    date
    startTime
    endTime
    staff { staffId name }
    serviceSlotPackages {
        slotPackageId
        servicePackage {
            servicePackageId
            servicePackageName
            service { serviceId serviceName }
        }
    }
    hasBooking
`;

// Builds the ServiceSlotInput body shared by createServiceSlot/updateServiceSlot.
const buildInputBody = (input: ServiceSlotInput): string => {
    const staffLine = input.staffId ? `staffId: "${input.staffId}",` : "";
    // Either recurring weekdays (GraphQL enum literals, unquoted) or a single date.
    const scheduleLine = input.daysOfWeek.length > 0
        ? `daysOfWeek: [${input.daysOfWeek.join(", ")}],`
        : `date: "${input.date}",`;
    return `
        ${staffLine}
        ${scheduleLine}
        startTime: "${toTimeScalar(input.startTime)}",
        endTime: "${toTimeScalar(input.endTime)}",
        servicePackageIds: [${input.servicePackageIds.map(id => `"${id}"`).join(", ")}]
    `;
};

export const getServiceSlots = async (
    token: string, date: string, staffId?: string, serviceId?: string, unassignedOnly?: boolean,
) => {
    const args = [`date: "${date}"`];
    if (staffId) args.push(`staffId: "${staffId}"`);
    if (serviceId) args.push(`serviceId: "${serviceId}"`);
    if (unassignedOnly) args.push(`unassignedOnly: true`);
    const query = `
        query {
            displayServiceSlots(${args.join(", ")}) {
                ${SLOT_FIELDS}
            }
        }
    `;
    return await doGraphQL<{ displayServiceSlots: ServiceSlot[] }>(query, token);
};

export const createServiceSlot = async (token: string, input: ServiceSlotInput) => {
    const query = `
        mutation {
            createServiceSlot(input: { ${buildInputBody(input)} }) {
                ${SLOT_FIELDS}
            }
        }
    `;
    return await doGraphQL<{ createServiceSlot: ServiceSlot }>(query, token);
};

// applyToFutureRecurring: false edits just this occurrence; true edits it plus
// every future occurrence sharing its staff/time/weekday (mirrors deleteServiceSlot's flag).
export const updateServiceSlot = async (
    token: string, serviceSlotId: string, input: ServiceSlotInput, applyToFutureRecurring: boolean,
) => {
    const query = `
        mutation {
            updateServiceSlot(
                serviceSlotId: "${serviceSlotId}",
                input: { ${buildInputBody(input)} },
                applyToFutureRecurring: ${applyToFutureRecurring}
            ) {
                ${SLOT_FIELDS}
            }
        }
    `;
    return await doGraphQL<{ updateServiceSlot: ServiceSlot }>(query, token);
};

// staffId null => unassign the slot (owner-managed).
export const reassignServiceSlotStaff = async (token: string, serviceSlotId: string, staffId: string | null) => {
    const staffArg = staffId ? `"${staffId}"` : "null";
    const query = `
        mutation {
            reassignServiceSlotStaff(serviceSlotId: "${serviceSlotId}", staffId: ${staffArg}) {
                ${SLOT_FIELDS}
            }
        }
    `;
    return await doGraphQL<{ reassignServiceSlotStaff: ServiceSlot }>(query, token);
};

export const deleteServiceSlot = async (token: string, serviceSlotId: string, deleteFutureRecurring: boolean) => {
    const query = `
        mutation {
            deleteServiceSlot(serviceSlotId: "${serviceSlotId}", deleteFutureRecurring: ${deleteFutureRecurring})
        }
    `;
    return await doGraphQL<{ deleteServiceSlot: boolean }>(query, token);
};

export const getAvailableStaffForSlot = async (token: string, serviceSlotId: string) => {
    const query = `
        query {
            availableStaffForSlot(serviceSlotId: "${serviceSlotId}") {
                staffId
                name
            }
        }
    `;
    return await doGraphQL<{ availableStaffForSlot: { staffId: string; name: string }[] }>(query, token);
};
