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

	packages := make([]*model.ServicePackage, len(s.ServicePackages))
	for i, pkg := range s.ServicePackages {
		items := make([]*model.PackageItem, len(pkg.PackageItems))
		for j, item := range pkg.PackageItems {
			items[j] = &model.PackageItem{
				PackageItemID:   strconv.FormatInt(item.PackageItemID, 10),
				PackageItemName: item.PackageItemName,
			}
		}
		packages[i] = &model.ServicePackage{
			ServicePackageID:    strconv.FormatInt(pkg.ServicePackageID, 10),
			ServiceID:           strconv.FormatInt(pkg.ServiceID, 10),
			ServicePackageName:  pkg.ServicePackageName,
			Description:         pkg.Description,
			PackageItems:        items,
			ServiceSlotPackages: []*model.ServiceSlotPackage{},
			RecurringSchedules:  []*model.RecurringSchedule{},
		}
	}

	return &model.Service{
		ServiceID:       strconv.FormatInt(s.ServiceID, 10),
		BusinessID:      strconv.FormatInt(s.BusinessID, 10),
		ServiceName:     s.ServiceName,
		Description:     s.Description,
		ServicePackages: packages,
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

func MapStaff(staff *param.StaffParam) *model.Staff {
	if staff == nil {
		return nil
	}

	position := staff.Position

	return &model.Staff{
		StaffID:            strconv.FormatInt(staff.StaffID, 10),
		UserID:             strconv.FormatInt(staff.UserID, 10),
		BusinessID:         strconv.FormatInt(staff.BusinessID, 10),
		Name:               staff.StaffName,
		Email:              staff.StaffEmail,
		MustResetPassword:  staff.MustResetPassword,
		ContactNumber:      staff.StaffContactNumber,
		Position:           &position,
		WorkingHours:       []*model.StaffWorkingHour{},
		LeaveApplications:  []*model.LeaveApplication{},
		ServiceSlots:       []*model.ServiceSlot{},
		RecurringSchedules: []*model.RecurringSchedule{},
	}
}
