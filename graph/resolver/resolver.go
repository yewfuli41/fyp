package resolver

import (
	"context"
	"fyp/app"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/graph"
	"fyp/graph/graphErrs"
	"fyp/graph/model"
	"fyp/internal/contexts"
	"strconv"
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

func parseID(id string) (int64, error) {
	return strconv.ParseInt(id, 10, 64)
}

// parseStaffUnavailability turns the three optional "this staff can't work
// then" arguments into a param, requiring all three together — a staff id with
// no range (or a range with no staff) would silently hide nothing, which is
// the failure mode most likely to go unnoticed.
func parseStaffUnavailability(staffID *string, from *string, until *string) (*param.StaffUnavailability, error) {
	haveStaff := staffID != nil && *staffID != ""
	haveFrom := from != nil && *from != ""
	haveUntil := until != nil && *until != ""
	if !haveStaff && !haveFrom && !haveUntil {
		return nil, nil
	}
	if !haveStaff || !haveFrom || !haveUntil {
		return nil, errs.ValidationErrors{{
			Field:   "unavailableStaffId",
			Message: "unavailableStaffId, unavailableFrom and unavailableUntil must be given together.",
		}}
	}
	id, err := parseID(*staffID)
	if err != nil {
		return nil, err
	}
	return &param.StaffUnavailability{StaffID: id, From: *from, Until: *until}, nil
}

func parseIDs(ids []string) ([]int64, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	result := make([]int64, len(ids))
	for i, id := range ids {
		v, err := parseID(id)
		if err != nil {
			return nil, err
		}
		result[i] = v
	}
	return result, nil
}

func mapBookingDetails(details []param.BookingDetailParam) []*model.BookingDetail {
	result := make([]*model.BookingDetail, len(details))
	for i := range details {
		result[i] = graph.MapBookingDetail(&details[i])
	}
	return result
}

// bookingAction wires the current user + booking id into a single-booking service call.
func (r *mutationResolver) bookingAction(
	ctx context.Context, bookingID string,
	action func(userID, id int64) (*param.BookingDetailParam, error),
) (*model.BookingDetail, error) {
	currentUser, err := contexts.CurrentUser(ctx)
	if err != nil {
		return nil, graphErrs.ToGraphQLError(err)
	}
	id, err := parseID(bookingID)
	if err != nil {
		return nil, graphErrs.ToGraphQLError(err)
	}
	result, err := action(currentUser.UserID, id)
	if err != nil {
		return nil, graphErrs.ToGraphQLError(err)
	}
	return graph.MapBookingDetail(result), nil
}
