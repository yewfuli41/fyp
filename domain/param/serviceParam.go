package param

import (
	"fmt"
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
		validationErrs = append(validationErrs, errs.ValidationError{Field: "serviceName", Message: "Service name is required"})
	} else if e, ok := maxLengthError("serviceName", "service name", p.ServiceName, maxServiceNameLength); ok {
		validationErrs = append(validationErrs, e)
	}

	for i, pkg := range p.ServicePackages {
		if e, ok := maxLengthError(
			fmt.Sprintf("servicePackages[%d].servicePackageName", i),
			"service package name", pkg.ServicePackageName, maxPackageNameLength,
		); ok {
			validationErrs = append(validationErrs, e)
		}

		for j, item := range pkg.PackageItems {
			if e, ok := maxLengthError(
				fmt.Sprintf("servicePackages[%d].packageItems[%d]", i, j),
				"package item name", item.PackageItemName, maxPackageNameLength,
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
