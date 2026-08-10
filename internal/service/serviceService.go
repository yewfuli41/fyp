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
	"time"
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
// When it has to synthesize the option outright (none submitted at all),
// EffectiveFrom is filled in as today too — the "must be set explicitly"
// rule in validateNewOptionWindow is about the owner's own input, not this
// system-generated fallback.
func ensureDefaultOption(p *param.ServiceParam) {
	if len(p.ServiceOptions) == 0 {
		p.ServiceOptions = []param.ServiceOptionParam{{
			ServiceOptionName: p.ServiceName,
			EffectiveFrom:     time.Now().Format("2006-01-02"),
		}}
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
				Field: fmt.Sprintf("serviceOptionName[%d]", i), Message: "Service option name already exists",
			})
		}
		seenPkgNames[lower] = true

		seenItemNames := make(map[string]bool)
		for j, item := range pkg.ServiceOptionItems {
			lowerItem := strings.ToLower(item.ServiceOptionItemName)
			if seenItemNames[lowerItem] {
				validationErrs = append(validationErrs, errs.ValidationError{
					Field: fmt.Sprintf("serviceOptionItemName[%d][%d]", i, j), Message: "Option item name already exists",
				})
			}
			seenItemNames[lowerItem] = true
		}

		if len(pkg.ServiceOptionItems) > 0 {
			signature := itemSetSignature(pkg.ServiceOptionItems)
			if firstIdx, exists := seenItemSets[signature]; exists {
				validationErrs = append(validationErrs, errs.ValidationError{
					Field:   fmt.Sprintf("serviceOptions[%d]", i),
					Message: fmt.Sprintf("Option items are identical to option %d", firstIdx+1),
				})
			} else {
				seenItemSets[signature] = i
			}
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
			// The default option (position 0) can't be given an end date.
			if i == 0 && pkg.EffectiveUntil != nil && *pkg.EffectiveUntil != "" {
				return errs.ValidationErrors{{Field: fmt.Sprintf("serviceOptions[%d]", i), Message: "The default option can't have an effective-until date."}}
			}
			pkg.IsDefault = i == 0
			from, err := validateNewOptionWindow(pkg, i)
			if err != nil {
				return err
			}
			pkg.EffectiveFrom = from
			createdPkg, err := s.serviceRepo.InsertServiceOption(ctx, tx, pkg)
			if err != nil {
				if database.IsUniqueViolation(err, "service_options_unique_name") {
					return errs.ValidationErrors{{Field: fmt.Sprintf("serviceOptionName[%d]", i), Message: "Service option name already exists"}}
				}
				return err
			}
			for j, item := range pkg.ServiceOptionItems {
				item.ServiceOptionID = createdPkg.ServiceOptionID
				if err := s.serviceRepo.InsertServiceOptionItem(ctx, tx, item); err != nil {
					if database.IsUniqueViolation(err, "service_option_items_unique_name") {
						return errs.ValidationErrors{{Field: fmt.Sprintf("serviceOptionItemName[%d][%d]", i, j), Message: "Option item name already exists"}}
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

		// The existing options, with their items, are the baseline the
		// submitted form is compared against.
		existingOptions, err := s.serviceRepo.GetServiceOptionsByServiceID(ctx, updated.ServiceID)
		if err != nil {
			return err
		}
		existingByID := make(map[int64]param.ServiceOptionParam, len(existingOptions))
		for i := range existingOptions {
			items, err := s.serviceRepo.GetServiceOptionItemsByOptionID(ctx, existingOptions[i].ServiceOptionID)
			if err != nil {
				return err
			}
			existingOptions[i].ServiceOptionItems = items
			existingByID[existingOptions[i].ServiceOptionID] = existingOptions[i]
		}
		matched := make(map[int64]bool, len(existingOptions))
		var defaultOptionID int64

		for i, pkg := range p.ServiceOptions {
			pkg.ServiceID = updated.ServiceID

			existing, isExisting := existingByID[pkg.ServiceOptionID]
			if pkg.ServiceOptionID != 0 && !isExisting {
				return errs.ValidationErrors{{Field: fmt.Sprintf("serviceOptions[%d]", i), Message: "This option no longer exists"}}
			}

			// The default option (position 0 — a service always needs one
			// available) can't be given an end date. To retire it, the owner
			// must set a different option as default first, then delete it.
			// Also checks the option's already-saved end date — unless this
			// same submission clears it (ClearEffectiveUntil), in which case
			// the clear below applies before this option becomes the default.
			if i == 0 {
				field := fmt.Sprintf("serviceOptions[%d]", i)
				if pkg.EffectiveUntil != nil && *pkg.EffectiveUntil != "" {
					return errs.ValidationErrors{{Field: field, Message: "The default option can't have an effective-until date. Set another option as default first."}}
				}
				if isExisting && existing.EffectiveUntil != nil && *existing.EffectiveUntil != "" && !pkg.ClearEffectiveUntil {
					return errs.ValidationErrors{{Field: field, Message: "This option has an effective-until date set. Clear it (or delete this option and add a replacement without one), then set it as default."}}
				}
			}

			// Brand new, standalone option — its own [from, until]. from
			// defaults to today when omitted; until optional (blank = permanent).
			if !isExisting {
				from, err := validateNewOptionWindow(pkg, i)
				if err != nil {
					return err
				}
				created, err := s.insertOptionWithItems(ctx, tx, param.ServiceOptionParam{
					ServiceID:         updated.ServiceID,
					ServiceOptionName: pkg.ServiceOptionName,
					Description:       pkg.Description,
					EffectiveFrom:     from,
					EffectiveUntil:    pkg.EffectiveUntil,
				}, pkg.ServiceOptionItems, i)
				if err != nil {
					return err
				}
				if i == 0 {
					defaultOptionID = created.ServiceOptionID
				}
				updated.ServiceOptions = append(updated.ServiceOptions, *created)
				continue
			}

			matched[existing.ServiceOptionID] = true
			if i == 0 {
				defaultOptionID = existing.ServiceOptionID
			}

			// Unchanged content → the option's name/items stay as-is, but its
			// own window can still be resized in place. The default can't
			// gain an end date; that's already guaranteed above.
			if !optionContentChanged(existing, pkg) {
				field := fmt.Sprintf("serviceOptions[%d]", i)
				today := time.Now().Format("2006-01-02")

				newFrom := existing.EffectiveFrom
				fromChanged := pkg.EffectiveFrom != "" && pkg.EffectiveFrom != existing.EffectiveFrom
				if fromChanged {
					msg := "Effective from cannot be in the past"
					if i == 0 {
						msg = "The default option's effective-from date cannot be in the past."
					}
					if pkg.EffectiveFrom < today {
						return errs.ValidationErrors{{Field: field, Message: msg}}
					}
					newFrom = pkg.EffectiveFrom
				}

				newUntil := existing.EffectiveUntil
				untilChanged := false
				if pkg.EffectiveUntil != nil && *pkg.EffectiveUntil != "" {
					if *pkg.EffectiveUntil < today {
						return errs.ValidationErrors{{Field: field, Message: "Effective until cannot be in the past"}}
					}
					if *pkg.EffectiveUntil < newFrom {
						return errs.ValidationErrors{{Field: field, Message: "Effective until must be on or after effective from"}}
					}
					untilChanged = existing.EffectiveUntil == nil || *existing.EffectiveUntil != *pkg.EffectiveUntil
					newUntil = pkg.EffectiveUntil
				} else if pkg.ClearEffectiveUntil && existing.EffectiveUntil != nil {
					newUntil = nil
					untilChanged = true
				}

				if fromChanged || untilChanged {
					if err := s.serviceRepo.SetOptionWindow(ctx, tx, existing.ServiceOptionID, newFrom, newUntil); err != nil {
						return err
					}
					existing.EffectiveFrom = newFrom
					existing.EffectiveUntil = newUntil
				}

				updated.ServiceOptions = append(updated.ServiceOptions, existing)
				continue
			}

			// Changed → renaming or changing items on an existing option directly
			// isn't supported (options are immutable). Delete it (it must have
			// no booking) and add a replacement option instead.
			return errs.ValidationErrors{{
				Field:   fmt.Sprintf("serviceOptions[%d]", i),
				Message: "To change this option's name or items, delete it and add a new option instead.",
			}}
		}

		// Options no longer in the submission → delete them, provided nothing
		// booked references them.
		for _, opt := range existingOptions {
			if matched[opt.ServiceOptionID] {
				continue
			}
			hasBooking, err := s.serviceRepo.HasBookingForOption(ctx, opt.ServiceOptionID)
			if err != nil {
				return err
			}
			if hasBooking {
				return errs.ValidationErrors{{Field: "serviceOptions", Message: fmt.Sprintf("Can't delete %q — it has a booking.", opt.ServiceOptionName)}}
			}
			if err := s.serviceRepo.SoftDeleteServiceOption(ctx, tx, opt.ServiceOptionID); err != nil {
				return err
			}
		}

		if err := s.serviceRepo.SetServiceDefaultOption(ctx, tx, updated.ServiceID, defaultOptionID); err != nil {
			return err
		}

		result = updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// optionContentChanged reports whether the submitted option differs from the
// stored option in name or in its set of item names.
func optionContentChanged(existing, submitted param.ServiceOptionParam) bool {
	if strings.TrimSpace(existing.ServiceOptionName) != strings.TrimSpace(submitted.ServiceOptionName) {
		return true
	}
	if len(existing.ServiceOptionItems) != len(submitted.ServiceOptionItems) {
		return true
	}
	counts := make(map[string]int)
	for _, it := range existing.ServiceOptionItems {
		counts[strings.ToLower(strings.TrimSpace(it.ServiceOptionItemName))]++
	}
	for _, it := range submitted.ServiceOptionItems {
		counts[strings.ToLower(strings.TrimSpace(it.ServiceOptionItemName))]--
	}
	for _, c := range counts {
		if c != 0 {
			return true
		}
	}
	return false
}

// validateNewOptionWindow validates the effective window for a brand new
// option: from is required (the owner must pick it explicitly — no more
// silently defaulting to today) and can't be in the past; until is optional
// (blank = permanent).
func validateNewOptionWindow(pkg param.ServiceOptionParam, i int) (string, error) {
	from := pkg.EffectiveFrom
	today := time.Now().Format("2006-01-02")
	field := fmt.Sprintf("serviceOptions[%d]", i)
	if from == "" {
		return "", errs.ValidationErrors{{Field: field, Message: "Effective from is required"}}
	}
	if from < today {
		return "", errs.ValidationErrors{{Field: field, Message: "Effective from cannot be in the past"}}
	}
	if pkg.EffectiveUntil != nil && *pkg.EffectiveUntil != "" && *pkg.EffectiveUntil < from {
		return "", errs.ValidationErrors{{Field: field, Message: "Effective until must be on or after effective from"}}
	}
	return from, nil
}

func (s *serviceService) insertOptionWithItems(ctx context.Context, tx *sql.Tx, opt param.ServiceOptionParam, items []param.ServiceOptionItemParam, i int) (*param.ServiceOptionParam, error) {
	created, err := s.serviceRepo.InsertServiceOption(ctx, tx, opt)
	if err != nil {
		if database.IsUniqueViolation(err, "service_options_unique_name") {
			return nil, errs.ValidationErrors{{Field: fmt.Sprintf("serviceOptionName[%d]", i), Message: "Service option name already exists"}}
		}
		return nil, err
	}
	for j, item := range items {
		item.ServiceOptionID = created.ServiceOptionID
		if err := s.serviceRepo.InsertServiceOptionItem(ctx, tx, item); err != nil {
			if database.IsUniqueViolation(err, "service_option_items_unique_name") {
				return nil, errs.ValidationErrors{{Field: fmt.Sprintf("serviceOptionItemName[%d][%d]", i, j), Message: "Option item name already exists"}}
			}
			return nil, err
		}
	}
	created.ServiceOptionItems = items
	return created, nil
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
		// HasBookingForService already confirmed no active booking exists
		// anywhere under this service, so every option (and the slots that
		// offer them) is safe to retire along with the service itself.
		options, err := s.serviceRepo.GetServiceOptionsByServiceID(ctx, serviceID)
		if err != nil {
			return err
		}
		for _, opt := range options {
			if err := s.serviceRepo.SoftDeleteServiceOption(ctx, tx, opt.ServiceOptionID); err != nil {
				return err
			}
		}
		if err := s.serviceRepo.CascadeDeleteServiceSlots(ctx, tx, serviceID); err != nil {
			return err
		}
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
			var items []param.ServiceOptionItemParam
			if packages[j].IsRemoved {
				items, err = s.serviceRepo.GetServiceOptionItemsByOptionIDIncludeDeleted(ctx, packages[j].ServiceOptionID)
			} else {
				items, err = s.serviceRepo.GetServiceOptionItemsByOptionID(ctx, packages[j].ServiceOptionID)
			}
			if err != nil {
				return nil, err
			}
			packages[j].ServiceOptionItems = items
		}
		services[i].ServiceOptions = packages
	}
	return services, nil
}
