import type { ServiceInput, ServicePackageInput } from "../services/ServiceService";

export const emptyPackage = (serviceName: string): ServicePackageInput => ({
    servicePackageName: "",
    description: "",
    packageItems: [{ packageItemName: serviceName }],
});

export const emptyInput = (): ServiceInput => ({
    serviceName: "",
    description: "",
    servicePackages: [],
});

// Keeps the default package named after the service, creating it on first
// keystroke and renaming it as the service name changes — mirrors the
// backend's ensureDefaultPackage. The default package is always at index 0:
// it's the only one ever prepended, addPackage only appends, and it can never
// be removed (no Remove button) — so position, not name matching, is what
// identifies it. Matching by name instead would misfire whenever a custom
// package's name coincidentally equals the service name.
export const syncDefaultPackage = (packages: ServicePackageInput[], newName: string): ServicePackageInput[] => {
    if (!newName.trim()) return packages;
    if (packages.length === 0) {
        return [{ servicePackageName: newName, description: "", packageItems: [{ packageItemName: newName }] }];
    }
    const updated = [...packages];
    updated[0] = { ...updated[0], servicePackageName: newName };
    return updated;
};

// Keeps every package's default item (named after the service, not the
// package) in sync — mirrors the backend's ensureDefaultPackageItems. Like the
// default package, the default item always lives at index 0 of each package's
// items (only ever created there; addItem appends, never removed), so it's
// identified by position rather than by matching text.
export const syncDefaultItems = (packages: ServicePackageInput[], newName: string): ServicePackageInput[] => {
    if (!newName.trim()) return packages;
    return packages.map(pkg => {
        const items = [...pkg.packageItems];
        if (items.length === 0) {
            items.push({ packageItemName: newName });
        } else {
            items[0] = { packageItemName: newName };
        }
        return { ...pkg, packageItems: items };
    });
};
