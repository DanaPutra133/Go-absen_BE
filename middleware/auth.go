package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientKey := c.GetHeader("x-api-key")

		serverKey := os.Getenv("API_KEY")
		if clientKey == "" || clientKey != serverKey {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Akses Ditolak: API Key Salah atau Tidak Ada",
			})
			c.Abort() 
			return
		}
		c.Next()
	}
}