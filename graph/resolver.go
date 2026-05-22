package graph

import (
	"fyp/app"
	"fyp/domain/param"
	"fyp/graph/model"
	"strconv"
	"time"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	App *app.App
}

func NewResolver(app *app.App) *Resolver {
	return &Resolver{
		App: app,
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
		ContactNumber:       user.ContactNumber,
		FailedLoginAttempts: int32(user.FailedLoginAttempts),
		LockedUntil:         lockedUntil,
		StaffProfiles:       []*model.Staff{},
		Bookings:            []*model.Booking{},
	}
}
