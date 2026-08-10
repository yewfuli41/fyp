import type { ServiceInput, ServiceOptionInput } from "../services/ServiceService";

export type OptionStatus = "upcoming" | "ongoing" | "expired";

// Where an option sits relative to today, given its effective window.
export const getOptionStatus = (effectiveFrom?: string, effectiveUntil?: string): OptionStatus => {
    const today = new Date().toISOString().slice(0, 10);
    if (effectiveFrom && effectiveFrom > today) return "upcoming";
    if (effectiveUntil && effectiveUntil < today) return "expired";
    return "ongoing";
};

export const OPTION_STATUS_LABEL: Record<OptionStatus, string> = {
    upcoming: "Upcoming",
    ongoing: "Ongoing",
    expired: "Previous",
};

export const OPTION_STATUS_VARIANT: Record<OptionStatus, string> = {
    upcoming: "info",
    ongoing: "success",
    expired: "secondary",
};

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

// Keeps a brand-new (unsaved) option[0] in sync with the service name as the
// owner types, so its name is prefilled sensibly. An already-saved option's
// name can no longer be edited directly (see ServiceFormModal — changing it
// requires removing it and adding a replacement), so a saved option[0] is
// left untouched here even as the service name changes.
export const syncDefaultOption = (options: ServiceOptionInput[], newName: string): ServiceOptionInput[] => {
    if (!newName.trim()) return options;
    if (options.length === 0) {
        return [{ serviceOptionName: newName, description: "", serviceOptionItems: [] }];
    }
    if (options[0].serviceOptionId) return options;
    const updated = [...options];
    updated[0] = { ...updated[0], serviceOptionName: newName };
    return updated;
};
