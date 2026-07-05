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
		SELECT 1 FROM service_slot_packages ssp
		JOIN service_packages sp ON sp.service_package_id = ssp.service_package_id
		JOIN services s ON s.service_id = sp.service_id
		WHERE ssp.service_slot_id = %s.service_slot_id
			AND s.business_id = %s
			AND ssp.deleted_at IS NULL
	)`, slotAlias, businessParam)
}

// hasBookingSQL returns an EXISTS clause reporting whether a service slot
// (alias slotAlias) has any active booking, for use as a computed column.
func hasBookingSQL(slotAlias string) string {
	return fmt.Sprintf(`EXISTS (
		SELECT 1 FROM bookings b
		JOIN service_slot_packages ssp ON b.slot_package_id = ssp.slot_package_id
		WHERE ssp.service_slot_id = %s.service_slot_id
			AND b.deleted_at IS NULL
	)`, slotAlias)
}

func (r *serviceSlotRepo) InsertServiceSlot(ctx context.Context, tx *sql.Tx, p param.ServiceSlotParam) (int64, error) {
	var slotID int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO service_slots (staff_id, recurring_schedule_id, date, start_time, end_time, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING service_slot_id
	`, staffArg(p.StaffID), recurringScheduleArg(p.RecurringScheduleID), p.Date, timeStr(p.StartTime), timeStr(p.EndTime), p.CreatedBy).Scan(&slotID)
	return slotID, err
}

func (r *serviceSlotRepo) InsertServiceSlotPackage(ctx context.Context, tx *sql.Tx, serviceSlotID int64, servicePackageID int64) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO service_slot_packages (service_package_id, service_slot_id)
		VALUES ($1, $2)
	`, servicePackageID, serviceSlotID)
	return err
}

// InsertRecurringSchedule records one (staff, day) series — not per package,
// since a slot's bookable packages are attached separately via
// service_slot_packages regardless of how many there are.
func (r *serviceSlotRepo) InsertRecurringSchedule(ctx context.Context, tx *sql.Tx, p param.ServiceSlotParam, day string) (int64, error) {
	// recurring_schedules.staff_id is NOT NULL, so only staff-assigned slots are recorded here.
	if p.StaffID == nil {
		return 0, nil
	}
	var id int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO recurring_schedules (staff_id, day, start_time, end_time, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING recurring_schedule_id
	`, *p.StaffID, day, timeStr(p.StartTime), timeStr(p.EndTime), p.CreatedBy).Scan(&id)
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
		FROM service_slots ss
		LEFT JOIN staff st ON st.staff_id = ss.staff_id
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
				SELECT 1 FROM service_slot_packages ssp
				JOIN service_packages sp ON sp.service_package_id = ssp.service_package_id
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
		packages, err := r.getSlotPackages(ctx, slots[i].ServiceSlotID)
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
		FROM service_slots ss
		LEFT JOIN staff st ON st.staff_id = ss.staff_id
		WHERE ss.service_slot_id = $1
			AND ss.deleted_at IS NULL
			AND `+businessScopeSQL("ss", "$2"), serviceSlotID, businessID)

	slot, err := scanServiceSlot(row)
	if err != nil {
		return nil, err
	}
	packages, err := r.getSlotPackages(ctx, slot.ServiceSlotID)
	if err != nil {
		return nil, err
	}
	slot.Packages = packages
	return slot, nil
}

func (r *serviceSlotRepo) getSlotPackages(ctx context.Context, serviceSlotID int64) ([]param.SlotPackageParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT
			ssp.slot_package_id,
			sp.service_package_id,
			sp.service_package_name,
			s.service_id,
			s.service_name
		FROM service_slot_packages ssp
		JOIN service_packages sp ON sp.service_package_id = ssp.service_package_id
		JOIN services s ON s.service_id = sp.service_id
		WHERE ssp.service_slot_id = $1
			AND ssp.deleted_at IS NULL
		ORDER BY ssp.slot_package_id
	`, serviceSlotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packages []param.SlotPackageParam
	for rows.Next() {
		var pkg param.SlotPackageParam
		if err := rows.Scan(&pkg.SlotPackageID, &pkg.ServicePackageID, &pkg.ServicePackageName, &pkg.ServiceID, &pkg.ServiceName); err != nil {
			return nil, err
		}
		packages = append(packages, pkg)
	}
	return packages, rows.Err()
}

