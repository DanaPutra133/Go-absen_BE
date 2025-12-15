package routes

import (
	"absensi-backend/controllers"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	r.GET("/", controllers.GetSystemStatus)

	r.POST("/attend", controllers.RecordAttendance)

	r.GET("/sessions", controllers.GetSessions)
	r.GET("/streaks", controllers.GetStreaks)
	r.GET("/sessions/:id", controllers.GetSessionDetail)
	// r.GET("/sessions/streaks/:id", controllers.GetSessionDetail2)
	// r.GET("/sessions/streaks/2/:id", controllers.GetSessionDetail3)
	r.POST("/session/stop", controllers.StopSession)
	r.POST("/session/open", controllers.OpenSession)
	r.GET("/history", controllers.GetUserHistory)

	// Handle 404
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  false,
			"error":   "Endpoint Not Found",
			"message": "Maaf, URL yang Anda tuju tidak ada di server ini.",
			"path":    c.Request.URL.Path,
		})
	})

	// Handle 405 (Method Salah)
	r.HandleMethodNotAllowed = true
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"status":  false,
			"error":   "Method Not Allowed",
			"message": "URL benar, tapi method HTTP salah (Misal: Harusnya POST tapi Anda pakai GET).",
			"method":  c.Request.Method,
		})
	})
}
