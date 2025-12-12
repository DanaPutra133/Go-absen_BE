package controllers

import (
	"absensi-backend/database"
	"absensi-backend/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Helper untuk parsing tanggal YYYY-MM-DD
func parseDate(dateStr string) time.Time {
	t, _ := time.Parse("2006-01-02", dateStr)
	return t
}

// ==========================================
// 1. API KHUSUS ADMIN / BOT (CONTROL SESSION)
// ==========================================

// POST: Stop Absensi (!absen stop)
func StopSession(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var session models.Session
	if err := database.DB.First(&session, "id = ?", req.SessionID).Error; err != nil {
		c.JSON(404, gin.H{"error": "Session tidak ditemukan"})
		return
	}

	session.IsActive = false
	database.DB.Save(&session)

	c.JSON(http.StatusOK, gin.H{"message": "Absensi berhasil ditutup."})
}

// POST: Buka Kembali Absensi (!absen open)
func OpenSession(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var session models.Session
	if err := database.DB.First(&session, "id = ?", req.SessionID).Error; err != nil {
		c.JSON(404, gin.H{"error": "Session tidak ditemukan"})
		return
	}

	session.IsActive = true
	database.DB.Save(&session)

	c.JSON(http.StatusOK, gin.H{"message": "Absensi berhasil dibuka kembali."})
}

// ==========================================
// 2. API USER (ABSENSI UTAMA)
// ==========================================

func RecordAttendance(c *gin.Context) {
	var req models.AttendanceRequest

	// Validasi JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set waktu default
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}
	todayDateStr := req.Timestamp.Format("2006-01-02")
	
	tx := database.DB.Begin()

	var session models.Session
	if err := tx.First(&session, "id = ?", req.SessionID).Error; err == nil {

		if !session.IsActive {
			tx.Rollback()
			c.JSON(403, gin.H{"message": "Absensi sudah ditutup oleh Admin!"})
			return
		}
	} else {
		session = models.Session{
			ID:        req.SessionID,
			GuildID:   req.GuildID,
			Reason:    req.Reason,
			StartTime: req.Timestamp,
			IsActive:  true,
		}
		if err := tx.Create(&session).Error; err != nil {
			tx.Rollback()
			c.JSON(500, gin.H{"error": "Gagal buat sesi: " + err.Error()})
			return
		}
	}
	var existingAttendee models.Attendee
	checkAtt := tx.Where("session_id = ? AND user_id = ?", req.SessionID, req.UserID).First(&existingAttendee)
	
	if checkAtt.RowsAffected > 0 {
		tx.Rollback()
		c.JSON(409, gin.H{"message": "User sudah absen di sesi ini!"})
		return
	}
	newAttendee := models.Attendee{
		SessionID: req.SessionID,
		UserID:    req.UserID,
		Timestamp: req.Timestamp,
	}
	if err := tx.Create(&newAttendee).Error; err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Gagal simpan attendee"})
		return
	}
	var streak models.Streak
	err := tx.Where("guild_id = ? AND user_id = ?", req.GuildID, req.UserID).First(&streak).Error

	newStreakCount := 1 

	if err == gorm.ErrRecordNotFound {
		streak = models.Streak{
			GuildID:        req.GuildID,
			UserID:         req.UserID,
			CurrentStreak:  1,
			LastAttendance: todayDateStr,
		}
		tx.Create(&streak)
	} else {
		lastAttDate := parseDate(streak.LastAttendance)
		currentDate := parseDate(todayDateStr)

		if currentDate.Day() == 1 {
			newStreakCount = 1
		} else {
			daysDiff := currentDate.Sub(lastAttDate).Hours() / 24

			if daysDiff >= 0.5 && daysDiff < 1.5 {
				
				newStreakCount = streak.CurrentStreak + 1
			} else if daysDiff < 0.5 {
				
				newStreakCount = streak.CurrentStreak
			} else {
				newStreakCount = 1
			}
		}
		streak.CurrentStreak = newStreakCount
		streak.LastAttendance = todayDateStr
		tx.Save(&streak)
	}

	tx.Commit()

	c.JSON(200, gin.H{
		"message":         "Absensi Berhasil",
		"streak_total":    streak.CurrentStreak,
		"last_attendance": todayDateStr,
		"guild_id":        req.GuildID,
		"user_id":         req.UserID,
		"reason":          req.Reason,
	})
}

// ==========================================
// 3. API PENDUKUNG (READ DATA)
// ==========================================

func GetStreaks(c *gin.Context) {
	guildID := c.Query("guild_id")
	userID := c.Query("user_id")

	var streaks []models.Streak
	query := database.DB

	if guildID != "" { query = query.Where("guild_id = ?", guildID) }
	if userID != "" { query = query.Where("user_id = ?", userID) }

	query.Find(&streaks)
	if len(streaks) == 0 {
		c.JSON(404, gin.H{"message": "Data streak tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, streaks)
}

func GetSessionDetail(c *gin.Context) {
	sessionID := c.Param("id")
	var session models.Session
	err := database.DB.Preload("Attendees").First(&session, "id = ?", sessionID).Error
	if err != nil {
		c.JSON(404, gin.H{"error": "Session ID tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"session_info": session, "total_hadir": len(session.Attendees)})
}

func GetUserHistory(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(400, gin.H{"error": "Parameter user_id wajib diisi"})
		return
	}
	var history []models.Attendee
	database.DB.Where("user_id = ?", userID).Order("timestamp desc").Find(&history)
	c.JSON(http.StatusOK, history)
}

func GetSessions(c *gin.Context) {
	var sessions []models.Session
	database.DB.Preload("Attendees").Find(&sessions)
	c.JSON(http.StatusOK, sessions)
}