func (r *serviceSlotRepo) ReassignStaff(ctx context.Context, tx *sql.Tx, serviceSlotID int64, businessID int64, staffID *int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE service_slots ss
		SET staff_id = $1
		WHERE ss.service_slot_id = $2
			AND ss.deleted_at IS NULL
			AND `+businessScopeSQL("ss", "$3"), staffArg(staffID), serviceSlotID, businessID)
	return err
}

func (r *serviceSlotRepo) HasBookingForServiceSlot(ctx context.Context, serviceSlotID int64) (bool, error) {
	var count int
	err := r.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM bookings b
		JOIN service_slot_packages ssp ON b.slot_package_id = ssp.slot_package_id
		WHERE ssp.service_slot_id = $1 AND b.deleted_at IS NULL
	`, serviceSlotID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *serviceSlotRepo) SoftDeleteServiceSlot(ctx context.Context, tx *sql.Tx, serviceSlotID int64, businessID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE service_slots ss
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
		UPDATE service_slots ss
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
		FROM service_slots ss
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

func (r *serviceSlotRepo) SoftDeleteRecurringSchedule(ctx context.Context, tx *sql.Tx, businessID int64, recurringScheduleID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE recurring_schedules rs
		SET deleted_at = NOW()
		FROM staff st
		WHERE rs.staff_id = st.staff_id
			AND st.business_id = $1
			AND rs.recurring_schedule_id = $2
			AND rs.deleted_at IS NULL
	`, businessID, recurringScheduleID)
	return err
}

func (r *serviceSlotRepo) StaffBelongsToBusiness(ctx context.Context, staffID int64, businessID int64) (bool, error) {
	var exists bool
	err := r.DB.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM staff WHERE staff_id = $1 AND business_id = $2 AND deleted_at IS NULL)
	`, staffID, businessID).Scan(&exists)
	return exists, err
}

func (r *serviceSlotRepo) PackagesBelongToBusiness(ctx context.Context, businessID int64, packageIDs []int64) (bool, error) {
	if len(packageIDs) == 0 {
		return false, nil
	}
	var count int
	err := r.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM service_packages sp
		JOIN services s ON s.service_id = sp.service_id
		WHERE s.business_id = $1
			AND sp.service_package_id = ANY($2)
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
			SELECT 1 FROM staff_working_hours
			WHERE staff_id = $1
				AND day = $2
				AND start_time <= $3
				AND end_time >= $4
				AND deleted_at IS NULL
		)
	`, staffID, weekday, timeStr(startTime), timeStr(endTime)).Scan(&exists)
	return exists, err
}

func (r *serviceSlotRepo) GetAvailableStaff(ctx context.Context, businessID int64, date string, weekday string, startTime, endTime time.Time, excludeSlotID int64) ([]param.StaffParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT
			st.staff_id,
			st.user_id,
			st.business_id,
			st.staff_name,
			u.email,
			u.must_reset_password,
			st.staff_contact_number,
			st.position
		FROM staff st
		JOIN users u ON u.user_id = st.user_id
		WHERE st.business_id = $1
			AND st.deleted_at IS NULL
			AND EXISTS (
				SELECT 1 FROM staff_working_hours swh
				WHERE swh.staff_id = st.staff_id
					AND swh.day = $2
					AND swh.start_time <= $3
					AND swh.end_time >= $4
					AND swh.deleted_at IS NULL
			)
			AND NOT EXISTS (
				SELECT 1 FROM service_slots ss
				WHERE ss.staff_id = st.staff_id
					AND ss.date = $5
					AND ss.start_time < $4
					AND ss.end_time > $3
					AND ss.service_slot_id <> $6
					AND ss.deleted_at IS NULL
			)
		ORDER BY st.staff_id
	`, businessID, weekday, timeStr(startTime), timeStr(endTime), date, excludeSlotID)
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
