package graph

import (
	"fyp/domain/param"
	"fyp/graph/model"
	"strconv"
	"strings"
	"time"
)

func MapBookingDetail(d *param.BookingDetailParam) *model.BookingDetail {
	if d == nil {
		return nil
	}
	// A walk-in's user_id is the recording owner/staff, not a real customer —
	// show a generic label instead of that person's own name.
	customerName := d.CustomerName
	if d.BookingType == "walk_in" {
		customerName = "Walk-in customer"
	}
	return &model.BookingDetail{
		BookingID:       strconv.FormatInt(d.BookingID, 10),
		Status:          model.BookingStatus(strings.ToUpper(d.Status)),
		BookingType:     model.BookingType(strings.ToUpper(d.BookingType)),
		ServiceSlotID:   strconv.FormatInt(d.ServiceSlotID, 10),
		SlotOptionID:    strconv.FormatInt(d.SlotOptionID, 10),
		ServiceOptionID: strconv.FormatInt(d.ServiceOptionID, 10),
		Date:            d.Date,
		StartTime:       d.StartTime,
		EndTime:         d.EndTime,
		ServiceName:     d.ServiceName,
		OptionName:      d.OptionName,
		StaffName:       d.StaffName,
		CustomerName:    customerName,
		CustomerEmail:   d.CustomerEmail,
		BusinessID:      strconv.FormatInt(d.BusinessID, 10),
		BusinessName:    d.BusinessName,
		Description:     d.Description,
		CreatedAt:       d.CreatedAt.Format(time.RFC3339),
	}
}

func MapUser(user *param.AuthUserParam) *model.User {
	if user == nil {
		return nil
	}

	var lockedUntil *string
	if user.LockedUntil != nil {
		value := user.LockedUntil.Format(time.RFC3339)
		lockedUntil = &value
	}

	return &model.User{
		UserID:              strconv.FormatInt(user.UserID, 10),
		Username:            user.Username,
		Email:               user.Email,
		ContactNumber:       *user.ContactNumber,
		FailedLoginAttempts: int32(user.FailedLoginAttempts),
		LockedUntil:         lockedUntil,
		MustResetPassword:   user.MustResetPassword,
		StaffProfile:        nil,
		Bookings:            []*model.Booking{},
	}
}

func MapService(s *param.ServiceParam) *model.Service {
	if s == nil {
		return nil
	}

	packages := make([]*model.ServiceOption, len(s.ServiceOptions))
	for i, pkg := range s.ServiceOptions {
		var effectiveFrom *string
		if pkg.EffectiveFrom != "" {
			effectiveFromValue := pkg.EffectiveFrom
			effectiveFrom = &effectiveFromValue
		}

		items := make([]*model.ServiceOptionItem, len(pkg.ServiceOptionItems))
		for j, item := range pkg.ServiceOptionItems {
			items[j] = &model.ServiceOptionItem{
				ServiceOptionItemID:   strconv.FormatInt(item.ServiceOptionItemID, 10),
				ServiceOptionItemName: item.ServiceOptionItemName,
			}
		}
		packages[i] = &model.ServiceOption{
			ServiceOptionID:    strconv.FormatInt(pkg.ServiceOptionID, 10),
			ServiceID:          strconv.FormatInt(pkg.ServiceID, 10),
			ServiceOptionName:  pkg.ServiceOptionName,
			Description:        pkg.Description,
			ServiceOptionItems: items,
			ServiceSlotOptions: []*model.ServiceSlotOption{},
			RecurringSchedules: []*model.RecurringSchedule{},
			Removed:            pkg.IsRemoved,
			EffectiveFrom:      effectiveFrom,
			EffectiveUntil:     pkg.EffectiveUntil,
			HasBooking:         pkg.HasBooking,
		}
	}

	return &model.Service{
		ServiceID:      strconv.FormatInt(s.ServiceID, 10),
		BusinessID:     strconv.FormatInt(s.BusinessID, 10),
		ServiceName:    s.ServiceName,
		Description:    s.Description,
		ServiceOptions: packages,
	}
}

