package config

import (
	"cineverse/models"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}
}

func ConnectDatabase() {
	LoadEnv()

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	DB = db
}

func MigrateAll() {
	// models will be imported in main before calling MigrateAll
	err := DB.AutoMigrate(
		&models.User{},
		&models.Movie{},
		&models.Show{},
		&models.Booking{},
		&models.RefreshToken{},
	)
	if err != nil {
		fmt.Println("Migration Failed", err)
		return
	}
	fmt.Println("Migrated Successfully !!")
}
