package main

import (
	"absensi-backend/database"
	"absensi-backend/middleware" 
	"absensi-backend/routes"
	"absensi-backend/controllers"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// Koneksi Database
	database.ConnectDatabase()

	
	r := gin.Default()
	r.SetTrustedProxies(nil)
	r.Use(middleware.TrafficLogger())
	r.GET("/stats", controllers.GetServerStats)
	r.Use(middleware.AuthMiddleware())
	routes.SetupRoutes(r)

	port := os.Getenv("PORT")
	r.Run(":" + port)
}