func MapBusinessProfile(businessProfile *param.BusinessProfileParam, owner *param.AuthUserParam) *model.BusinessProfile {
	if businessProfile == nil {
		return nil
	}

	workingHours := make([]*model.BusinessWorkingHour, len(businessProfile.WorkingHours))
	for i, wh := range businessProfile.WorkingHours {
		workingHours[i] = &model.BusinessWorkingHour{
			Day:       model.DayOfWeek(wh.Day),
			StartTime: wh.StartTime,
			EndTime:   wh.EndTime,
		}
	}

	return &model.BusinessProfile{
		BusinessID:            strconv.FormatInt(businessProfile.BusinessID, 10),
		OwnerUserID:           strconv.FormatInt(businessProfile.OwnerUserID, 10),
		Owner:                 MapUser(owner),
		BusinessName:          businessProfile.BusinessName,
		Description:           businessProfile.Description,
		Address:               businessProfile.Address,
		ImageURL:              businessProfile.ImageURL,
		BusinessContactNumber: businessProfile.BusinessContactNumber,
		BusinessEmail:         businessProfile.BusinessEmail,
		WorkingHours:          workingHours,
		Services:              []*model.Service{},
		Staff:                 []*model.Staff{},
	}
}

func MapServiceSlot(s *param.ServiceSlotParam) *model.ServiceSlot {
	if s == nil {
		return nil
	}

	slotID := strconv.FormatInt(s.ServiceSlotID, 10)

	var staffIDStr *string
	var staffModel *model.Staff
	if s.StaffID != nil {
		idStr := strconv.FormatInt(*s.StaffID, 10)
		staffIDStr = &idStr
		staffModel = MapStaff(&param.StaffParam{StaffID: *s.StaffID, StaffName: s.StaffName})
	}

	packages := make([]*model.ServiceSlotOption, len(s.Packages))
	for i, pkg := range s.Packages {
		packages[i] = &model.ServiceSlotOption{
			SlotOptionID:    strconv.FormatInt(pkg.SlotOptionID, 10),
			ServiceOptionID: strconv.FormatInt(pkg.ServiceOptionID, 10),
			ServiceSlotID:   slotID,
			ServiceOption: &model.ServiceOption{
				ServiceOptionID:   strconv.FormatInt(pkg.ServiceOptionID, 10),
				ServiceID:         strconv.FormatInt(pkg.ServiceID, 10),
				ServiceOptionName: pkg.ServiceOptionName,
				Service: &model.Service{
					ServiceID:      strconv.FormatInt(pkg.ServiceID, 10),
					ServiceName:    pkg.ServiceName,
					ServiceOptions: []*model.ServiceOption{},
				},
				ServiceOptionItems: []*model.ServiceOptionItem{},
				ServiceSlotOptions: []*model.ServiceSlotOption{},
				RecurringSchedules: []*model.RecurringSchedule{},
			},
			Bookings: []*model.Booking{},
		}
	}

	return &model.ServiceSlot{
		ServiceSlotID:      slotID,
		StaffID:            staffIDStr,
		Staff:              staffModel,
		Date:               s.Date,
		StartTime:          s.StartTime,
		EndTime:            s.EndTime,
		ServiceSlotOptions: packages,
		HasBooking:         s.HasBooking,
	}
}

func MapStaff(staff *param.StaffParam) *model.Staff {
	if staff == nil {
		return nil
	}

	position := staff.Position

	staffIDStr := strconv.FormatInt(staff.StaffID, 10)
	workingHours := make([]*model.StaffWorkingHour, len(staff.WorkingHours))
	for i, wh := range staff.WorkingHours {
		workingHours[i] = &model.StaffWorkingHour{
			StaffID:   staffIDStr,
			Day:       model.DayOfWeek(wh.Day),
			StartTime: wh.StartTime,
			EndTime:   wh.EndTime,
		}
	}

	return &model.Staff{
		StaffID:            staffIDStr,
		UserID:             strconv.FormatInt(staff.UserID, 10),
		BusinessID:         strconv.FormatInt(staff.BusinessID, 10),
		Name:               staff.StaffName,
		Email:              staff.StaffEmail,
		MustResetPassword:  staff.MustResetPassword,
		ContactNumber:      staff.StaffContactNumber,
		Position:           &position,
		WorkingHours:       workingHours,
		LeaveApplications:  []*model.LeaveApplication{},
		ServiceSlots:       []*model.ServiceSlot{},
		RecurringSchedules: []*model.RecurringSchedule{},
		HasBooking:         staff.HasBooking,
	}
}

