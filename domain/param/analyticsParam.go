package param

// StatusCountParam is a booking count for one status ("pending", "accepted", …).
type StatusCountParam struct {
	Status string
	Count  int64
}

// TypeCountParam is a booking count for one booking_type ("online", "walk_in").
type TypeCountParam struct {
	BookingType string
	Count       int64
}

// BookingSummaryParam is the headline totals widget — overall count (for the
// selected range) plus breakdowns by status and type, and two always-live,
// business-wide (not range-scoped) figures: today's accepted appointments
// and the total pending queue.
type BookingSummaryParam struct {
	TotalBookings      int64
	ByStatus           []StatusCountParam
	ByType             []TypeCountParam
	TodayAcceptedCount int64
	PendingCount       int64
}

// BookingTrendPointParam is one point on the bookings-over-time chart.
type BookingTrendPointParam struct {
	Date  string // "YYYY-MM-DD" (bucket start, per the requested granularity)
	Count int64
}

// ServicePopularityParam is one service's booking count within the range.
type ServicePopularityParam struct {
	ServiceID    int64
	ServiceName  string
	BookingCount int64
}

// UtilizationBreakdownParam is one point within a section — a weekday name,
// a zero-padded day-of-month ("01".."31"), a zero-padded month ("01".."12"),
// or a year, depending on the requested grouping (see GetSlotUtilization).
type UtilizationBreakdownParam struct {
	Label       string
	TotalSlots  int64
	BookedSlots int64
}

// UtilizationSectionParam groups related points under one subtitle — "date"
// grouping sections by month ("2026-07"), "month" grouping sections by year
// ("2026"). "weekday" and "year" groupings have no natural subtitle, so they
// come back as a single section with an empty Subtitle.
type UtilizationSectionParam struct {
	Subtitle string
	Points   []UtilizationBreakdownParam
}

// SlotUtilizationParam is the overall slot-utilization widget: totals plus
// the breakdown bucketed (and, for date/month, sectioned) by the requested grouping.
type SlotUtilizationParam struct {
	TotalSlots  int64
	BookedSlots int64
	Sections    []UtilizationSectionParam
}

// StaffUtilizationParam is one staff member's (or, when StaffID is nil,
// owner-managed slots') booking load within the range: how many slots were
// booked and how many hours that adds up to.
type StaffUtilizationParam struct {
	StaffID     *int64 // nil => owner-managed (no assigned staff)
	StaffName   string
	BookedSlots int64   // "Bookings"
	BookedHours float64 // "Hours Booked"
}

// CustomerRetentionParam is the new-vs-returning customer split within the range.
type CustomerRetentionParam struct {
	NewCustomers       int64
	ReturningCustomers int64
}

// ServiceCancellationParam is one service's cancelled-or-rejected booking
// count within the range — the two are combined since both mean the same
// thing to the business: a booking that didn't happen.
type ServiceCancellationParam struct {
	ServiceID   int64
	ServiceName string
	Count       int64
}

// StaffCancellationParam is one staff member's cancelled/rejected bookings
// within the range, split by who decided it — StaffID is nil for
// owner-managed (unassigned) slots. Rejections are always staff-initiated
// (a customer can't reject their own booking), so StaffCount also absorbs
// every rejection; CustomerCount only ever counts customer-initiated
// cancellations.
type StaffCancellationParam struct {
	StaffID       *int64
	StaffName     string
	CustomerCount int64
	StaffCount    int64
}

// CancellationAnalysisParam is the cancellation/rejection widget.
type CancellationAnalysisParam struct {
	TotalBookings  int64 // denominator for the rate below
	TotalCancelled int64
	TotalRejected  int64
	ByService      []ServiceCancellationParam
	ByStaff        []StaffCancellationParam
}

// ServiceFilterOptionParam is one entry in the cancellation analysis's
// service filter — unlike the dropdown on the service-management screens,
// this includes soft-deleted services, since a deleted service's past
// cancellations/rejections still show up in the byService breakdown and
// should stay filterable. Grouped by name, same as GetServicePopularity and
// GetCancellationAnalysis's byService breakdown, so a service that was
// deleted and recreated (or renamed back) is one filterable option covering
// every service_id that ever used that name — ServiceIDs is what gets
// passed as serviceIds when this option is selected. Deleted is true only
// when none of those service_ids are currently active.
type ServiceFilterOptionParam struct {
	ServiceIDs  []int64
	ServiceName string
	Deleted     bool
}
