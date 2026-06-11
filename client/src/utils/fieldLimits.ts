// Maximum input lengths, mirroring the VARCHAR limits in the database schema
// (see database/schema.sql). Used for `maxLength` on form inputs so over-long
// values are blocked client-side; the backend enforces the same limits.
export const FIELD_LIMITS = {
    username: 100,
    email: 255,
    contactNumber: 30,
    businessName: 255,
    businessEmail: 255,
    businessContactNumber: 30,
    serviceName: 255,
    servicePackageName: 255,
    packageItemName: 255,
} as const;