func MapLeaveApplication(l *param.LeaveApplicationParam) *model.LeaveApplication {
	if l == nil {
		return nil
	}

	var position *string
	if l.Position != "" {
		position = &l.Position
	}
	var decidedAt *string
	if l.DecidedAt != nil {
		formatted := l.DecidedAt.Format(time.RFC3339)
		decidedAt = &formatted
	}

	affectedBookings := make([]*model.BookingDetail, len(l.AffectedBookings))
	for i := range l.AffectedBookings {
		affectedBookings[i] = MapBookingDetail(&l.AffectedBookings[i])
	}

	return &model.LeaveApplication{
		LeaveID:          strconv.FormatInt(l.LeaveID, 10),
		StaffID:          strconv.FormatInt(l.StaffID, 10),
		StaffName:        l.StaffName,
		Position:         position,
		StartDate:        l.StartDate,
		EndDate:          l.EndDate,
		Justification:    l.Justification,
		FileURL:          l.FileURL,
		Status:           model.LeaveStatus(strings.ToUpper(l.Status)),
		Remark:           l.Remark,
		DecidedAt:        decidedAt,
		CreatedAt:        l.CreatedAt.Format(time.RFC3339),
		AffectedBookings: affectedBookings,
	}
}

// ── Analytics (UC-12) ────────────────────────────────────────────────────────

func MapBookingSummary(s *param.BookingSummaryParam) *model.BookingSummary {
	if s == nil {
		return nil
	}
	byStatus := make([]*model.StatusCount, len(s.ByStatus))
	for i, sc := range s.ByStatus {
		byStatus[i] = &model.StatusCount{
			Status: model.BookingStatus(strings.ToUpper(sc.Status)),
			Count:  int32(sc.Count),
		}
	}
	byType := make([]*model.TypeCount, len(s.ByType))
	for i, tc := range s.ByType {
		byType[i] = &model.TypeCount{
			BookingType: model.BookingType(strings.ToUpper(tc.BookingType)),
			Count:       int32(tc.Count),
		}
	}
	return &model.BookingSummary{
		TotalBookings:      int32(s.TotalBookings),
		ByStatus:           byStatus,
		ByType:             byType,
		TodayAcceptedCount: int32(s.TodayAcceptedCount),
		PendingCount:       int32(s.PendingCount),
	}
}

func MapBookingTrendPoints(points []param.BookingTrendPointParam) []*model.BookingTrendPoint {
	result := make([]*model.BookingTrendPoint, len(points))
	for i, p := range points {
		result[i] = &model.BookingTrendPoint{Date: p.Date, Count: int32(p.Count)}
	}
	return result
}

func MapServicePopularity(items []param.ServicePopularityParam) []*model.ServicePopularity {
	result := make([]*model.ServicePopularity, len(items))
	for i, sp := range items {
		result[i] = &model.ServicePopularity{
			ServiceID:    strconv.FormatInt(sp.ServiceID, 10),
			ServiceName:  sp.ServiceName,
			BookingCount: int32(sp.BookingCount),
		}
	}
	return result
}

// utilizationRate is bookedSlots/totalSlots, 0 when there are no slots at
// all (rather than dividing by zero) — shared by slot- and staff-utilization.
func utilizationRate(booked, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(booked) / float64(total)
}

