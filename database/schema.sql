CREATE TABLE IF NOT EXISTS users (
    user_id BIGSERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    contact_number VARCHAR(30),
    password TEXT NOT NULL,
    failed_login_attempts INT NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ NULL,
    must_reset_password BOOLEAN NOT NULL DEFAULT FALSE,
    CHECK (failed_login_attempts >= 0)
);

CREATE TABLE IF NOT EXISTS business_profiles (
    business_id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL UNIQUE REFERENCES users(user_id) ON DELETE RESTRICT,
    business_name VARCHAR(255) NOT NULL,
    description TEXT,
    address TEXT,
    image_url TEXT,
    business_contact_number VARCHAR(30),
    business_email VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS business_working_hours (
    business_work_hour_id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES business_profiles(business_id) ON DELETE CASCADE,
    day VARCHAR(30) NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    deleted_at TIMESTAMPTZ,
    CHECK (day IN ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday')),
    CHECK (start_time < end_time)
);

CREATE TABLE IF NOT EXISTS services (
    service_id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES business_profiles(business_id) ON DELETE CASCADE,
    service_name VARCHAR(255) NOT NULL,
    description TEXT,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS service_packages (
    service_package_id BIGSERIAL PRIMARY KEY,
    service_id BIGINT NOT NULL REFERENCES services(service_id) ON DELETE CASCADE,
    service_package_name VARCHAR(255) NOT NULL,
    description TEXT,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS package_items (
    package_item_id BIGSERIAL PRIMARY KEY,
    service_package_id BIGINT NOT NULL REFERENCES service_packages(service_package_id) ON DELETE CASCADE,
    package_item_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS staff (
    staff_id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(user_id) ON DELETE RESTRICT,
    business_id BIGINT NOT NULL REFERENCES business_profiles(business_id) ON DELETE CASCADE,
    staff_name VARCHAR(255) NOT NULL,
    staff_contact_number VARCHAR(30),
    position VARCHAR(100),
    deleted_at TIMESTAMPTZ,
    UNIQUE (user_id, business_id)
);

CREATE TABLE IF NOT EXISTS staff_working_hours (
    staff_work_hour_id BIGSERIAL PRIMARY KEY,
    staff_id BIGINT NOT NULL REFERENCES staff(staff_id) ON DELETE CASCADE,
    day VARCHAR(30) NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    deleted_at TIMESTAMPTZ,
    CHECK (day IN ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday')),
    CHECK (start_time < end_time)
);

CREATE TABLE IF NOT EXISTS leave_applications (
    leave_id BIGSERIAL PRIMARY KEY,
    staff_id BIGINT NOT NULL REFERENCES staff(staff_id) ON DELETE CASCADE,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    justification TEXT,
    file_url TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    remark TEXT,
    decided_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (start_date <= end_date),
    CHECK (status IN ('pending', 'approved', 'rejected'))
);

CREATE TABLE IF NOT EXISTS recurring_schedules (
    recurring_schedule_id BIGSERIAL PRIMARY KEY,
    staff_id BIGINT NOT NULL REFERENCES staff(staff_id) ON DELETE CASCADE,
    day VARCHAR(30) NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES users(user_id) ON DELETE SET NULL,
    CHECK (day IN ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday')),
    CHECK (start_time < end_time)
);

CREATE TABLE IF NOT EXISTS service_slots (
    service_slot_id BIGSERIAL PRIMARY KEY,
    staff_id BIGINT REFERENCES staff(staff_id) ON DELETE CASCADE,
    recurring_schedule_id BIGINT REFERENCES recurring_schedules(recurring_schedule_id) ON DELETE CASCADE,
    date DATE NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES users(user_id) ON DELETE SET NULL,
    CHECK (start_time < end_time)
);

CREATE TABLE IF NOT EXISTS service_slot_packages (
    slot_package_id BIGSERIAL PRIMARY KEY,
    service_package_id BIGINT NOT NULL REFERENCES service_packages(service_package_id) ON DELETE CASCADE,
    service_slot_id BIGINT NOT NULL REFERENCES service_slots(service_slot_id) ON DELETE CASCADE,
    deleted_at TIMESTAMPTZ,
    UNIQUE (service_package_id, service_slot_id)
);

CREATE SEQUENCE IF NOT EXISTS booking_group_seq START 1;

CREATE TABLE IF NOT EXISTS bookings (
    booking_id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
    slot_package_id BIGINT NOT NULL REFERENCES service_slot_packages(slot_package_id) ON DELETE RESTRICT,
    booking_group_id BIGINT NOT NULL,
    status VARCHAR(30),
    booking_type VARCHAR(30) NOT NULL,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at TIMESTAMPTZ,
    decided_by BIGINT REFERENCES users(user_id) ON DELETE SET NULL,
    CHECK (status IN ('accepted', 'rescheduled', 'cancelled', 'past', 'rejected', 'pending')),
    CHECK (booking_type IN ('online', 'walk_in'))
);

CREATE INDEX IF NOT EXISTS idx_business_working_hours_business_id ON business_working_hours(business_id);
CREATE INDEX IF NOT EXISTS idx_services_business_id ON services(business_id);
CREATE INDEX IF NOT EXISTS idx_service_packages_service_id ON service_packages(service_id);
CREATE INDEX IF NOT EXISTS idx_package_items_service_package_id ON package_items(service_package_id);
CREATE INDEX IF NOT EXISTS idx_staff_user_id ON staff(user_id);
CREATE INDEX IF NOT EXISTS idx_staff_business_id ON staff(business_id);
CREATE INDEX IF NOT EXISTS idx_staff_working_hours_staff_id ON staff_working_hours(staff_id);
CREATE INDEX IF NOT EXISTS idx_leave_applications_staff_id ON leave_applications(staff_id);
CREATE INDEX IF NOT EXISTS idx_service_slots_staff_id ON service_slots(staff_id);
CREATE INDEX IF NOT EXISTS idx_service_slots_recurring_schedule_id ON service_slots(recurring_schedule_id);
CREATE INDEX IF NOT EXISTS idx_service_slot_packages_service_package_id ON service_slot_packages(service_package_id);
CREATE INDEX IF NOT EXISTS idx_service_slot_packages_service_slot_id ON service_slot_packages(service_slot_id);
CREATE INDEX IF NOT EXISTS idx_recurring_schedules_staff_id ON recurring_schedules(staff_id);
CREATE INDEX IF NOT EXISTS idx_bookings_user_id ON bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_slot_package_id ON bookings(slot_package_id);
CREATE UNIQUE INDEX services_unique_name
ON services (business_id, LOWER(service_name))
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX service_packages_unique_name
ON service_packages (service_id, LOWER(service_package_name))
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX package_items_unique_name
ON package_items (service_package_id, LOWER(package_item_name))
WHERE deleted_at IS NULL;
