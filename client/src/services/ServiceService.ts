import { doGraphQL } from "../api/graphql";

export interface ServiceOptionItemInput {
    serviceOptionItemName: string;
}

export interface ServiceOptionInput {
    // Set when editing an existing option, which is entirely immutable — name,
    // items and effective window are all fixed at creation. The fields below
    // therefore only ever carry values for a brand new option (one submitted
    // with no serviceOptionId).
    serviceOptionId?: string;
    serviceOptionName: string;
    description?: string;
    serviceOptionItems: ServiceOptionItemInput[];
    effectiveFrom?: string;  // "YYYY-MM-DD"
    effectiveUntil?: string; // "YYYY-MM-DD"
    // Vestigial: this used to ask the server to clear an existing option's
    // saved effectiveUntil. A saved window can no longer be changed, so the
    // form never sets it and the server rejects it if some other client does
    // (a stale bundle, a saved Postman request). Kept on the wire because the
    // GraphQL input still declares it.
    clearEffectiveUntil?: boolean;
    // The option's saved effective window, as loaded from the server. For an
    // existing option this is the only window there is — the form shows it in
    // disabled inputs. Not sent to the server.
    currentEffectiveFrom?: string;
    currentEffectiveUntil?: string;
    // Read-only, as loaded from the server — lets the form refuse to delete
    // this option immediately instead of only finding out after a submit.
    // Not sent to the server.
    hasBooking?: boolean;
}

export interface ServiceInput {
    serviceName: string;
    description?: string;
    serviceOptions: ServiceOptionInput[];
}

export interface ServiceOptionItem {
    serviceOptionItemId: string;
    serviceOptionItemName: string;
}

export interface ServiceOption {
    serviceOptionId: string;
    serviceOptionName: string;
    description?: string;
    serviceOptionItems: ServiceOptionItem[];
    removed: boolean;
    effectiveFrom?: string;
    effectiveUntil?: string;
    // true once anything has ever booked this option — deleting it is
    // rejected server-side while this is true.
    hasBooking: boolean;
}

export interface Service {
    serviceId: string;
    serviceName: string;
    description?: string;
    serviceOptions: ServiceOption[];
}

const SERVICE_FIELDS = `
    serviceId
    serviceName
    description
    serviceOptions {
        serviceOptionId
        serviceOptionName
        description
        removed
        effectiveFrom
        effectiveUntil
        hasBooking
        serviceOptionItems {
            serviceOptionItemId
            serviceOptionItemName
        }
    }
`;

const serializeInput = (service: ServiceInput): string => `{
    serviceName: ${JSON.stringify(service.serviceName)},
    description: ${service.description ? JSON.stringify(service.description) : "null"},
    serviceOptions: [${service.serviceOptions.map(pkg => `{
        serviceOptionId: ${pkg.serviceOptionId ? JSON.stringify(pkg.serviceOptionId) : "null"},
        serviceOptionName: ${JSON.stringify(pkg.serviceOptionName)},
        description: ${pkg.description ? JSON.stringify(pkg.description) : "null"},
        effectiveFrom: ${pkg.effectiveFrom ? JSON.stringify(pkg.effectiveFrom) : "null"},
        effectiveUntil: ${pkg.effectiveUntil ? JSON.stringify(pkg.effectiveUntil) : "null"},
        clearEffectiveUntil: ${pkg.clearEffectiveUntil ? "true" : "false"},
        serviceOptionItems: [${pkg.serviceOptionItems.map(item => `{
            serviceOptionItemName: ${JSON.stringify(item.serviceOptionItemName)}
        }`).join(",")}]
    }`).join(",")}]
}`;

export const getBusinessServices = async (token: string) => {
    const query = `
        query {
            displayServices {
                ${SERVICE_FIELDS}
            }
        }
    `;
    return await doGraphQL<{ displayServices: Service[] }>(query, token);
};

export const createService = async (token: string, service: ServiceInput) => {
    const query = `
        mutation {
            createService(service: ${serializeInput(service)}) {
                ${SERVICE_FIELDS}
            }
        }
    `;
    return await doGraphQL<{ createService: Service }>(query, token);
};

export const updateService = async (token: string, serviceId: string, service: ServiceInput) => {
    const query = `
        mutation {
            updateService(serviceId: "${serviceId}", service: ${serializeInput(service)}) {
                ${SERVICE_FIELDS}
            }
        }
    `;
    return await doGraphQL<{ updateService: Service }>(query, token);
};

export const deleteService = async (token: string, serviceId: string) => {
    const query = `
        mutation {
            deleteService(serviceId: "${serviceId}")
        }
    `;
    return await doGraphQL<{ deleteService: boolean }>(query, token);
};
