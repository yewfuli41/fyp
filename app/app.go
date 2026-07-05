package app

import (
	"database/sql"
	"fyp/config"
	"fyp/internal/interfaces"
	"fyp/internal/repository"
	"fyp/internal/service"
)

type App struct {
	AuthService        interfaces.IAuthService
	ProfileService     interfaces.IProfileService
	BusinessService    interfaces.IBusinessService
	ServiceService     interfaces.IServiceService
	StaffService       interfaces.IStaffService
	ServiceSlotService interfaces.IServiceSlotService
	EmailService       interfaces.IEmailService
}

func NewApp(db *sql.DB, cfg *config.Config) *App {
	authRepo := repository.NewAuthRepo(db)
	authService := service.NewAuthService(authRepo, cfg.Auth)
	profileRepo := repository.NewProfileRepo(db)
	profileService := service.NewProfileService(profileRepo)
	businessRepo := repository.NewBusinessRepo(db)
	businessService := service.NewBusinessService(db, businessRepo)
	serviceRepo := repository.NewServiceRepo(db)
	serviceService := service.NewServiceService(db, serviceRepo)
	emailService := service.NewSendGridEmailService(cfg.Email)
	staffRepo := repository.NewStaffRepo(db)
	staffService := service.NewStaffService(db, staffRepo, businessRepo, authRepo, emailService)
	serviceSlotRepo := repository.NewServiceSlotRepo(db)
	serviceSlotService := service.NewServiceSlotService(db, serviceSlotRepo, cfg.ServiceSlot)
	return &App{
		AuthService:        authService,
		ProfileService:     profileService,
		BusinessService:    businessService,
		ServiceService:     serviceService,
		StaffService:       staffService,
		ServiceSlotService: serviceSlotService,
		EmailService:       emailService,
	}
}
