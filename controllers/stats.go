package controllers

import (
	"absensi-backend/database"
	"absensi-backend/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func GetServerStats(c *gin.Context) {
	now := time.Now()
	endTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())

	startTime := endTime.Add(-1 * time.Hour) 

	var dbStats []models.TrafficStat
	database.DB.Where("timestamp >= ? AND timestamp <= ?", startTime.UnixMilli(), endTime.UnixMilli()).Order("timestamp asc").Find(&dbStats)

	statsMap := make(map[int64]models.TrafficStat)
	for _, s := range dbStats {
		statsMap[s.Timestamp] = s
	}

	var finalData []models.TrafficStat

	currentCheck := startTime

	for !currentCheck.After(endTime) {
		ts := currentCheck.UnixMilli()

		if val, exists := statsMap[ts]; exists {
			finalData = append(finalData, val)
		} else {
			finalData = append(finalData, models.TrafficStat{
				Timestamp: ts,
				GET:       0,
				POST:      0,
				PUT:       0,
				DELETE:    0,
			})
		}
		currentCheck = currentCheck.Add(1 * time.Minute)
	}
	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data":   finalData,
	})
}