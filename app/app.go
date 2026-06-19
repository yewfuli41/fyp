package app

import (
	"database/sql"
	"fyp/config"
	"fyp/internal/interfaces"
	"fyp/internal/repository"
	"fyp/internal/service"
)

type App struct {
	AuthService     interfaces.IAuthService
	ProfileService  interfaces.IProfileService
	BusinessService interfaces.IBusinessService
	ServiceService  interfaces.IServiceService
	StaffService    interfaces.IStaffService
}

func NewApp(db *sql.DB, authConfig config.AuthConfig) *App {
	authRepo := repository.NewAuthRepo(db)
	authService := service.NewAuthService(authRepo, authConfig)
	profileRepo := repository.NewProfileRepo(db)
	profileService := service.NewProfileService(profileRepo)
	businessRepo := repository.NewBusinessRepo(db)
	businessService := service.NewBusinessService(db, businessRepo)
	serviceRepo := repository.NewServiceRepo(db)
	serviceService := service.NewServiceService(db, serviceRepo)
	staffRepo := repository.NewStaffRepo(db)
	staffService := service.NewStaffService(db, staffRepo)
	return &App{
		AuthService:     authService,
		ProfileService:  profileService,
		BusinessService: businessService,
		ServiceService:  serviceService,
		StaffService:    staffService,
	}
}
