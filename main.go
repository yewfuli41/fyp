package main

import (
	"fyp/app"
	"fyp/config"
	"fyp/database"
	"log"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env.local"); err != nil {
		log.Fatal("Error loading .env.local")
	}

	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatal("Config error: ", err)
	}

	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Database error: ", err)
	}
	defer db.Close()
	app := app.NewApp(db, cfg.Auth)

	server(app, cfg)
}
