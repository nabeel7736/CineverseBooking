package main

import (
	"fmt"
	"log"
	"os"

	"cineverse/config"
	"cineverse/models"
	"cineverse/routes"
	"cineverse/utils"

	"gorm.io/gorm"
)

func main() {
	config.ConnectDatabase()
	db := config.DB

	// migrate
	if err := migrate(db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	utils.SeedDummyTheatres()

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
	return db.AutoMigrate(&models.User{}, &models.Admin{}, &models.Movie{}, &models.Show{}, &models.Booking{}, &models.RefreshToken{},
		&models.Theatre{}, &models.Screen{}, &models.BookingSeat{}, &models.Payment{}, &models.Wishlist{})
}
