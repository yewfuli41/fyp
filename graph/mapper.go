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
		ContactNumber:       user.ContactNumber,
		FailedLoginAttempts: int32(user.FailedLoginAttempts),
		LockedUntil:         lockedUntil,
		StaffProfiles:       []*model.Staff{},
		Bookings:            []*model.Booking{},
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
