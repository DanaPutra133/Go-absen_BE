package routes

import (
	"absensi-backend/controllers"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "Absensi Backend Running"})
	})

	r.POST("/attend", controllers.RecordAttendance)

	r.GET("/sessions", controllers.GetSessions)
	r.GET("/streaks", controllers.GetStreaks)
	r.GET("/sessions/:id", controllers.GetSessionDetail) 
	r.POST("/session/stop", controllers.StopSession)
	r.POST("/session/open", controllers.OpenSession)
	r.GET("/history", controllers.GetUserHistory)
}