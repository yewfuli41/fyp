package main

import (
	"fyp/database"
	"log"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env.local"); err != nil {
		log.Fatal("Error loading .env.local")
	}

	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Database error", err)
	}
	defer db.Close()

	server()
}
