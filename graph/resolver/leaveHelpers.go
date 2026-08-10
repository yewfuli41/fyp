package resolver

import (
	"context"
	"fmt"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/graph/model"
)

// currentStaffProfile resolves the calling user's own staff profile — used
// by the leave-application mutations/queries a staff member acts on for
// themselves (applyLeave, deleteLeaveApplication, myLeaveApplications).
func currentStaffProfile(ctx context.Context, r *Resolver, currentUser *param.AuthUserParam) (*param.StaffParam, error) {
	staffProfile, err := r.App.StaffService.GetStaffProfileByUserID(ctx, currentUser.UserID)
	if err != nil {
		return nil, errs.ValidationErrors{{Field: "leaveId", Message: "Only staff members can perform this action."}}
	}
	return staffProfile, nil
}

func toWorkingHourParams(inputs []*model.WorkingHourInput) []param.WorkingHourParam {
	result := make([]param.WorkingHourParam, len(inputs))
	for i, wh := range inputs {
		result[i] = param.WorkingHourParam{Day: string(wh.Day), StartTime: wh.StartTime, EndTime: wh.EndTime}
	}
	return result
}

func toReassignmentParams(inputs []*model.SlotReassignmentInput) ([]param.SlotReassignmentParam, error) {
	result := make([]param.SlotReassignmentParam, len(inputs))
	for i, in := range inputs {
		sid, err := parseID(in.ServiceSlotID)
		if err != nil {
			return nil, fmt.Errorf("invalid service slot ID")
		}
		var staffID *int64
		if in.StaffID != nil && *in.StaffID != "" {
			id, err := parseID(*in.StaffID)
			if err != nil {
				return nil, fmt.Errorf("invalid staff ID")
			}
			staffID = &id
		}
		result[i] = param.SlotReassignmentParam{ServiceSlotID: sid, StaffID: staffID}
	}
	return result, nil
}
