package main

import (
	"fyp/app"
	"fyp/config"
	"fyp/database"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env.local"); err != nil {
		log.Fatal("Error loading .env.local")
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatal("Config error", err)
	}

	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Database error", err)
	}
	defer db.Close()
	app := app.NewApp(db, cfg.Auth)

	server(app)
}
