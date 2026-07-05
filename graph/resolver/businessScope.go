package resolver

import (
	"context"
	"errors"
	"fyp/domain/errs"
	"fyp/domain/param"
)

// resolveBusinessScope resolves who's acting on business-scoped data
// (service slots, services, etc).
//
// A business owner gets their own businessID with no restriction (staffID
// nil). A staff member (no business profile of their own) gets their
// business's ID plus their own staffID — callers must then force any
// staff-related input to that staffID and verify any existing entity already
// belongs to them before mutating or returning it.
func resolveBusinessScope(ctx context.Context, r *Resolver, currentUser *param.AuthUserParam) (businessID int64, staffID *int64, err error) {
	businessProfile, err := r.App.BusinessService.GetBusinessProfileByOwnerID(ctx, currentUser.UserID)
	if err == nil {
		return businessProfile.BusinessID, nil, nil
	}
	if !errors.Is(err, errs.ErrBusinessProfileNotFound) {
		return 0, nil, err
	}

	staffProfile, staffErr := r.App.StaffService.GetStaffProfileByUserID(ctx, currentUser.UserID)
	if staffErr != nil {
		return 0, nil, err
	}

	return staffProfile.BusinessID, &staffProfile.StaffID, nil
}
