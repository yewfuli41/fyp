package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"fyp/database"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"fyp/utils"
	"sort"
	"strings"
	"time"
)

type businessService struct {
	businessRepo    interfaces.IBusinessRepo
	serviceSlotRepo interfaces.IServiceSlotRepo
	staffRepo       interfaces.IStaffRepo
	tx              *database.Transaction
}

func NewBusinessService(db *sql.DB, businessRepo interfaces.IBusinessRepo, serviceSlotRepo interfaces.IServiceSlotRepo, staffRepo interfaces.IStaffRepo) interfaces.IBusinessService {
	return &businessService{
		businessRepo:    businessRepo,
		serviceSlotRepo: serviceSlotRepo,
		staffRepo:       staffRepo,
		tx:              database.NewTransaction(db),
	}
}

// validateSlotsWithinNewWorkingHours checks that every existing owner-managed
// slot window is still covered by the NEW proposed working hours on its
// weekday — otherwise the edit would silently strand a slot (bookable or
// already booked) outside business hours, with no check anywhere else ever
// catching it after the fact.
func validateSlotsWithinNewWorkingHours(windows []param.SlotWindowParam, newHours []param.WorkingHourParam) error {
	byDay := make(map[string][]param.WorkingHourParam)
	for _, wh := range newHours {
		byDay[wh.Day] = append(byDay[wh.Day], wh)
	}

	seen := make(map[string]bool)
	var offendingDates []string
	weekdayByDate := make(map[string]string)
	for _, w := range windows {
		weekday, err := weekdayOf(w.Date)
		if err != nil {
			continue
		}
		wStart := timeOfDay(w.StartTime)
		wEnd := timeOfDay(w.EndTime)
		covered := false
		for _, bwh := range byDay[weekday] {
			bStart := timeOfDay(bwh.StartTime)
			bEnd := timeOfDay(bwh.EndTime)
			if !wStart.Before(bStart) && !wEnd.After(bEnd) {
				covered = true
				break
			}
		}
		if !covered && !seen[w.Date] {
			seen[w.Date] = true
			offendingDates = append(offendingDates, w.Date)
			weekdayByDate[w.Date] = weekday
		}
	}
	if len(offendingDates) == 0 {
		return nil
	}
	sort.Strings(offendingDates)
	labels := make([]string, len(offendingDates))
	for i, date := range offendingDates {
		labels[i] = fmt.Sprintf("%s (%s)", date, utils.CapitalizeFirst(weekdayByDate[date]))
	}
	return errs.ValidationErrors{{
		Field:   "workingHours",
		Message: fmt.Sprintf("Can't update working hours — existing slots fall outside the new hours on: %s.", strings.Join(labels, ", ")),
	}}
}

func (s *businessService) RegisterBusinessProfile(ctx context.Context, businessParam param.BusinessProfileParam) (*param.BusinessProfileParam, error) {
	// A staff member is already tied to a business as an employee — they
	// can't also register as that (or another) business's owner.
	if _, err := s.staffRepo.GetStaffByUserID(ctx, businessParam.OwnerUserID); err == nil {
		return nil, fmt.Errorf("You are already registered as a staff member and cannot also register a business.")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	var businessProfile *param.BusinessProfileParam
	err := s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		if err := businessParam.ValidateRegisterBusinessProfile(); err != nil {
			return err
		}

		createdBusiness, err := s.businessRepo.InsertBusinessProfile(ctx, tx, businessParam)
		if err != nil {
			return err
		}

		createdBusiness.WorkingHours = businessParam.WorkingHours
		businessProfile = createdBusiness

		err = s.businessRepo.InsertBusinessWorkingHours(ctx, tx, *createdBusiness)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if database.IsUniqueViolation(err, "business_profiles_owner_user_id_key") {
			return nil, fmt.Errorf("You already registered a business profile")
		} else if database.IsForeignKeyViolation(err, "business_profiles_owner_user_id_fkey") {
			return nil, fmt.Errorf("Unable to register business profile. Please log in again.")
		}
		return nil, err
	}

	return businessProfile, nil
}

func (s *businessService) UpdateBusinessProfile(ctx context.Context, businessParam param.BusinessProfileParam) (*param.BusinessProfileParam, error) {
	var businessProfile *param.BusinessProfileParam
	err := s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		if err := businessParam.ValidateRegisterBusinessProfile(); err != nil {
			return err
		}

		updatedBusiness, err := s.businessRepo.UpdateBusinessProfile(ctx, tx, businessParam)
		if err != nil {
			return err
		}

		today := time.Now().Format("2006-01-02")
		windows, err := s.serviceSlotRepo.GetFutureUnassignedSlotWindows(ctx, updatedBusiness.BusinessID, today)
		if err != nil {
			return err
		}
		if err := validateSlotsWithinNewWorkingHours(windows, businessParam.WorkingHours); err != nil {
			return err
		}

		if err := s.businessRepo.DeleteBusinessWorkingHours(ctx, tx, updatedBusiness.BusinessID); err != nil {
			return err
		}

		businessParam.BusinessID = updatedBusiness.BusinessID
		if err := s.businessRepo.InsertBusinessWorkingHours(ctx, tx, businessParam); err != nil {
			return err
		}

		updatedBusiness.WorkingHours = businessParam.WorkingHours
		businessProfile = updatedBusiness
		return nil
	})
	if err != nil {
		return nil, err
	}

	return businessProfile, nil
}

func (s *businessService) GetBusinesses(ctx context.Context, search string) ([]param.BusinessProfileParam, error) {
	return s.businessRepo.GetBusinesses(ctx, search)
}

func (s *businessService) GetBusinessProfileByOwnerID(ctx context.Context, ownerID int64) (*param.BusinessProfileParam, error) {
	businessProfile, err := s.businessRepo.GetBusinessProfileByOwnerID(ctx, ownerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrBusinessProfileNotFound
		}
		return nil, err
	}

	workingHours, err := s.businessRepo.GetBusinessWorkingHours(ctx, businessProfile.BusinessID)
	if err != nil {
		return nil, err
	}

	businessProfile.WorkingHours = workingHours
	return businessProfile, nil
}
