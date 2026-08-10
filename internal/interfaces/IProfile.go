package interfaces

import (
	"context"
	"fyp/domain/param"
)

type IProfileRepo interface {
	UpdateUser(ctx context.Context, param param.ProfileParam) error
}

type IProfileService interface {
	UpdateProfile(ctx context.Context, param param.ProfileParam) error
}
