package repository

import (
	"context"
	"database/sql"
	"fmt"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"time"

	"github.com/lib/pq"
)

type serviceSlotRepo struct {
	DB *sql.DB
}

func NewServiceSlotRepo(db *sql.DB) interfaces.IServiceSlotRepo {
	return &serviceSlotRepo{DB: db}
}

func timeStr(t time.Time) string {
	return t.Format("15:04:05")
}

func staffArg(staffID *int64) any {
	if staffID == nil {
		return nil
	}
	return *staffID
}

func recurringScheduleArg(recurringScheduleID *int64) any {
	if recurringScheduleID == nil {
		return nil
	}
	return *recurringScheduleID
}

// businessScopeSQL returns an EXISTS clause that scopes a service slot (alias ss)
// to a business via its packages' parent service. Slots have no business_id and
// may have no staff, so scoping goes through the packages.
func businessScopeSQL(slotAlias, businessParam string) string {
	return fmt.Sprintf(`EXISTS (
		SELECT 1 FROM fyp_fuli_service_slot_options ssp
		JOIN fyp_fuli_service_options sp ON sp.service_option_id = ssp.service_option_id
		JOIN fyp_fuli_services s ON s.service_id = sp.service_id
		WHERE ssp.service_slot_id = %s.service_slot_id
			AND s.business_id = %s
			AND ssp.deleted_at IS NULL
	)`, slotAlias, businessParam)
}

// hasBookingSQL returns an EXISTS clause reporting whether a service slot
// (alias slotAlias) has any active booking, for use as a computed column.
func hasBookingSQL(slotAlias string) string {
	return fmt.Sprintf(`EXISTS (
		SELECT 1 FROM fyp_fuli_bookings b
		JOIN fyp_fuli_service_slot_options ssp ON b.slot_option_id = ssp.slot_option_id
		WHERE ssp.service_slot_id = %s.service_slot_id
			AND b.deleted_at IS NULL
			AND b.status NOT IN ('cancelled', 'rejected')
	)`, slotAlias)
}

func (r *serviceSlotRepo) InsertServiceSlot(ctx context.Context, tx *sql.Tx, p param.ServiceSlotParam) (int64, error) {
	var slotID int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO fyp_fuli_service_slots (staff_id, recurring_schedule_id, date, start_time, end_time, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING service_slot_id
	`, staffArg(p.StaffID), recurringScheduleArg(p.RecurringScheduleID), p.Date, timeStr(p.StartTime), timeStr(p.EndTime), p.CreatedBy).Scan(&slotID)
	return slotID, err
}

func (r *serviceSlotRepo) InsertServiceSlotOption(ctx context.Context, tx *sql.Tx, serviceSlotID int64, serviceOptionID int64) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO fyp_fuli_service_slot_options (service_option_id, service_slot_id)
		VALUES ($1, $2)
	`, serviceOptionID, serviceSlotID)
	return err
}

