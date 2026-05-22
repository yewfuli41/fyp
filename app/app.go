package app

import (
	"database/sql"
	"fyp/config"
	"fyp/internal/interfaces"
	"fyp/internal/repository"
	"fyp/internal/service"
)

type App struct {
	AuthService interfaces.IAuthService
}

func NewApp(db *sql.DB, authConfig config.AuthConfig) *App {
	authRepo := repository.NewAuthRepo(db)
	authService := service.NewAuthService(authRepo, authConfig)
	return &App{
		AuthService: authService,
	}
}
