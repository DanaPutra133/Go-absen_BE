package database

import (
	"absensi-backend/models"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	database, err := gorm.Open(sqlite.Open("absensi.db"), &gorm.Config{})
	if err != nil {
		panic("Gagal koneksi ke database!")
	}

	// Membuat tabel otomatis
	err = database.AutoMigrate(
		&models.Session{}, 
		&models.Attendee{}, 
		&models.Streak{}, 
		&models.TrafficStat{})
	if err != nil {
		log.Fatal("Gagal migrasi database:", err)
	}

	DB = database
}