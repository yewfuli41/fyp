import { doGraphQL } from "../api/graphql";

export interface PackageItemInput {
    packageItemName: string;
}

export interface ServicePackageInput {
    servicePackageName: string;
    description?: string;
    packageItems: PackageItemInput[];
}

export interface ServiceInput {
    serviceName: string;
    description?: string;
    servicePackages: ServicePackageInput[];
}

export interface PackageItem {
    packageItemId: string;
    packageItemName: string;
}

export interface ServicePackage {
    servicePackageId: string;
    servicePackageName: string;
    description?: string;
    packageItems: PackageItem[];
}

export interface Service {
    serviceId: string;
    serviceName: string;
    description?: string;
    servicePackages: ServicePackage[];
}

const SERVICE_FIELDS = `
    serviceId
    serviceName
    description
    servicePackages {
        servicePackageId
        servicePackageName
        description
        packageItems {
            packageItemId
            packageItemName
        }
    }
`;

const serializeInput = (service: ServiceInput): string => `{
    serviceName: ${JSON.stringify(service.serviceName)},
    description: ${service.description ? JSON.stringify(service.description) : "null"},
    servicePackages: [${service.servicePackages.map(pkg => `{
        servicePackageName: ${JSON.stringify(pkg.servicePackageName)},
        description: ${pkg.description ? JSON.stringify(pkg.description) : "null"},
        packageItems: [${pkg.packageItems.map(item => `{
            packageItemName: ${JSON.stringify(item.packageItemName)}
        }`).join(",")}]
    }`).join(",")}]
}`;

export const getBusinessServices = async (token: string) => {
    const query = `
        query {
            businessServices {
                ${SERVICE_FIELDS}
            }
        }
    `;
    return await doGraphQL<{ businessServices: Service[] }>(query, token);
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
