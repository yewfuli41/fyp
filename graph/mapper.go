package graph

import (
	"fyp/domain/param"
	"fyp/graph/model"
	"strconv"
	"time"
)

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
		items := make([]*model.ServiceOptionItem, len(pkg.ServiceOptionItems))
		for j, item := range pkg.ServiceOptionItems {
			items[j] = &model.ServiceOptionItem{
				ServiceOptionItemID:   strconv.FormatInt(item.ServiceOptionItemID, 10),
				ServiceOptionItemName: item.ServiceOptionItemName,
			}
		}
		packages[i] = &model.ServiceOption{
			ServiceOptionID:    strconv.FormatInt(pkg.ServiceOptionID, 10),
			ServiceID:           strconv.FormatInt(pkg.ServiceID, 10),
			ServiceOptionName:  pkg.ServiceOptionName,
			Description:         pkg.Description,
			ServiceOptionItems:        items,
			ServiceSlotOptions: []*model.ServiceSlotOption{},
			RecurringSchedules:  []*model.RecurringSchedule{},
		}
	}

	return &model.Service{
		ServiceID:       strconv.FormatInt(s.ServiceID, 10),
		BusinessID:      strconv.FormatInt(s.BusinessID, 10),
		ServiceName:     s.ServiceName,
		Description:     s.Description,
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
			ServiceSlotID:    slotID,
			ServiceOption: &model.ServiceOption{
				ServiceOptionID:   strconv.FormatInt(pkg.ServiceOptionID, 10),
				ServiceID:          strconv.FormatInt(pkg.ServiceID, 10),
				ServiceOptionName: pkg.ServiceOptionName,
				Service: &model.Service{
					ServiceID:       strconv.FormatInt(pkg.ServiceID, 10),
					ServiceName:     pkg.ServiceName,
					ServiceOptions: []*model.ServiceOption{},
				},
				ServiceOptionItems:        []*model.ServiceOptionItem{},
				ServiceSlotOptions: []*model.ServiceSlotOption{},
				RecurringSchedules:  []*model.RecurringSchedule{},
			},
			Bookings: []*model.Booking{},
		}
	}

	return &model.ServiceSlot{
		ServiceSlotID:       slotID,
		StaffID:             staffIDStr,
		Staff:               staffModel,
		Date:                s.Date,
		StartTime:           s.StartTime,
		EndTime:             s.EndTime,
		ServiceSlotOptions: packages,
		HasBooking:          s.HasBooking,
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
	}
}
