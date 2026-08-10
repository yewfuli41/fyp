CREATE TABLE IF NOT EXISTS fyp_fuli_users (
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

CREATE TABLE IF NOT EXISTS fyp_fuli_business_profiles (
    business_id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL UNIQUE REFERENCES fyp_fuli_users(user_id) ON DELETE RESTRICT,
    business_name VARCHAR(255) NOT NULL,
    description TEXT,
    address TEXT,
    image_url TEXT,
    business_contact_number VARCHAR(30),
    business_email VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS fyp_fuli_business_working_hours (
    business_work_hour_id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES fyp_fuli_business_profiles(business_id) ON DELETE CASCADE,
    day VARCHAR(30) NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    deleted_at TIMESTAMPTZ,
    CHECK (day IN ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday')),
    CHECK (start_time < end_time)
);

CREATE TABLE IF NOT EXISTS fyp_fuli_services (
    service_id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES fyp_fuli_business_profiles(business_id) ON DELETE CASCADE,
    service_name VARCHAR(255) NOT NULL,
    description TEXT,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS fyp_fuli_service_options (
    service_option_id BIGSERIAL PRIMARY KEY,
    service_id BIGINT NOT NULL REFERENCES fyp_fuli_services(service_id) ON DELETE CASCADE,
    service_option_name VARCHAR(255) NOT NULL,
    description TEXT,
    deleted_at TIMESTAMPTZ,
    -- An option's own validity window — name/items are immutable once
    -- created, so this is just when this specific option is offered, not a
    -- pointer into a version history.
    effective_from DATE NOT NULL DEFAULT CURRENT_DATE,
    effective_until DATE,
    -- True for exactly one option per service. The default can't be removed
    -- or given an end date, so a service always has one option available.
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    CHECK (effective_until IS NULL OR effective_until >= effective_from)
);

CREATE TABLE IF NOT EXISTS fyp_fuli_service_option_items (
    service_option_item_id BIGSERIAL PRIMARY KEY,
    service_option_id BIGINT NOT NULL REFERENCES fyp_fuli_service_options(service_option_id) ON DELETE CASCADE,
    service_option_item_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS fyp_fuli_staff (
    staff_id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES fyp_fuli_users(user_id) ON DELETE RESTRICT,
    business_id BIGINT NOT NULL REFERENCES fyp_fuli_business_profiles(business_id) ON DELETE CASCADE,
    staff_name VARCHAR(255) NOT NULL,
    staff_contact_number VARCHAR(30),
    position VARCHAR(100),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS fyp_fuli_staff_working_hours (
    staff_work_hour_id BIGSERIAL PRIMARY KEY,
    staff_id BIGINT NOT NULL REFERENCES fyp_fuli_staff(staff_id) ON DELETE CASCADE,
    day VARCHAR(30) NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    deleted_at TIMESTAMPTZ,
    CHECK (day IN ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday')),
    CHECK (start_time < end_time)
);

CREATE TABLE IF NOT EXISTS fyp_fuli_leave_applications (
    leave_id BIGSERIAL PRIMARY KEY,
    staff_id BIGINT NOT NULL REFERENCES fyp_fuli_staff(staff_id) ON DELETE CASCADE,
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
    CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled'))
);

CREATE TABLE IF NOT EXISTS fyp_fuli_recurring_schedules (
    recurring_schedule_id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES fyp_fuli_business_profiles(business_id) ON DELETE CASCADE,
    -- NULL => owner-managed (no assigned staff) — business_id is what scopes
    -- the series in that case, since there's no staff row to go through.
    staff_id BIGINT REFERENCES fyp_fuli_staff(staff_id) ON DELETE CASCADE,
    day VARCHAR(30) NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES fyp_fuli_users(user_id) ON DELETE SET NULL,
    CHECK (day IN ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday')),
    CHECK (start_time < end_time)
);

CREATE TABLE IF NOT EXISTS fyp_fuli_service_slots (
    service_slot_id BIGSERIAL PRIMARY KEY,
    staff_id BIGINT REFERENCES fyp_fuli_staff(staff_id) ON DELETE CASCADE,
    recurring_schedule_id BIGINT REFERENCES fyp_fuli_recurring_schedules(recurring_schedule_id) ON DELETE CASCADE,
    date DATE NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES fyp_fuli_users(user_id) ON DELETE SET NULL,
    CHECK (start_time < end_time)
);

CREATE TABLE IF NOT EXISTS fyp_fuli_service_slot_options (
    slot_option_id BIGSERIAL PRIMARY KEY,
    service_option_id BIGINT NOT NULL REFERENCES fyp_fuli_service_options(service_option_id) ON DELETE CASCADE,
    service_slot_id BIGINT NOT NULL REFERENCES fyp_fuli_service_slots(service_slot_id) ON DELETE CASCADE,
    deleted_at TIMESTAMPTZ,
    UNIQUE (service_option_id, service_slot_id)
);

CREATE SEQUENCE IF NOT EXISTS fyp_fuli_booking_group_seq START 1;

CREATE TABLE IF NOT EXISTS fyp_fuli_bookings (
    booking_id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES fyp_fuli_users(user_id) ON DELETE RESTRICT,
    slot_option_id BIGINT NOT NULL REFERENCES fyp_fuli_service_slot_options(slot_option_id) ON DELETE RESTRICT,
    booking_group_id BIGINT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    booking_type VARCHAR(30) NOT NULL,
    description TEXT,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at TIMESTAMPTZ,
    decided_by BIGINT REFERENCES fyp_fuli_users(user_id) ON DELETE SET NULL,
    CHECK (status IN ('accepted', 'rescheduled', 'cancelled', 'past', 'rejected', 'pending')),
    CHECK (booking_type IN ('online', 'walk_in'))
);

CREATE INDEX IF NOT EXISTS idx_business_working_hours_business_id ON fyp_fuli_business_working_hours(business_id);
CREATE INDEX IF NOT EXISTS idx_services_business_id ON fyp_fuli_services(business_id);
CREATE INDEX IF NOT EXISTS idx_service_options_service_id ON fyp_fuli_service_options(service_id);
CREATE INDEX IF NOT EXISTS idx_service_option_items_service_option_id ON fyp_fuli_service_option_items(service_option_id);
CREATE INDEX IF NOT EXISTS idx_staff_user_id ON fyp_fuli_staff(user_id);
CREATE INDEX IF NOT EXISTS idx_staff_business_id ON fyp_fuli_staff(business_id);
CREATE INDEX IF NOT EXISTS idx_staff_working_hours_staff_id ON fyp_fuli_staff_working_hours(staff_id);
CREATE INDEX IF NOT EXISTS idx_leave_applications_staff_id ON fyp_fuli_leave_applications(staff_id);
CREATE INDEX IF NOT EXISTS idx_service_slots_staff_id ON fyp_fuli_service_slots(staff_id);
CREATE INDEX IF NOT EXISTS idx_service_slots_recurring_schedule_id ON fyp_fuli_service_slots(recurring_schedule_id);
CREATE INDEX IF NOT EXISTS idx_service_slot_options_service_option_id ON fyp_fuli_service_slot_options(service_option_id);
CREATE INDEX IF NOT EXISTS idx_service_slot_options_service_slot_id ON fyp_fuli_service_slot_options(service_slot_id);
CREATE INDEX IF NOT EXISTS idx_recurring_schedules_staff_id ON fyp_fuli_recurring_schedules(staff_id);
CREATE INDEX IF NOT EXISTS idx_bookings_user_id ON fyp_fuli_bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_slot_option_id ON fyp_fuli_bookings(slot_option_id);
CREATE UNIQUE INDEX services_unique_name
ON fyp_fuli_services (business_id, LOWER(service_name))
WHERE deleted_at IS NULL;

-- effective_from is part of the key so a service can reuse an option name for
-- a later, unrelated option starting on a different date (e.g. a seasonal
-- special that recurs each year under the same name), while still preventing
-- two options from starting on the same date with the same name.
CREATE UNIQUE INDEX service_options_unique_name
ON fyp_fuli_service_options (service_id, LOWER(service_option_name), effective_from)
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX service_option_items_unique_name
ON fyp_fuli_service_option_items (service_option_id, LOWER(service_option_item_name))
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX uq_active_booking_per_slot
ON fyp_fuli_bookings(slot_option_id)
WHERE deleted_at IS NULL
  AND status NOT IN ('cancelled', 'rejected');