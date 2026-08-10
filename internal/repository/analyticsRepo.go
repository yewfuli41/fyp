package repository

import (
	"context"
	"database/sql"
	"fmt"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"sort"
	"strconv"
	"time"

	"github.com/lib/pq"
)

type analyticsRepo struct {
	DB *sql.DB
}

func NewAnalyticsRepo(db *sql.DB) interfaces.IAnalyticsRepo {
	return &analyticsRepo{DB: db}
}

// bookingBusinessJoin is the shared join path from bookings to the business
// that owns them, plus the appointment slot — reused by every widget below.
// Mirrors bookingDetailSelect's join shape in bookingRepo.go.
const bookingBusinessJoin = `
	FROM bookings b
	JOIN service_slot_options sso ON b.slot_option_id = sso.slot_option_id
	JOIN service_options so ON so.service_option_id = sso.service_option_id
	JOIN services s ON s.service_id = so.service_id
	JOIN service_slots ss ON ss.service_slot_id = sso.service_slot_id
	WHERE s.business_id = $1
		AND b.deleted_at IS NULL
		AND ss.date BETWEEN $2 AND $3
`

// bookingBusinessJoinNoRange is bookingBusinessJoin without the date-range
// filter — for the two BookingSummary figures that are always live
// ("today", "all pending") rather than scoped to the selected range.
const bookingBusinessJoinNoRange = `
	FROM bookings b
	JOIN service_slot_options sso ON b.slot_option_id = sso.slot_option_id
	JOIN service_options so ON so.service_option_id = sso.service_option_id
	JOIN services s ON s.service_id = so.service_id
	JOIN service_slots ss ON ss.service_slot_id = sso.service_slot_id
	WHERE s.business_id = $1
		AND b.deleted_at IS NULL
`

func (r *analyticsRepo) GetBookingSummary(ctx context.Context, businessID int64, from, until string) (*param.BookingSummaryParam, error) {
	summary := &param.BookingSummaryParam{}

	statusRows, err := r.DB.QueryContext(ctx, `
		SELECT b.status, COUNT(*)
	`+bookingBusinessJoin+`
		GROUP BY b.status
	`, businessID, from, until)
	if err != nil {
		return nil, err
	}
	defer statusRows.Close()
	for statusRows.Next() {
		var sc param.StatusCountParam
		if err := statusRows.Scan(&sc.Status, &sc.Count); err != nil {
			return nil, err
		}
		summary.ByStatus = append(summary.ByStatus, sc)
		summary.TotalBookings += sc.Count
	}
	if err := statusRows.Err(); err != nil {
		return nil, err
	}

	typeRows, err := r.DB.QueryContext(ctx, `
		SELECT b.booking_type, COUNT(*)
	`+bookingBusinessJoin+`
		GROUP BY b.booking_type
	`, businessID, from, until)
	if err != nil {
		return nil, err
	}
	defer typeRows.Close()
	for typeRows.Next() {
		var tc param.TypeCountParam
		if err := typeRows.Scan(&tc.BookingType, &tc.Count); err != nil {
			return nil, err
		}
		summary.ByType = append(summary.ByType, tc)
	}
	if err := typeRows.Err(); err != nil {
		return nil, err
	}

	err = r.DB.QueryRowContext(ctx, `
		SELECT COUNT(*)
	`+bookingBusinessJoinNoRange+`
			AND b.status = 'accepted'
			AND ss.date = CURRENT_DATE
	`, businessID).Scan(&summary.TodayAcceptedCount)
	if err != nil {
		return nil, err
	}

	err = r.DB.QueryRowContext(ctx, `
		SELECT COUNT(*)
	`+bookingBusinessJoinNoRange+`
			AND b.status = 'pending'
	`, businessID).Scan(&summary.PendingCount)
	if err != nil {
		return nil, err
	}

	return summary, nil
}

