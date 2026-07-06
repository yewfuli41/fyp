package param

import (
	"fmt"
	"fyp/domain/errs"
	"strings"
)

type ServiceOptionItemParam struct {
	ServiceOptionItemID    int64
	ServiceOptionID int64
	ServiceOptionItemName  string
}

type ServiceOptionParam struct {
	ServiceOptionID   int64
	ServiceID          int64
	ServiceOptionName string
	Description        *string
	ServiceOptionItems       []ServiceOptionItemParam
}

type ServiceParam struct {
	ServiceID       int64
	BusinessID      int64
	ServiceName     string
	Description     *string
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
