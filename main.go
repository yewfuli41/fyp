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

	/*from := mail.NewEmail(cfg.Email.FromName, cfg.Email.FromEmail)
	to := mail.NewEmail("", "yewfuli41@gmail.com")
	message := mail.NewSingleEmail(from, "SendGrid Test", to, "i love u", "<p>i love u</p>")

	client := sendgrid.NewSendClient(cfg.Email.SendGridAPIKey)

	resp, err := client.Send(message)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Status:", resp.StatusCode)
	log.Println("Body:", resp.Body)*/

	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Database error: ", err)
	}
	defer db.Close()
	app := app.NewApp(db, cfg)

	server(app, cfg)
}