// InsertRecurringSchedule records one (staff-or-owner-managed, day) series —
// not per package, since a slot's bookable packages are attached separately
// via service_slot_options regardless of how many there are. staff_id is
// nullable (owner-managed), so business_id is what scopes the series in
// that case — see SoftDeleteRecurringSchedule.
func (r *serviceSlotRepo) InsertRecurringSchedule(ctx context.Context, tx *sql.Tx, p param.ServiceSlotParam, day string) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO fyp_fuli_recurring_schedules (business_id, staff_id, day, start_time, end_time, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING recurring_schedule_id
	`, p.BusinessID, staffArg(p.StaffID), day, timeStr(p.StartTime), timeStr(p.EndTime), p.CreatedBy).Scan(&id)
	return id, err
}

func (r *serviceSlotRepo) GetServiceSlotsByBusinessAndDate(ctx context.Context, businessID int64, date string, staffID *int64, serviceID *int64, unassignedOnly bool) ([]param.ServiceSlotParam, error) {
	query := `
		SELECT
			ss.service_slot_id,
			ss.staff_id,
			st.staff_name,
			ss.recurring_schedule_id,
			to_char(ss.date, 'YYYY-MM-DD'),
			ss.start_time,
			ss.end_time,
			ss.created_by,
			` + hasBookingSQL("ss") + ` AS has_booking
		FROM fyp_fuli_service_slots ss
		LEFT JOIN fyp_fuli_staff st ON st.staff_id = ss.staff_id
		WHERE ss.date = $2
			AND ss.deleted_at IS NULL
			AND ` + businessScopeSQL("ss", "$1")
	args := []any{businessID, date}
	if unassignedOnly {
		query += " AND ss.staff_id IS NULL"
	} else if staffID != nil {
		args = append(args, *staffID)
		query += fmt.Sprintf(" AND ss.staff_id = $%d", len(args))
	}
	if serviceID != nil {
		args = append(args, *serviceID)
		query += fmt.Sprintf(`
			AND EXISTS (
				SELECT 1 FROM fyp_fuli_service_slot_options ssp
				JOIN fyp_fuli_service_options sp ON sp.service_option_id = ssp.service_option_id
				WHERE ssp.service_slot_id = ss.service_slot_id
					AND sp.service_id = $%d
					AND ssp.deleted_at IS NULL
			)`, len(args))
	}
	query += " ORDER BY ss.start_time, ss.service_slot_id"

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []param.ServiceSlotParam
	for rows.Next() {
		slot, err := scanServiceSlot(rows)
		if err != nil {
			return nil, err
		}
		slots = append(slots, *slot)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range slots {
		packages, err := r.getSlotOptions(ctx, slots[i].ServiceSlotID)
		if err != nil {
			return nil, err
		}
		slots[i].Packages = packages
	}
	return slots, nil
}

func (r *serviceSlotRepo) GetServiceSlotByID(ctx context.Context, serviceSlotID int64, businessID int64) (*param.ServiceSlotParam, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT
			ss.service_slot_id,
			ss.staff_id,
			st.staff_name,
			ss.recurring_schedule_id,
			to_char(ss.date, 'YYYY-MM-DD'),
			ss.start_time,
			ss.end_time,
			ss.created_by,
			`+hasBookingSQL("ss")+` AS has_booking
		FROM fyp_fuli_service_slots ss
		LEFT JOIN fyp_fuli_staff st ON st.staff_id = ss.staff_id
		WHERE ss.service_slot_id = $1
			AND ss.deleted_at IS NULL
			AND `+businessScopeSQL("ss", "$2"), serviceSlotID, businessID)

	slot, err := scanServiceSlot(row)
	if err != nil {
		return nil, err
	}
	packages, err := r.getSlotOptions(ctx, slot.ServiceSlotID)
	if err != nil {
		return nil, err
	}
	slot.Packages = packages
	return slot, nil
}

func (r *serviceSlotRepo) getSlotOptions(ctx context.Context, serviceSlotID int64) ([]param.SlotTierParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT
			ssp.slot_option_id,
			sp.service_option_id,
			sp.service_option_name,
			s.service_id,
			s.service_name
		FROM fyp_fuli_service_slot_options ssp
		JOIN fyp_fuli_service_options sp ON sp.service_option_id = ssp.service_option_id
		JOIN fyp_fuli_services s ON s.service_id = sp.service_id
		WHERE ssp.service_slot_id = $1
			AND ssp.deleted_at IS NULL
		ORDER BY sp.service_option_id
	`, serviceSlotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packages []param.SlotTierParam
	for rows.Next() {
		var pkg param.SlotTierParam
		if err := rows.Scan(&pkg.SlotOptionID, &pkg.ServiceOptionID, &pkg.ServiceOptionName, &pkg.ServiceID, &pkg.ServiceName); err != nil {
			return nil, err
		}
		packages = append(packages, pkg)
	}
	return packages, rows.Err()
}

func (r *serviceSlotRepo) ReassignStaff(ctx context.Context, tx *sql.Tx, serviceSlotID int64, businessID int64, staffID *int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_service_slots ss
		SET staff_id = $1
		WHERE ss.service_slot_id = $2
			AND ss.deleted_at IS NULL
			AND `+businessScopeSQL("ss", "$3"), staffArg(staffID), serviceSlotID, businessID)
	return err
}

// HasBookingForServiceSlot reports whether a slot has any booking still
// awaiting an outcome — pending counts as "has a booking" here (not just
// accepted/rescheduled): a customer is waiting on a decision, so deleting the
// slot out from under them should be blocked/retained, not silently deleted
// with their request auto-rejected.
// HasBookingForServiceSlot reports whether serviceSlotID has a genuinely
// active booking — past, cancelled, and rejected bookings don't block
// deleting the slot.
func (r *serviceSlotRepo) HasBookingForServiceSlot(ctx context.Context, serviceSlotID int64) (bool, error) {
	var count int
	err := r.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM fyp_fuli_bookings b
		JOIN fyp_fuli_service_slot_options ssp ON b.slot_option_id = ssp.slot_option_id
		WHERE ssp.service_slot_id = $1
			AND b.deleted_at IS NULL
			AND b.status IN ('pending', 'accepted', 'rescheduled')
	`, serviceSlotID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *serviceSlotRepo) SoftDeleteServiceSlot(ctx context.Context, tx *sql.Tx, serviceSlotID int64, businessID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_service_slots ss
		SET deleted_at = NOW()
		WHERE ss.service_slot_id = $1
			AND ss.deleted_at IS NULL
			AND `+businessScopeSQL("ss", "$2"), serviceSlotID, businessID)
	return err
}

