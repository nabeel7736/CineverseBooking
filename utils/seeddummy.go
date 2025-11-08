package utils

import (
	"cineverse/config"
	"cineverse/models"
)

func SeedDummyTheatres() {
	theatres := []models.Theatre{

		{
			Name:     "Galaxy Cinemas",
			Location: "Calicut",
			Screens: []models.Screen{
				{Name: "Screen 1", SeatsTotal: 100},
				{Name: "Screen 2", SeatsTotal: 120},
			},
		},
		{
			Name:     "Dreams Multiplex",
			Location: "Malappuram",
			Screens: []models.Screen{
				{Name: "Screen 1", SeatsTotal: 90},
			},
		},
		{
			Name:     "CineVerse Theatre",
			Location: "Manjeri",
			Screens: []models.Screen{
				{Name: "Screen 1", SeatsTotal: 60},
				{Name: "Screen 2", SeatsTotal: 80},
			},
		},
	}

	for _, t := range theatres {
		var existing models.Theatre
		if err := config.DB.Where("name = ?", t.Name).First(&existing).Error; err != nil {
			config.DB.Create(&t)
		}
	}
}
