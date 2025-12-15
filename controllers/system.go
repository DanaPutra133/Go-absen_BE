package controllers

import (
	"absensi-backend/database"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

var AppStartTime = time.Now()

func GetSystemStatus(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	sqlDB, err := database.DB.DB()
	dbStatus := "Connected"
	openConns := 0

	if err != nil {
		dbStatus = "Error Connection"
	} else {
		if errPing := sqlDB.Ping(); errPing != nil {
			dbStatus = "Disconnected / Timeout"
		}
		openConns = sqlDB.Stats().OpenConnections
	}

	uptime := time.Since(AppStartTime)

	c.JSON(http.StatusOK, gin.H{
		"app_name": "Backend Absen V0.1",
		"status":   "Running",

		"server_info": gin.H{
			"uptime":          uptime.String(),
			"start_time":      AppStartTime.Format(time.RFC1123),
			"goroutines":      runtime.NumGoroutine(),
			"memory_usage_mb": m.Alloc / 1024 / 1024,
			"total_alloc_mb":  m.TotalAlloc / 1024 / 1024,
			"sys_memory_mb":   m.Sys / 1024 / 1024,
			"garbage_collect": m.NumGC,
		},
		"database_info": gin.H{
			"status":           dbStatus,
			"type":             "SQLite",
			"open_connections": openConns,
		},
		"time_check": gin.H{
			"server_time_raw": time.Now().String(),
			"jakarta_time":    getJakartaDateStr() + " (Date Only)",
		},
	})
}
