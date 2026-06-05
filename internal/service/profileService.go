package service

import (
	"context"
	"fyp/database"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/interfaces"
)

type profileService struct {
	profileRepo interfaces.IProfileRepo
}

func NewProfileService(profileRepo interfaces.IProfileRepo) interfaces.IProfileService {
	return &profileService{
		profileRepo: profileRepo,
	}
}

func (s *profileService) UpdateProfile(ctx context.Context, param param.ProfileParam) error {
	if err := param.ValidateProfile(); err != nil {
		return err
	}
	if err := s.profileRepo.UpdateUser(ctx, param); err != nil {
		if database.IsUniqueViolation(err, "users_email_key") {
			return errs.ValidationErrors{
				{Field: "email", Message: "This email is already registered"},
			}
		}
		return err
	}
	return nil
}