func MapSlotUtilization(u *param.SlotUtilizationParam) *model.SlotUtilization {
	if u == nil {
		return nil
	}
	sections := make([]*model.UtilizationSection, len(u.Sections))
	for i, s := range u.Sections {
		points := make([]*model.UtilizationBreakdownPoint, len(s.Points))
		for j, p := range s.Points {
			points[j] = &model.UtilizationBreakdownPoint{
				Label:       p.Label,
				TotalSlots:  int32(p.TotalSlots),
				BookedSlots: int32(p.BookedSlots),
			}
		}
		sections[i] = &model.UtilizationSection{Subtitle: s.Subtitle, Points: points}
	}
	return &model.SlotUtilization{
		TotalSlots:      int32(u.TotalSlots),
		BookedSlots:     int32(u.BookedSlots),
		UtilizationRate: utilizationRate(u.BookedSlots, u.TotalSlots),
		Sections:        sections,
	}
}

func MapStaffUtilization(items []param.StaffUtilizationParam) []*model.StaffUtilization {
	result := make([]*model.StaffUtilization, len(items))
	for i, su := range items {
		var id *string
		if su.StaffID != nil {
			v := strconv.FormatInt(*su.StaffID, 10)
			id = &v
		}
		result[i] = &model.StaffUtilization{
			StaffID:     id,
			StaffName:   su.StaffName,
			Bookings:    int32(su.BookedSlots),
			HoursBooked: su.BookedHours,
		}
	}
	return result
}

func MapCustomerRetention(r *param.CustomerRetentionParam) *model.CustomerRetention {
	if r == nil {
		return nil
	}
	return &model.CustomerRetention{
		NewCustomers:       int32(r.NewCustomers),
		ReturningCustomers: int32(r.ReturningCustomers),
		TotalCustomers:     int32(r.NewCustomers + r.ReturningCustomers),
	}
}

// rate is numerator/denominator, 0 when the denominator is zero — shared by
// CancellationAnalysis's combined cancelled-or-rejected rate.
func rate(numerator, denominator int64) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func MapCancellationAnalysis(c *param.CancellationAnalysisParam) *model.CancellationAnalysis {
	if c == nil {
		return nil
	}
	byService := make([]*model.ServiceCancellation, len(c.ByService))
	for i, sc := range c.ByService {
		byService[i] = &model.ServiceCancellation{
			ServiceID:   strconv.FormatInt(sc.ServiceID, 10),
			ServiceName: sc.ServiceName,
			Count:       int32(sc.Count),
		}
	}
	byStaff := make([]*model.StaffCancellation, len(c.ByStaff))
	for i, sc := range c.ByStaff {
		var staffID *string
		if sc.StaffID != nil {
			v := strconv.FormatInt(*sc.StaffID, 10)
			staffID = &v
		}
		byStaff[i] = &model.StaffCancellation{
			StaffID:       staffID,
			StaffName:     sc.StaffName,
			CustomerCount: int32(sc.CustomerCount),
			StaffCount:    int32(sc.StaffCount),
		}
	}
	totalCancelledOrRejected := c.TotalCancelled + c.TotalRejected
	return &model.CancellationAnalysis{
		TotalCancelled:           int32(c.TotalCancelled),
		TotalRejected:            int32(c.TotalRejected),
		TotalCancelledOrRejected: int32(totalCancelledOrRejected),
		CancelledOrRejectedRate:  rate(totalCancelledOrRejected, c.TotalBookings),
		ByService:                byService,
		ByStaff:                  byStaff,
	}
}

func MapServiceFilterOptions(options []param.ServiceFilterOptionParam) []*model.ServiceFilterOption {
	result := make([]*model.ServiceFilterOption, len(options))
	for i, o := range options {
		ids := make([]string, len(o.ServiceIDs))
		for j, id := range o.ServiceIDs {
			ids[j] = strconv.FormatInt(id, 10)
		}
		result[i] = &model.ServiceFilterOption{
			ServiceIds:  ids,
			ServiceName: o.ServiceName,
			Deleted:     o.Deleted,
		}
	}
	return result
}
