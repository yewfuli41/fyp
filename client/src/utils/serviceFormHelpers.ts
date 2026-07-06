import type { ServiceInput, ServiceOptionInput } from "../services/ServiceService";

export const emptyOption = (serviceName: string): ServiceOptionInput => ({
    serviceOptionName: "",
    description: "",
    serviceOptionItems: [{ serviceOptionItemName: serviceName }],
});

export const emptyInput = (): ServiceInput => ({
    serviceName: "",
    description: "",
    serviceOptions: [],
});

// Keeps the default package named after the service, creating it on first
// keystroke and renaming it as the service name changes — mirrors the
// backend's ensureDefaultPackage. The default package is always at index 0:
// it's the only one ever prepended, addPackage only appends, and it can never
// be removed (no Remove button) — so position, not name matching, is what
// identifies it. Matching by name instead would misfire whenever a custom
// package's name coincidentally equals the service name.
export const syncDefaultOption = (packages: ServiceOptionInput[], newName: string): ServiceOptionInput[] => {
    if (!newName.trim()) return packages;
    if (packages.length === 0) {
        return [{ serviceOptionName: newName, description: "", serviceOptionItems: [{ serviceOptionItemName: newName }] }];
    }
    const updated = [...packages];
    updated[0] = { ...updated[0], serviceOptionName: newName };
    return updated;
};

// Keeps every package's default item (named after the service, not the
// package) in sync — mirrors the backend's ensureDefaultServiceOptionItems. Like the
// default package, the default item always lives at index 0 of each package's
// items (only ever created there; addItem appends, never removed), so it's
// identified by position rather than by matching text.
export const syncDefaultItems = (packages: ServiceOptionInput[], newName: string): ServiceOptionInput[] => {
    if (!newName.trim()) return packages;
    return packages.map(pkg => {
        const items = [...pkg.serviceOptionItems];
        if (items.length === 0) {
            items.push({ serviceOptionItemName: newName });
        } else {
            items[0] = { serviceOptionItemName: newName };
        }
        return { ...pkg, serviceOptionItems: items };
    });
};