func (r *serviceSlotRepo) SoftDeleteServiceSlots(ctx context.Context, tx *sql.Tx, slotIDs []int64, businessID int64) error {
	if len(slotIDs) == 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_service_slots ss
		SET deleted_at = NOW()
		WHERE ss.service_slot_id = ANY($1)
			AND ss.deleted_at IS NULL
			AND `+businessScopeSQL("ss", "$2"), pq.Array(slotIDs), businessID)
	return err
}

// GetFutureRecurringSlotIDs finds every non-deleted occurrence (from fromDate
// onward) of an exact recurring series, identified by its recurring_schedule_id
// — no more guessing by staff/time/weekday coincidence.
func (r *serviceSlotRepo) GetFutureRecurringSlotIDs(ctx context.Context, businessID int64, recurringScheduleID int64, fromDate string) ([]int64, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT ss.service_slot_id
		FROM fyp_fuli_service_slots ss
		WHERE ss.recurring_schedule_id = $2
			AND ss.date >= $3::date
			AND ss.deleted_at IS NULL
			AND `+businessScopeSQL("ss", "$1"), businessID, recurringScheduleID, fromDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetSlotDates returns the ("YYYY-MM-DD") dates of the given slots, sorted.
func (r *serviceSlotRepo) GetSlotDates(ctx context.Context, slotIDs []int64) ([]string, error) {
	if len(slotIDs) == 0 {
		return nil, nil
	}
	rows, err := r.DB.QueryContext(ctx, `
		SELECT to_char(date, 'YYYY-MM-DD')
		FROM fyp_fuli_service_slots
		WHERE service_slot_id = ANY($1)
		ORDER BY date
	`, pq.Array(slotIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		dates = append(dates, d)
	}
	return dates, rows.Err()
}

func (r *serviceSlotRepo) SoftDeleteRecurringSchedule(ctx context.Context, tx *sql.Tx, businessID int64, recurringScheduleID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_recurring_schedules rs
		SET deleted_at = NOW()
		WHERE rs.business_id = $1
			AND rs.recurring_schedule_id = $2
			AND rs.deleted_at IS NULL
	`, businessID, recurringScheduleID)
	return err
}

func (r *serviceSlotRepo) StaffBelongsToBusiness(ctx context.Context, staffID int64, businessID int64) (bool, error) {
	var exists bool
	err := r.DB.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM fyp_fuli_staff WHERE staff_id = $1 AND business_id = $2 AND deleted_at IS NULL)
	`, staffID, businessID).Scan(&exists)
	return exists, err
}

func (r *serviceSlotRepo) OptionsBelongToBusiness(ctx context.Context, businessID int64, packageIDs []int64) (bool, error) {
	if len(packageIDs) == 0 {
		return false, nil
	}
	var count int
	err := r.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM fyp_fuli_service_options sp
		JOIN fyp_fuli_services s ON s.service_id = sp.service_id
		WHERE s.business_id = $1
			AND sp.service_option_id = ANY($2)
			AND sp.deleted_at IS NULL
	`, businessID, pq.Array(packageIDs)).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == len(packageIDs), nil
}

func (r *serviceSlotRepo) StaffCoversTime(ctx context.Context, staffID int64, weekday string, startTime, endTime time.Time) (bool, error) {
	var exists bool
	err := r.DB.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM fyp_fuli_staff_working_hours
			WHERE staff_id = $1
				AND day = $2
				AND start_time <= $3
				AND end_time >= $4
				AND deleted_at IS NULL
		)
	`, staffID, weekday, timeStr(startTime), timeStr(endTime)).Scan(&exists)
	return exists, err
}

