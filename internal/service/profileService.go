package service

import (
	"context"
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

func (s *profileService) GetUserProfile(ctx context.Context, userID int64) (*param.ProfileParam, error) {
	return nil, nil
}
