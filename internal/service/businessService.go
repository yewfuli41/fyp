package service

import (
	"context"
	"database/sql"
	"fmt"
	"fyp/database"
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

		err = s.businessRepo.InsertBusinessWorkingHours(ctx, tx, businessParam)
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
