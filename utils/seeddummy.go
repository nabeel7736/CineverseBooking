package utils

import (
	"cineverse/config"
	"cineverse/models"
)

func SeedDummyTheatres() {
	theatres := []models.Theatre{

		{
			Name:                "Galaxy Cinemas",
			Location:            "Calicut",
			ParkingAvailable:    true,
			CarParkingFee:       50.00,
			BikeParkingFee:      20.00,
			CarParkingCapacity:  50,
			BikeParkingCapacity: 100,

			Screens: []models.Screen{
				{Name: "Screen 1", SeatsTotal: 100},
				{Name: "Screen 2", SeatsTotal: 120},
			},
		},
		{
			Name:                "Dreams Multiplex",
			Location:            "Malappuram",
			ParkingAvailable:    true,
			CarParkingFee:       40.00,
			BikeParkingFee:      15.00,
			CarParkingCapacity:  30,
			BikeParkingCapacity: 70,

			Screens: []models.Screen{
				{Name: "Screen 1", SeatsTotal: 90},
			},
		},
		{
			Name:                "CineVerse Theatre",
			Location:            "Manjeri",
			ParkingAvailable:    true,
			CarParkingFee:       30.00,
			BikeParkingFee:      10.00,
			CarParkingCapacity:  20,
			BikeParkingCapacity: 50,

			Screens: []models.Screen{
				{Name: "Screen 1", SeatsTotal: 60},
				{Name: "Screen 2", SeatsTotal: 80},
			},
		},
	}

	for _, t := range theatres {
		var existing models.Theatre
		err := config.DB.Where("name = ?", t.Name).First(&existing).Error

		if err == nil {
			existing.Location = t.Location
			existing.ParkingAvailable = t.ParkingAvailable
			existing.CarParkingFee = t.CarParkingFee
			existing.BikeParkingFee = t.BikeParkingFee
			existing.CarParkingCapacity = t.CarParkingCapacity
			existing.BikeParkingCapacity = t.BikeParkingCapacity
			config.DB.Save(&existing)

			for _, screen := range t.Screens {
				var existingScreen models.Screen

				if config.DB.Where("theatre_id =? AND name = ?", existing.ID, screen.Name).First(&existingScreen).Error == nil {
					existingScreen.SeatsTotal = screen.SeatsTotal
					config.DB.Save(&existingScreen)
				} else {
					screen.TheatreID = existing.ID
					config.DB.Create(&screen)
				}
			}
		} else {
			config.DB.Create(&t)
		}

	}
}