func (r *analyticsRepo) GetBookingTrend(ctx context.Context, businessID int64, from, until, granularity string) ([]param.BookingTrendPointParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT to_char(date_trunc($4, ss.date::timestamp), 'YYYY-MM-DD'), COUNT(*)
	`+bookingBusinessJoin+`
		GROUP BY 1
		ORDER BY 1
	`, businessID, from, until, granularity)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []param.BookingTrendPointParam
	for rows.Next() {
		var p param.BookingTrendPointParam
		if err := rows.Scan(&p.Date, &p.Count); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// GetServicePopularity groups by service_name rather than service_id: a
// service that was deleted and recreated (services are soft-deleted, see
// services.deleted_at) gets a new service_id but keeps the same name, and
// should still count as one line in the popularity chart rather than
// fragmenting across the old and new rows.
func (r *analyticsRepo) GetServicePopularity(ctx context.Context, businessID int64, from, until string) ([]param.ServicePopularityParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT MIN(s.service_id), s.service_name, COUNT(*)
	`+bookingBusinessJoin+`
		GROUP BY s.service_name
		ORDER BY COUNT(*) DESC
	`, businessID, from, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []param.ServicePopularityParam
	for rows.Next() {
		var p param.ServicePopularityParam
		if err := rows.Scan(&p.ServiceID, &p.ServiceName, &p.BookingCount); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

// slotUtilRow is one service_slot in range, with enough to bucket it by
// date/weekday and by staff — shared by GetSlotUtilization and
// GetStaffUtilization so the (identical) underlying query only runs once
// per caller.
type slotUtilRow struct {
	staffID   sql.NullInt64
	staffName sql.NullString
	date      string // "YYYY-MM-DD"
	dow       int
	startTime time.Time
	endTime   time.Time
	booked    bool
}

// hours is this slot's duration — startTime/endTime are TIME columns, so
// both come back on the same zero date and a plain Sub() is safe.
func (r slotUtilRow) hours() float64 {
	return r.endTime.Sub(r.startTime).Hours()
}

// getSlotUtilRows returns every service_slot for businessID in [from, until],
// each tagged with whether it has an active-or-past booking (mirrors
// hasBookingSQL in serviceSlotRepo.go: a completed appointment still counts
// as "utilized", only cancelled/rejected slots don't).
func (r *analyticsRepo) getSlotUtilRows(ctx context.Context, businessID int64, from, until string) ([]slotUtilRow, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT
			ss.staff_id,
			st.staff_name,
			to_char(ss.date, 'YYYY-MM-DD'),
			EXTRACT(DOW FROM ss.date)::int,
			ss.start_time,
			ss.end_time,
			EXISTS (
				SELECT 1 FROM bookings b
				JOIN service_slot_options sso2 ON b.slot_option_id = sso2.slot_option_id
				WHERE sso2.service_slot_id = ss.service_slot_id
					AND b.deleted_at IS NULL
					AND b.status NOT IN ('cancelled', 'rejected')
			)
		FROM service_slots ss
		LEFT JOIN staff st ON st.staff_id = ss.staff_id
		WHERE ss.deleted_at IS NULL
			AND ss.date BETWEEN $2 AND $3
			AND EXISTS (
				SELECT 1 FROM service_slot_options ssp
				JOIN service_options so ON so.service_option_id = ssp.service_option_id
				JOIN services s ON s.service_id = so.service_id
				WHERE ssp.service_slot_id = ss.service_slot_id
					AND s.business_id = $1
					AND ssp.deleted_at IS NULL
			)
	`, businessID, from, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []slotUtilRow
	for rows.Next() {
		var row slotUtilRow
		if err := rows.Scan(&row.staffID, &row.staffName, &row.date, &row.dow, &row.startTime, &row.endTime, &row.booked); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// fullWeekdayLabels is every weekday, Monday-first.
func fullWeekdayLabels() []string {
	return []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
}

// allMonths is "01".."12".
func allMonths() []string {
	months := make([]string, 12)
	for i := range months {
		months[i] = fmt.Sprintf("%02d", i+1)
	}
	return months
}

// daysInMonth is "01".."NN" for the given year/month (28-31 depending).
func daysInMonth(year, month int) []string {
	lastDay := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	days := make([]string, lastDay)
	for d := 1; d <= lastDay; d++ {
		days[d-1] = fmt.Sprintf("%02d", d)
	}
	return days
}

// yearLabels is every year from startYear through endYear, inclusive
// (endYear clamped up to startYear if it'd otherwise be earlier).
func yearLabels(startYear, endYear int) []string {
	if endYear < startYear {
		endYear = startYear
	}
	labels := make([]string, 0, endYear-startYear+1)
	for y := startYear; y <= endYear; y++ {
		labels = append(labels, strconv.Itoa(y))
	}
	return labels
}

type yearMonth struct {
	year  int
	month int
}

// monthsInRange is every (year, month) from start's month through end's
// month, inclusive (clamped up to start's month if end would otherwise
// precede it).
func monthsInRange(start, end time.Time) []yearMonth {
	d := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	endMonth := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	if endMonth.Before(d) {
		endMonth = d
	}
	var months []yearMonth
	for !d.After(endMonth) {
		months = append(months, yearMonth{year: d.Year(), month: int(d.Month())})
		d = d.AddDate(0, 1, 0)
	}
	return months
}

// bucketKey is one slot's (section, point) coordinates under the requested
// grouping — "date" sections by month ("2026-07") with a day-of-month point
// ("15"); "month" sections by year ("2026") with a month point ("07");
// "weekday"/"year" have no natural section, so sectionKey is always "".
func bucketKey(row slotUtilRow, groupBy string) (sectionKey, pointLabel string) {
	switch groupBy {
	case "date":
		return row.date[:7], row.date[8:10]
	case "month":
		return row.date[:4], row.date[5:7]
	case "year":
		return "", row.date[:4]
	default: // "weekday"
		return "", weekdayFromDOW(row.dow)
	}
}

// buildSection merges the actually-observed points (keyed by label) into the
// full expected point set for a section, defaulting anything missing to
// zero — so e.g. a Sunday with no slots still shows up, empty, rather than
// silently vanishing from the breakdown. Any observed label outside the
// expected set (shouldn't normally happen) is still kept, appended at the end,
// rather than dropping real data.
func buildSection(subtitle string, fullPoints []string, actual map[string]*param.UtilizationBreakdownParam) param.UtilizationSectionParam {
	points := make([]param.UtilizationBreakdownParam, 0, len(fullPoints))
	seen := make(map[string]bool, len(fullPoints))
	for _, label := range fullPoints {
		if p, ok := actual[label]; ok {
			points = append(points, *p)
		} else {
			points = append(points, param.UtilizationBreakdownParam{Label: label})
		}
		seen[label] = true
	}
	var extra []string
	for label := range actual {
		if !seen[label] {
			extra = append(extra, label)
		}
	}
	sort.Strings(extra)
	for _, label := range extra {
		points = append(points, *actual[label])
	}
	return param.UtilizationSectionParam{Subtitle: subtitle, Points: points}
}

// GetBusinessCreatedAt returns businessID's creation date ("YYYY-MM-DD") —
// used both to bound GetSlotUtilization's date/month/year groupings and, on
// the frontend, to bound the dashboard's own Years filter to years the
// business could actually have data in.
func (r *analyticsRepo) GetBusinessCreatedAt(ctx context.Context, businessID int64) (string, error) {
	var createdAt string
	err := r.DB.QueryRowContext(ctx, `
		SELECT to_char(created_at, 'YYYY-MM-DD') FROM business_profiles WHERE business_id = $1
	`, businessID).Scan(&createdAt)
	return createdAt, err
}

// GetSlotUtilization buckets every slot by the requested grouping — weekday,
// date, month, or year — a single dimension, not a weekday×hour grid, since
// which hour a slot starts isn't itself something a business owner filters
// by. Every possible bucket is included even when empty (e.g. a Sunday, or a
// day before the business opened, with zero slots), so the breakdown always
// reads as a complete calendar rather than silently omitting gaps. Unlike
// "weekday" (which stays scoped to the selected top-level range), "date",
// "month", and "year" all span the business's whole history — from its own
// creation date through the selected range's end — since a business owner
// picking one of those wants the full calendar, not a window that happens to
// match whatever the top-level filter is currently set to. "date" and
// "month" come back sectioned (by month, and by year, respectively) so a
// long history doesn't read as one giant undifferentiated strip.
func (r *analyticsRepo) GetSlotUtilization(ctx context.Context, businessID int64, from, until, groupBy string) (*param.SlotUtilizationParam, error) {
	effectiveFrom := from
	if groupBy != "weekday" {
		createdAt, err := r.GetBusinessCreatedAt(ctx, businessID)
		if err != nil {
			return nil, err
		}
		effectiveFrom = createdAt
	}

	rows, err := r.getSlotUtilRows(ctx, businessID, effectiveFrom, until)
	if err != nil {
		return nil, err
	}

	result := &param.SlotUtilizationParam{}
	sectionData := make(map[string]map[string]*param.UtilizationBreakdownParam)
	for _, row := range rows {
		result.TotalSlots++
		if row.booked {
			result.BookedSlots++
		}

		sectionKey, pointLabel := bucketKey(row, groupBy)
		points, ok := sectionData[sectionKey]
		if !ok {
			points = make(map[string]*param.UtilizationBreakdownParam)
			sectionData[sectionKey] = points
		}
		point, ok := points[pointLabel]
		if !ok {
			point = &param.UtilizationBreakdownParam{Label: pointLabel}
			points[pointLabel] = point
		}
		point.TotalSlots++
		if row.booked {
			point.BookedSlots++
		}
	}

	start, errStart := time.Parse("2006-01-02", effectiveFrom)
	end, errEnd := time.Parse("2006-01-02", until)
	if errStart != nil || errEnd != nil {
		return nil, fmt.Errorf("invalid date range %q..%q", effectiveFrom, until)
	}

	switch groupBy {
	case "date":
		for _, ym := range monthsInRange(start, end) {
			subtitle := fmt.Sprintf("%04d-%02d", ym.year, ym.month)
			result.Sections = append(result.Sections, buildSection(subtitle, daysInMonth(ym.year, ym.month), sectionData[subtitle]))
		}
	case "month":
		for _, y := range yearLabels(start.Year(), end.Year()) {
			result.Sections = append(result.Sections, buildSection(y, allMonths(), sectionData[y]))
		}
	case "year":
		result.Sections = append(result.Sections, buildSection("", yearLabels(start.Year(), end.Year()), sectionData[""]))
	default: // "weekday"
		result.Sections = append(result.Sections, buildSection("", fullWeekdayLabels(), sectionData[""]))
	}
	return result, nil
}

// GetStaffUtilization includes an "Owner-managed" bucket (StaffID nil) for
// slots with no assigned staff, alongside every real staff member's own bucket.
func (r *analyticsRepo) GetStaffUtilization(ctx context.Context, businessID int64, from, until string) ([]param.StaffUtilizationParam, error) {
	rows, err := r.getSlotUtilRows(ctx, businessID, from, until)
	if err != nil {
		return nil, err
	}

	type staffKey struct {
		id    int64
		valid bool
	}
	byStaff := make(map[staffKey]*param.StaffUtilizationParam)
	var order []staffKey
	for _, row := range rows {
		key := staffKey{}
		name := "Owner-managed"
		var id *int64
		if row.staffID.Valid {
			key = staffKey{id: row.staffID.Int64, valid: true}
			name = row.staffName.String
			v := row.staffID.Int64
			id = &v
		}
		staff, ok := byStaff[key]
		if !ok {
			staff = &param.StaffUtilizationParam{StaffID: id, StaffName: name}
			byStaff[key] = staff
			order = append(order, key)
		}
		if row.booked {
			staff.BookedSlots++
			staff.BookedHours += row.hours()
		}
	}

	results := make([]param.StaffUtilizationParam, 0, len(order))
	for _, key := range order {
		results = append(results, *byStaff[key])
	}
	return results, nil
}

// weekdayFromDOW converts Postgres's EXTRACT(DOW) (0=Sunday..6=Saturday) to
// the lowercase weekday names used by the DayOfWeek enum elsewhere.
func weekdayFromDOW(dow int) string {
	names := [7]string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}
	if dow < 0 || dow > 6 {
		return ""
	}
	return names[dow]
}

func (r *analyticsRepo) GetCustomerRetention(ctx context.Context, businessID int64, from, until string) (*param.CustomerRetentionParam, error) {
	result := &param.CustomerRetentionParam{}
	err := r.DB.QueryRowContext(ctx, `
		WITH customers AS (
			SELECT b.user_id, MIN(b.created_at) AS first_booking_at
			FROM bookings b
			JOIN service_slot_options sso ON b.slot_option_id = sso.slot_option_id
			JOIN service_options so ON so.service_option_id = sso.service_option_id
			JOIN services s ON s.service_id = so.service_id
			WHERE s.business_id = $1 AND b.deleted_at IS NULL AND b.booking_type != 'walk_in'
			GROUP BY b.user_id
		),
		active AS (
			SELECT DISTINCT b.user_id
			FROM bookings b
			JOIN service_slot_options sso ON b.slot_option_id = sso.slot_option_id
			JOIN service_options so ON so.service_option_id = sso.service_option_id
			JOIN services s ON s.service_id = so.service_id
			WHERE s.business_id = $1 AND b.deleted_at IS NULL AND b.booking_type != 'walk_in'
				AND b.created_at::date BETWEEN $2::date AND $3::date
		)
		SELECT
			COUNT(*) FILTER (WHERE c.first_booking_at::date >= $2::date),
			COUNT(*) FILTER (WHERE c.first_booking_at::date < $2::date)
		FROM active a
		JOIN customers c ON c.user_id = a.user_id
	`, businessID, from, until).Scan(&result.NewCustomers, &result.ReturningCustomers)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// cancellationFilterSQL appends the optional staffIds/serviceIds narrowing
// shared by every query in GetCancellationAnalysis, returning the extended
// query and args ready to pass to QueryContext/QueryRowContext. An empty or
// nil slice means "any" (no filter) for that dimension.
func cancellationFilterSQL(query string, args []any, staffIDs, serviceIDs []int64) (string, []any) {
	if len(staffIDs) > 0 {
		args = append(args, pq.Array(staffIDs))
		query += fmt.Sprintf(" AND ss.staff_id = ANY($%d)", len(args))
	}
	if len(serviceIDs) > 0 {
		args = append(args, pq.Array(serviceIDs))
		query += fmt.Sprintf(" AND s.service_id = ANY($%d)", len(args))
	}
	return query, args
}

// GetCancellationAnalysis combines cancelled and rejected throughout — both
// mean the same thing to the business (a booking that didn't happen) — and,
// for the byStaff breakdown, splits that combined count by who decided it.
// A customer can never reject their own booking (see bookingService's
// isBusinessSide guard on RejectBooking), so every rejection counts as
// staff-initiated; a cancellation counts as customer-initiated only when
// decided_by is the booking's own customer on a real (non-walk-in) booking
// — a walk-in's decided_by/user_id both belong to the staff/owner who
// recorded it, not a real customer.
func (r *analyticsRepo) GetCancellationAnalysis(ctx context.Context, businessID int64, from, until string, staffIDs, serviceIDs []int64) (*param.CancellationAnalysisParam, error) {
	result := &param.CancellationAnalysisParam{}

	totalsQuery, totalsArgs := cancellationFilterSQL(`
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE b.status = 'cancelled'),
			COUNT(*) FILTER (WHERE b.status = 'rejected')
	`+bookingBusinessJoin, []any{businessID, from, until}, staffIDs, serviceIDs)
	if err := r.DB.QueryRowContext(ctx, totalsQuery, totalsArgs...).
		Scan(&result.TotalBookings, &result.TotalCancelled, &result.TotalRejected); err != nil {
		return nil, err
	}

	byServiceQuery, byServiceArgs := cancellationFilterSQL(`
		SELECT MIN(s.service_id), s.service_name, COUNT(*)
	`+bookingBusinessJoin+`
			AND b.status IN ('cancelled', 'rejected')
	`, []any{businessID, from, until}, staffIDs, serviceIDs)
	// Grouped by name, not service_id — see GetServicePopularity's comment on
	// why a soft-deleted-and-recreated service shouldn't fragment its count.
	byServiceQuery += " GROUP BY s.service_name ORDER BY COUNT(*) DESC"
	rows, err := r.DB.QueryContext(ctx, byServiceQuery, byServiceArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var sc param.ServiceCancellationParam
		if err := rows.Scan(&sc.ServiceID, &sc.ServiceName, &sc.Count); err != nil {
			return nil, err
		}
		result.ByService = append(result.ByService, sc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	const customerInitiated = `b.status = 'cancelled' AND b.decided_by = b.user_id AND b.booking_type != 'walk_in'`
	const staffInitiated = `b.status = 'rejected' OR (b.status = 'cancelled' AND NOT (b.decided_by = b.user_id AND b.booking_type != 'walk_in'))`
	byStaffQuery, byStaffArgs := cancellationFilterSQL(`
		SELECT
			ss.staff_id AS staff_id,
			COALESCE(st.staff_name, 'Owner-managed') AS staff_name,
			COUNT(*) FILTER (WHERE `+customerInitiated+`) AS customer_count,
			COUNT(*) FILTER (WHERE `+staffInitiated+`) AS staff_count
		FROM bookings b
		JOIN service_slot_options sso ON b.slot_option_id = sso.slot_option_id
		JOIN service_options so ON so.service_option_id = sso.service_option_id
		JOIN services s ON s.service_id = so.service_id
		JOIN service_slots ss ON ss.service_slot_id = sso.service_slot_id
		LEFT JOIN staff st ON st.staff_id = ss.staff_id
		WHERE s.business_id = $1
			AND b.deleted_at IS NULL
			AND ss.date BETWEEN $2 AND $3
			AND b.status IN ('cancelled', 'rejected')
	`, []any{businessID, from, until}, staffIDs, serviceIDs)
	// Postgres only substitutes a SELECT-list alias in ORDER BY for a bare
	// name — inside an expression like "customer_count + staff_count" it
	// tries (and fails) to resolve them as real input columns instead, so
	// the FILTER expressions are repeated here rather than referencing the
	// aliases above.
	byStaffQuery += " GROUP BY ss.staff_id, st.staff_name" +
		" ORDER BY COUNT(*) FILTER (WHERE " + customerInitiated + ") + COUNT(*) FILTER (WHERE " + staffInitiated + ") DESC"
	staffRows, err := r.DB.QueryContext(ctx, byStaffQuery, byStaffArgs...)
	if err != nil {
		return nil, err
	}
	defer staffRows.Close()
	for staffRows.Next() {
		var sc param.StaffCancellationParam
		var staffIDCol sql.NullInt64
		if err := staffRows.Scan(&staffIDCol, &sc.StaffName, &sc.CustomerCount, &sc.StaffCount); err != nil {
			return nil, err
		}
		if staffIDCol.Valid {
			sc.StaffID = &staffIDCol.Int64
		}
		result.ByStaff = append(result.ByStaff, sc)
	}
	return result, staffRows.Err()
}

// GetServiceFilterOptions lists every service ever owned by businessID,
// active or soft-deleted, for the cancellation analysis service filter —
// grouped by name (see ServiceFilterOptionParam) the same way
// GetServicePopularity and GetCancellationAnalysis's byService breakdown are.
func (r *analyticsRepo) GetServiceFilterOptions(ctx context.Context, businessID int64) ([]param.ServiceFilterOptionParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT service_name, array_agg(service_id ORDER BY service_id), bool_and(deleted_at IS NOT NULL)
		FROM services
		WHERE business_id = $1
		GROUP BY service_name
		ORDER BY service_name
	`, businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []param.ServiceFilterOptionParam
	for rows.Next() {
		var o param.ServiceFilterOptionParam
		var ids pq.Int64Array
		if err := rows.Scan(&o.ServiceName, &ids, &o.Deleted); err != nil {
			return nil, err
		}
		o.ServiceIDs = []int64(ids)
		results = append(results, o)
	}
	return results, rows.Err()
}
