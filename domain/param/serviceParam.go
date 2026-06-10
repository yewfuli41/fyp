package param

import (
	"fyp/domain/errs"
	"strings"
)

type PackageItemParam struct {
	PackageItemID    int64
	ServicePackageID int64
	PackageItemName  string
}

type ServicePackageParam struct {
	ServicePackageID   int64
	ServiceID          int64
	ServicePackageName string
	Description        *string
	PackageItems       []PackageItemParam
}

type ServiceParam struct {
	ServiceID       int64
	BusinessID      int64
	ServiceName     string
	Description     *string
	ServicePackages []ServicePackageParam
}

func (p ServiceParam) Validate() error {
	var validationErrs errs.ValidationErrors

	if strings.TrimSpace(p.ServiceName) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "serviceName", Message: "service name is required"})
	}

	if len(validationErrs) > 0 {
		return validationErrs
	}
	return nil
}
