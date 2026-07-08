package service

import (
	"context"
	"database/sql"
	"fmt"
	"fyp/database"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"sort"
	"strings"
)

type serviceService struct {
	serviceRepo interfaces.IServiceRepo
	tx          *database.Transaction
}

func NewServiceService(db *sql.DB, serviceRepo interfaces.IServiceRepo) interfaces.IServiceService {
	return &serviceService{
		serviceRepo: serviceRepo,
		tx:          database.NewTransaction(db),
	}
}


// ensureDefaultOption guarantees there is always at least one option.
// If the first option has no name the service name is used as a fallback.
func ensureDefaultOption(p *param.ServiceParam) {
	if len(p.ServiceOptions) == 0 {
		p.ServiceOptions = []param.ServiceOptionParam{{ServiceOptionName: p.ServiceName}}
		return
	}
	if strings.TrimSpace(p.ServiceOptions[0].ServiceOptionName) == "" {
		p.ServiceOptions[0].ServiceOptionName = p.ServiceName
	}
}

// itemSetSignature returns a canonical, order-independent representation of a
// package's item names, used to detect packages with identical item lists.
func itemSetSignature(items []param.ServiceOptionItemParam) string {
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = strings.ToLower(item.ServiceOptionItemName)
	}
	sort.Strings(names)
	return strings.Join(names, "\x00")
}

func validatePackageDuplicates(packages []param.ServiceOptionParam) errs.ValidationErrors {
	var validationErrs errs.ValidationErrors
	seenPkgNames := make(map[string]bool)
	seenItemSets := make(map[string]int)
	for i, pkg := range packages {
		lower := strings.ToLower(pkg.ServiceOptionName)
		if seenPkgNames[lower] {
			validationErrs = append(validationErrs, errs.ValidationError{
				Field: fmt.Sprintf("serviceOptionName[%d]", i), Message: "Service package name already exists",
			})
		}
		seenPkgNames[lower] = true

		seenItemNames := make(map[string]bool)
		for j, item := range pkg.ServiceOptionItems {
			lowerItem := strings.ToLower(item.ServiceOptionItemName)
			if seenItemNames[lowerItem] {
				validationErrs = append(validationErrs, errs.ValidationError{
					Field: fmt.Sprintf("serviceOptionItemName[%d][%d]", i, j), Message: "Package item name already exists",
				})
			}
			seenItemNames[lowerItem] = true
		}

		signature := itemSetSignature(pkg.ServiceOptionItems)
		if firstIdx, exists := seenItemSets[signature]; exists {
			validationErrs = append(validationErrs, errs.ValidationError{
				Field:   fmt.Sprintf("serviceOptions[%d]", i),
				Message: fmt.Sprintf("Package items are identical to package %d", firstIdx+1),
			})
		} else {
			seenItemSets[signature] = i
		}
	}
	return validationErrs
}

func (s *serviceService) CreateService(ctx context.Context, p param.ServiceParam) (*param.ServiceParam, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	ensureDefaultOption(&p)

	if validationErrs := validatePackageDuplicates(p.ServiceOptions); len(validationErrs) > 0 {
		return nil, validationErrs
	}

	var result *param.ServiceParam
	err := s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		created, err := s.serviceRepo.InsertService(ctx, tx, p)
		if err != nil {
			if database.IsUniqueViolation(err, "services_unique_name") {
				return errs.ValidationErrors{{Field: "serviceName", Message: "Service name already exists"}}
			}
			return err
		}

		for i, pkg := range p.ServiceOptions {
			pkg.ServiceID = created.ServiceID
			createdPkg, err := s.serviceRepo.InsertServiceOption(ctx, tx, pkg)
			if err != nil {
				if database.IsUniqueViolation(err, "service_options_unique_name") {
					return errs.ValidationErrors{{Field: fmt.Sprintf("serviceOptionName[%d]", i), Message: "Service package name already exists"}}
				}
				return err
			}
			for j, item := range pkg.ServiceOptionItems {
				item.ServiceOptionID = createdPkg.ServiceOptionID
				if err := s.serviceRepo.InsertServiceOptionItem(ctx, tx, item); err != nil {
					if database.IsUniqueViolation(err, "service_option_items_unique_name") {
						return errs.ValidationErrors{{Field: fmt.Sprintf("serviceOptionItemName[%d][%d]", i, j), Message: "Package item name already exists"}}
					}
					return err
				}
			}
			createdPkg.ServiceOptionItems = pkg.ServiceOptionItems
			created.ServiceOptions = append(created.ServiceOptions, *createdPkg)
		}
		result = created
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *serviceService) UpdateService(ctx context.Context, p param.ServiceParam) (*param.ServiceParam, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	ensureDefaultOption(&p)

	if validationErrs := validatePackageDuplicates(p.ServiceOptions); len(validationErrs) > 0 {
		return nil, validationErrs
	}

	var result *param.ServiceParam
	err := s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		updated, err := s.serviceRepo.UpdateService(ctx, tx, p)
		if err != nil {
			if database.IsUniqueViolation(err, "services_unique_name") {
				return errs.ValidationErrors{{Field: "serviceName", Message: "Service name already exists"}}
			}
			return err
		}

		if err := s.serviceRepo.DeleteServiceOptionsByServiceID(ctx, tx, updated.ServiceID); err != nil {
			return err
		}

		for i, pkg := range p.ServiceOptions {
			pkg.ServiceID = updated.ServiceID
			createdPkg, err := s.serviceRepo.InsertServiceOption(ctx, tx, pkg)
			if err != nil {
				if database.IsUniqueViolation(err, "service_options_unique_name") {
					return errs.ValidationErrors{{Field: fmt.Sprintf("serviceOptionName[%d]", i), Message: "Service package name already exists"}}
				}
				return err
			}
			for j, item := range pkg.ServiceOptionItems {
				item.ServiceOptionID = createdPkg.ServiceOptionID
				if err := s.serviceRepo.InsertServiceOptionItem(ctx, tx, item); err != nil {
					if database.IsUniqueViolation(err, "service_option_items_unique_name") {
						return errs.ValidationErrors{{Field: fmt.Sprintf("serviceOptionItemName[%d][%d]", i, j), Message: "Package item name already exists"}}
					}
					return err
				}
			}
			createdPkg.ServiceOptionItems = pkg.ServiceOptionItems
			updated.ServiceOptions = append(updated.ServiceOptions, *createdPkg)
		}
		result = updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *serviceService) DeleteService(ctx context.Context, serviceID int64, businessID int64) error {
	hasBooking, err := s.serviceRepo.HasBookingForService(ctx, serviceID)
	if err != nil {
		return err
	}
	if hasBooking {
		return fmt.Errorf("Deletion disabled - booking exists.")
	}

	return s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		return s.serviceRepo.SoftDeleteService(ctx, tx, serviceID, businessID)
	})
}

func (s *serviceService) GetServicesByBusinessID(ctx context.Context, businessID int64) ([]param.ServiceParam, error) {
	services, err := s.serviceRepo.GetServicesByBusinessID(ctx, businessID)
	if err != nil {
		return nil, err
	}

	for i := range services {
		packages, err := s.serviceRepo.GetServiceOptionsByServiceID(ctx, services[i].ServiceID)
		if err != nil {
			return nil, err
		}
		for j := range packages {
			items, err := s.serviceRepo.GetServiceOptionItemsByOptionID(ctx, packages[j].ServiceOptionID)
			if err != nil {
				return nil, err
			}
			packages[j].ServiceOptionItems = items
		}
		services[i].ServiceOptions = packages
	}
	return services, nil
}
