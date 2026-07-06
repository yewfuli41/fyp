import { doGraphQL } from "../api/graphql";

export interface ServiceOptionItemInput {
    serviceOptionItemName: string;
}

export interface ServiceOptionInput {
    serviceOptionName: string;
    description?: string;
    serviceOptionItems: ServiceOptionItemInput[];
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
        serviceOptionName: ${JSON.stringify(pkg.serviceOptionName)},
        description: ${pkg.description ? JSON.stringify(pkg.description) : "null"},
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
