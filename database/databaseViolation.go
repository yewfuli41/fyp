package database

import (
	"errors"

	"github.com/lib/pq"
)

// Constraint names as Postgres actually reports them, which is what the
// checks below compare against. Ones Postgres names itself follow
// <table>_<column>_key / _fkey, so they carry the fyp_fuli_ table prefix —
// leaving it off silently turns every check into "no match", and the raw
// driver error reaches the user instead of a validation message. Naming them
// once here keeps production code and its tests on the same string.
const (
	ConstraintUserEmail             = "fyp_fuli_users_email_key"
	ConstraintStaffUserID           = "fyp_fuli_staff_user_id_key"
	ConstraintBusinessOwner         = "fyp_fuli_business_profiles_owner_user_id_key"
	ConstraintBusinessOwnerRef      = "fyp_fuli_business_profiles_owner_user_id_fkey"
	ConstraintServiceName           = "services_unique_name"
	ConstraintServiceOptionName     = "service_options_unique_name"
	ConstraintServiceOptionItemName = "service_option_items_unique_name"
	ConstraintActiveBookingPerSlot  = "uq_active_booking_per_slot"
)

func IsForeignKeyViolation(err error, constraint string) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23503" && pqErr.Constraint == constraint
}

func IsUniqueViolation(err error, constraint string) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505" && pqErr.Constraint == constraint
}

func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "40001" || pqErr.Code == "40P01"
	}

	return false
}
