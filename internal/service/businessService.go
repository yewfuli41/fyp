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
)

type businessService struct {
	businessRepo interfaces.IBusinessRepo
	tx           *database.Transaction
}

func NewBusinessService(db *sql.DB, businessRepo interfaces.IBusinessRepo) interfaces.IBusinessService {
	return &businessService{
		businessRepo: businessRepo,
		tx:           database.NewTransaction(db),
	}
}

func (s *businessService) RegisterBusinessProfile(ctx context.Context, businessParam param.BusinessProfileParam) (*param.BusinessProfileParam, error) {
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
