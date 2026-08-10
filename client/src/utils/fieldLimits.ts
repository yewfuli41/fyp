// Maximum input lengths, mirroring the VARCHAR limits in the database schema
// (see database/schema.sql). Used for `maxLength` on form inputs so over-long
// values are blocked client-side; the backend enforces the same limits.

export const MIN_PASSWORD_LENGTH = 8;
export const MIN_CONTACT_NUMBER_DIGITS = 10;

export const isValidEmail = (v: string) => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v);
export const isValidContactNumber = (v: string) => v.replace(/\D/g, "").length >= MIN_CONTACT_NUMBER_DIGITS;
export const isValidPassword = (v: string) => v.length >= MIN_PASSWORD_LENGTH;

export function shouldClearEmailError(error: string | undefined, value: string): boolean {
  if (!error) return false;
  if (error.endsWith(" is required")) return value.trim() !== "";
  if (error.endsWith("format is invalid")) return isValidEmail(value);
  return false;
}

export function shouldClearContactNumberError(error: string | undefined, value: string): boolean {
  if (!error) return false;
  const lower = error.toLowerCase();
  if (lower.endsWith(" is required")) return value.trim() !== "";
  if (lower.includes("must have at least")) return isValidContactNumber(value);
  return false;
}

export function shouldClearPasswordError(error: string | undefined, value: string): boolean {
  if (!error) return false;
  if (error === "Password is required") return value.trim() !== "";
  if (error.startsWith("Password must be at least")) return isValidPassword(value);
  return false;
}

export const FIELD_LIMITS = {
    username: 100,
    email: 255,
    contactNumber: 30,
    businessName: 255,
    businessEmail: 255,
    businessContactNumber: 30,
    serviceName: 255,
    serviceOptionName: 255,
    serviceOptionItemName: 255,
    staffName: 255,
    staffContactNumber: 30,
    position: 100,
} as const;