// GetFutureUnassignedSlotWindows returns the (date, start, end) of every
// non-deleted owner-managed (staff_id IS NULL) slot for a business from
// fromDate onward — used to check a business working-hours edit doesn't
// strand an existing slot outside the new hours.
func (r *serviceSlotRepo) GetFutureUnassignedSlotWindows(ctx context.Context, businessID int64, fromDate string) ([]param.SlotWindowParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT to_char(ss.date, 'YYYY-MM-DD'), ss.start_time, ss.end_time
		FROM fyp_fuli_service_slots ss
		WHERE ss.staff_id IS NULL
			AND ss.date >= $2::date
			AND ss.deleted_at IS NULL
			AND `+businessScopeSQL("ss", "$1"), businessID, fromDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var windows []param.SlotWindowParam
	for rows.Next() {
		var w param.SlotWindowParam
		if err := rows.Scan(&w.Date, &w.StartTime, &w.EndTime); err != nil {
			return nil, err
		}
		windows = append(windows, w)
	}
	return windows, rows.Err()
}

func (r *serviceSlotRepo) queryAssignedSlots(ctx context.Context, staffID int64, dateCond string, dateArgs ...any) ([]param.AssignedSlotParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT ss.service_slot_id, to_char(ss.date, 'YYYY-MM-DD'), ss.start_time, ss.end_time, `+hasBookingSQL("ss")+`
		FROM fyp_fuli_service_slots ss
		WHERE ss.staff_id = $1
			AND `+dateCond+`
			AND ss.deleted_at IS NULL
	`, append([]any{staffID}, dateArgs...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []param.AssignedSlotParam
	for rows.Next() {
		var s param.AssignedSlotParam
		if err := rows.Scan(&s.ServiceSlotID, &s.Date, &s.StartTime, &s.EndTime, &s.HasBooking); err != nil {
			return nil, err
		}
		slots = append(slots, s)
	}
	return slots, rows.Err()
}

// GetAssignedSlotsInRange returns a staff's non-deleted assigned slots whose
// date falls within [fromDate, toDate] — used to find every slot a leave
// application would affect.
func (r *serviceSlotRepo) GetAssignedSlotsInRange(ctx context.Context, staffID int64, fromDate, toDate string) ([]param.AssignedSlotParam, error) {
	return r.queryAssignedSlots(ctx, staffID, "ss.date >= $2::date AND ss.date <= $3::date", fromDate, toDate)
}

// GetFutureAssignedSlotWindows returns a staff's non-deleted assigned slots
// from fromDate onward — used to check a working-hours edit against them.
func (r *serviceSlotRepo) GetFutureAssignedSlotWindows(ctx context.Context, staffID int64, fromDate string) ([]param.AssignedSlotParam, error) {
	return r.queryAssignedSlots(ctx, staffID, "ss.date >= $2::date", fromDate)
}

func (r *serviceSlotRepo) BusinessCoversTime(ctx context.Context, businessID int64, weekday string, startTime, endTime time.Time) (bool, error) {
	var exists bool
	err := r.DB.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM fyp_fuli_business_working_hours
			WHERE business_id = $1
				AND day = $2
				AND start_time <= $3
				AND end_time >= $4
				AND deleted_at IS NULL
		)
	`, businessID, weekday, timeStr(startTime), timeStr(endTime)).Scan(&exists)
	return exists, err
}

func (r *serviceSlotRepo) GetAvailableStaff(ctx context.Context, businessID int64, date string, weekday string, startTime, endTime time.Time, excludeSlotID int64) ([]param.StaffParam, error) {
	rows, err := r.DB.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			st.staff_id,
			st.user_id,
			st.business_id,
			st.staff_name,
			u.email,
			u.must_reset_password,
			st.staff_contact_number,
			st.position,
			%s
		FROM fyp_fuli_staff st
		JOIN fyp_fuli_users u ON u.user_id = st.user_id
		WHERE st.business_id = $1
			AND st.deleted_at IS NULL
			AND EXISTS (
				SELECT 1 FROM fyp_fuli_staff_working_hours swh
				WHERE swh.staff_id = st.staff_id
					AND swh.day = $2
					AND swh.start_time <= $3
					AND swh.end_time >= $4
					AND swh.deleted_at IS NULL
			)
			AND NOT EXISTS (
				SELECT 1 FROM fyp_fuli_service_slots ss
				WHERE ss.staff_id = st.staff_id
					AND ss.date = $5
					AND ss.start_time < $4
					AND ss.end_time > $3
					AND ss.service_slot_id <> $6
					AND ss.deleted_at IS NULL
			)
		ORDER BY st.staff_id
	`, staffHasBookingSQL("st")), businessID, weekday, timeStr(startTime), timeStr(endTime), date, excludeSlotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var staffList []param.StaffParam
	for rows.Next() {
		staff, err := scanStaffRow(rows)
		if err != nil {
			return nil, err
		}
		staffList = append(staffList, *staff)
	}
	return staffList, rows.Err()
}

func scanServiceSlot(row rowScannerService) (*param.ServiceSlotParam, error) {
	var slot param.ServiceSlotParam
	var staffID sql.NullInt64
	var staffName sql.NullString
	var recurringScheduleID sql.NullInt64
	var createdBy sql.NullInt64
	if err := row.Scan(
		&slot.ServiceSlotID,
		&staffID,
		&staffName,
		&recurringScheduleID,
		&slot.Date,
		&slot.StartTime,
		&slot.EndTime,
		&createdBy,
		&slot.HasBooking,
	); err != nil {
		return nil, err
	}
	if staffID.Valid {
		id := staffID.Int64
		slot.StaffID = &id
	}
	if staffName.Valid {
		slot.StaffName = staffName.String
	}
	if recurringScheduleID.Valid {
		id := recurringScheduleID.Int64
		slot.RecurringScheduleID = &id
	}
	if createdBy.Valid {
		slot.CreatedBy = createdBy.Int64
	}
	return &slot, nil
}
