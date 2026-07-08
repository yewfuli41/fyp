import type { ServiceInput, ServiceOptionInput } from "../services/ServiceService";

export const emptyOption = (): ServiceOptionInput => ({
    serviceOptionName: "",
    description: "",
    serviceOptionItems: [],
});

export const emptyInput = (): ServiceInput => ({
    serviceName: "",
    description: "",
    serviceOptions: [],
});

// Keeps option[0] in sync with the service name as the owner types.
// Option[0] is still editable — if the owner changes it manually that value
// will be overwritten on the next service-name keystroke, but the backend
// falls back to the service name if the field is left blank anyway.
export const syncDefaultOption = (options: ServiceOptionInput[], newName: string): ServiceOptionInput[] => {
    if (!newName.trim()) return options;
    if (options.length === 0) {
        return [{ serviceOptionName: newName, description: "", serviceOptionItems: [] }];
    }
    const updated = [...options];
    updated[0] = { ...updated[0], serviceOptionName: newName };
    return updated;
};
