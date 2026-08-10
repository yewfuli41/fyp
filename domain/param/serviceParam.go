package param

import (
	"fmt"
	"fyp/domain/errs"
	"strings"
)

type ServiceOptionItemParam struct {
	ServiceOptionItemID   int64
	ServiceOptionID       int64
	ServiceOptionItemName string
}

type ServiceOptionParam struct {
	ServiceOptionID    int64
	ServiceID          int64
	ServiceOptionName  string
	Description        *string
	ServiceOptionItems []ServiceOptionItemParam

	// An option's own validity window. EffectiveUntil nil = open-ended. Name
	// and items are immutable once created (see optionContentChanged in
	// serviceService.go) — to change them, delete this option (only possible
	// while it has no booking) and create a replacement; only the window
	// itself can be resized in place.
	EffectiveFrom  string  // "YYYY-MM-DD"
	EffectiveUntil *string // "YYYY-MM-DD" or nil

	IsRemoved bool // output: this option's own window doesn't cover today

	// HasBooking: output only — true once anything has ever booked this
	// option, which is exactly what blocks deleting it (see
	// serviceService.UpdateService). Lets the frontend refuse the delete
	// immediately instead of only finding out after a full form submit.
	HasBooking bool

	// IsDefault: true for exactly one option per service — it can't be
	// deleted or given an end date, so a service always has one available.
	IsDefault bool

	// ClearEffectiveUntil: true to explicitly clear an unchanged option's
	// already-saved EffectiveUntil, reopening it. EffectiveUntil alone can't
	// express "clear it" since blank and "not set this time" serialize
	// identically over the wire.
	ClearEffectiveUntil bool
}

type ServiceParam struct {
	ServiceID      int64
	BusinessID     int64
	ServiceName    string
	Description    *string
	ServiceOptions []ServiceOptionParam
}

func (p ServiceParam) Validate() error {
	var validationErrs errs.ValidationErrors

	if strings.TrimSpace(p.ServiceName) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "serviceName", Message: "Service name is required"})
	} else if e, ok := maxLengthError("serviceName", "service name", p.ServiceName, maxServiceNameLength); ok {
		validationErrs = append(validationErrs, e)
	}

	for i, pkg := range p.ServiceOptions {
		if e, ok := maxLengthError(
			fmt.Sprintf("serviceOptions[%d].serviceOptionName", i),
			"service option name", pkg.ServiceOptionName, maxPackageNameLength,
		); ok {
			validationErrs = append(validationErrs, e)
		}

		for j, item := range pkg.ServiceOptionItems {
			if e, ok := maxLengthError(
				fmt.Sprintf("serviceOptions[%d].serviceOptionItems[%d]", i, j),
				"service option item name", item.ServiceOptionItemName, maxPackageNameLength,
			); ok {
				validationErrs = append(validationErrs, e)
			}
		}
	}

	if len(validationErrs) > 0 {
		return validationErrs
	}
	return nil
}
