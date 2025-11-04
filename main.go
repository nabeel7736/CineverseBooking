package main

import (
	"fmt"
	"log"
	"os"

	"cineverse/config"
	"cineverse/models"
	"cineverse/routes"

	"gorm.io/gorm"
)

func main() {
	config.ConnectDatabase()
	db := config.DB

	// migrate
	if err := migrate(db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	r := routes.SetupRouter()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf(":%s", port)
	log.Printf("server running on %s", addr)
	r.Run(addr)
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(&models.User{}, &models.Movie{}, &models.Show{}, &models.Booking{}, &models.RefreshToken{},
		&models.Theatre{}, &models.Hall{}, &models.Seat{}, &models.BookingSeat{}, &models.Payment{})
}
