package middleware

import (
	"absensi-backend/database"
	"absensi-backend/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TrafficLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/stats" {
			c.Next()
			return
		}

		now := time.Now()
		roundedTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
		timestampMilli := roundedTime.UnixMilli()

		method := c.Request.Method
		incGet, incPost, incPut, incDel := 0, 0, 0, 0

		switch method {
		case "GET":
			incGet = 1
		case "POST":
			incPost = 1
		case "PUT":
			incPut = 1
		case "DELETE":
			incDel = 1
		}

		stats := models.TrafficStat{
			Timestamp: timestampMilli,
			GET:       incGet,
			POST:      incPost,
			PUT:       incPut,
			DELETE:    incDel,
		}

		database.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "timestamp"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"get":        gorm.Expr("get + ?", incGet),
				"post":       gorm.Expr("post + ?", incPost),
				"put":        gorm.Expr("put + ?", incPut),
				"delete_req": gorm.Expr("delete_req + ?", incDel),
			}),
		}).Create(&stats)

		c.Next()
	}
}