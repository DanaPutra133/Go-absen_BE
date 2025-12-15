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

// Helper untuk zona waktu
func getJakartaDateStr() string {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*60*60)
	}
	return time.Now().In(loc).Format("2006-01-02")
}

// ==========================================
// 1. API KHUSUS ADMIN / BOT (CONTROL SESSION)
// ==========================================

func StopSession(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter session_id wajib diisi (contoh: ?session_id=123)"})
		return
	}
	todayStr := getJakartaDateStr()
	realID := sessionID + "-" + todayStr
	var session models.Session
	if err := database.DB.First(&session, "id = ?", realID).Error; err != nil {
		c.JSON(404, gin.H{"error": "Sesi hari ini belum dibuat/belum ada yang absen."})
		return
	}
	if !session.IsActive {
		c.JSON(400, gin.H{"message": "Sesi absensi SUDAH DITUTUP sebelumnya. Tidak ada perubahan."})
		return
	}
	session.IsActive = false
	database.DB.Save(&session)

	c.JSON(http.StatusOK, gin.H{
		"message":       "Absensi BERHASIL DITUTUP (Paused). User tidak bisa absen sementara.",
		"session_db_id": realID,
	})
}

func OpenSession(c *gin.Context) {
	sessionID := c.Query("session_id")

	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter session_id wajib diisi"})
		return
	}

	todayStr := getJakartaDateStr()
	realID := sessionID + "-" + todayStr
	var session models.Session
	if err := database.DB.First(&session, "id = ?", realID).Error; err != nil {
		c.JSON(404, gin.H{"error": "Tidak ada sesi hari ini untuk dibuka kembali."})
		return
	}
	if session.IsActive {
		c.JSON(400, gin.H{"message": "Sesi absensi SEDANG BERJALAN (Sudah Terbuka)."})
		return
	}
	session.IsActive = true
	database.DB.Save(&session)

	c.JSON(http.StatusOK, gin.H{
		"message":       "Absensi DIBUKA KEMBALI (Resumed). Silakan lanjut absen.",
		"session_db_id": realID,
	})
}

// ==========================================
// 2. API USER (ABSENSI UTAMA)
// ==========================================

func RecordAttendance(c *gin.Context) {
	var req models.AttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	loc, errLoc := time.LoadLocation("Asia/Jakarta")
	if errLoc != nil {
		loc = time.FixedZone("WIB", 7*60*60)
	}

	nowJakarta := time.Now().In(loc)

	if req.Timestamp.IsZero() {
		req.Timestamp = nowJakarta
	}
	todayDateStr := req.Timestamp.In(loc).Format("2006-01-02")
	realSessionID := req.SessionID + "-" + todayDateStr

	tx := database.DB.Begin()
	var session models.Session
	if err := tx.First(&session, "id = ?", realSessionID).Error; err != nil {
		session = models.Session{
			ID:        realSessionID, // Simpan ID Unik Harian
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
	} else {
		if !session.IsActive {
			tx.Rollback()
			c.JSON(403, gin.H{"message": "Absensi hari ini sudah ditutup Admin!"})
			return
		}
	}
	var existingAttendee models.Attendee
	checkAtt := tx.Where("session_id = ? AND user_id = ?", realSessionID, req.UserID).First(&existingAttendee)

	if checkAtt.RowsAffected > 0 {
		tx.Rollback()
		c.JSON(409, gin.H{"message": "User sudah absen hari ini!"})
		return
	}
	newAttendee := models.Attendee{
		SessionID: realSessionID,
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

		if currentDate.Format("2006-01-02") == streak.LastAttendance {
			newStreakCount = streak.CurrentStreak
		} else if currentDate.Day() == 1 {
			newStreakCount = 1
		} else {
			daysDiff := currentDate.Sub(lastAttDate).Hours() / 24
			if daysDiff >= 0.5 && daysDiff < 1.5 {
				newStreakCount = streak.CurrentStreak + 1
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
		"session_id":      req.SessionID,
		"session_db_id":   realSessionID,
	})
}

// ==========================================
// 3. API PENDUKUNG (READ DATA)
// ==========================================

func GetStreaks(c *gin.Context) {
	guildID := c.Query("guild_id")
	userID := c.Query("user_id")
	sessionID := c.Query("session_id")

	var streaks []models.Streak

	query := database.DB.Table("streaks")

	if sessionID != "" {
		if len(sessionID) < 15 {
			todayStr := getJakartaDateStr()
			sessionID = sessionID + "-" + todayStr
		}
		// --------------------------------------------------
		query = query.Joins("JOIN attendees ON attendees.user_id = streaks.user_id").
			Where("attendees.session_id = ?", sessionID)

		if guildID == "" {
			query = query.Where("streaks.guild_id = (SELECT guild_id FROM sessions WHERE id = ?)", sessionID)
		}
	}

	if guildID != "" {
		query = query.Where("streaks.guild_id = ?", guildID)
	}
	if userID != "" {
		query = query.Where("streaks.user_id = ?", userID)
	}
	if err := query.Find(&streaks).Error; err != nil {
		c.JSON(500, gin.H{"error": "Gagal mengambil data streak"})
		return
	}

	if len(streaks) == 0 {
		c.JSON(404, gin.H{"message": "Data streak tidak ditemukan untuk kriteria ini"})
		return
	}

	c.JSON(http.StatusOK, streaks)
}

type AttendeeResponse struct {
	UserID        string    `json:"user_id"`
	Timestamp     time.Time `json:"timestamp"`
	CurrentStreak int       `json:"current_streak"`
}

func GetSessionDetail2(c *gin.Context) {
	channelID := c.Param("id")
	sessionID := c.Param("id")
	var session models.Session
	if err := database.DB.Where("id = ?", sessionID).First(&session).Error; err != nil {
		c.JSON(404, gin.H{"error": "Session ID tidak ditemukan"})
		return
	}
	var attendees []AttendeeResponse

	err := database.DB.Table("attendees").
		Select("attendees.user_id, attendees.timestamp, streaks.current_streak").
		Joins("LEFT JOIN streaks ON streaks.user_id = attendees.user_id AND streaks.guild_id = ?", session.GuildID).
		Where("attendees.session_id = ?", sessionID).
		Scan(&attendees).Error

	if err != nil {
		c.JSON(500, gin.H{"error": "Gagal mengambil data peserta"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_info": gin.H{
			"session_id":    channelID,
			"session_db_id": session.ID,
			"guild_id":      session.GuildID,
			"reason":        session.Reason,
			"startTime":     session.StartTime,
			"is_active":     session.IsActive,
		},
		"total_hadir": len(attendees),
		"attendees":   attendees,
	})
}

func GetSessionDetail3(c *gin.Context) {
	channelID := c.Param("id")
	todayStr := time.Now().Format("2006-01-02")
	targetSessionID := channelID + "-" + todayStr

	var session models.Session
	if err := database.DB.Where("id = ?", targetSessionID).First(&session).Error; err != nil {
		c.JSON(404, gin.H{
			"error": "Belum ada sesi absensi untuk hari ini (" + todayStr + ")",
			"session_info": gin.H{
				"session_id": channelID,
				"is_active":  false,
			},
			"total_hadir": 0,
			"attendees":   []string{},
		})
		return
	}

	var attendees []AttendeeResponse
	database.DB.Table("attendees").
		Select("attendees.user_id, attendees.timestamp, streaks.current_streak").
		Joins("LEFT JOIN streaks ON streaks.user_id = attendees.user_id AND streaks.guild_id = ?", session.GuildID).
		Where("attendees.session_id = ?", targetSessionID).
		Scan(&attendees)

	c.JSON(http.StatusOK, gin.H{
		"session_info": gin.H{
			"session_id":    channelID,
			"session_db_id": session.ID,
			"guild_id":      session.GuildID,
			"reason":        session.Reason,
			"startTime":     session.StartTime,
			"is_active":     session.IsActive,
		},
		"total_hadir": len(attendees),
		"attendees":   attendees,
	})
}

func GetSessionDetail(c *gin.Context) {
	inputID := c.Param("id")
	var session models.Session
	var finalID string

	if err := database.DB.Where("id = ?", inputID).First(&session).Error; err == nil {
		finalID = session.ID
	} else {
		todayStr := getJakartaDateStr()
		targetID := inputID + "-" + todayStr

		if err2 := database.DB.Where("id = ?", targetID).First(&session).Error; err2 == nil {
			finalID = session.ID
		} else {
			c.JSON(404, gin.H{
				"error": "Sesi tidak ditemukan.",
				"session_info": gin.H{
					"session_id": inputID,
					"is_active":  false,
				},
				"total_hadir": 0,
				"attendees":   []string{},
			})
			return
		}
	}

	// --- AMBIL DATA PESERTA + STREAK (JOIN) ---
	var attendees []AttendeeResponse

	err := database.DB.Table("attendees").
		Select("attendees.user_id, attendees.timestamp, streaks.current_streak").
		Joins("LEFT JOIN streaks ON streaks.user_id = attendees.user_id AND streaks.guild_id = ?", session.GuildID).
		Where("attendees.session_id = ?", finalID). // Pakai ID yang ketemu tadi
		Scan(&attendees).Error

	if err != nil {
		c.JSON(500, gin.H{"error": "Gagal mengambil data peserta"})
		return
	}

	realChannelID := session.ID
	if len(session.ID) > 11 {
		realChannelID = session.ID[:len(session.ID)-11]
	}

	c.JSON(http.StatusOK, gin.H{
		"session_info": gin.H{
			"session_id":    realChannelID,
			"session_db_id": session.ID,
			"guild_id":      session.GuildID,
			"reason":        session.Reason,
			"startTime":     session.StartTime,
			"is_active":     session.IsActive,
		},
		"total_hadir": len(attendees),
		"attendees":   attendees,
	})
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

type SessionListResponse struct {
	SessionID   string            `json:"session_id"`
	SessionDBID string            `json:"session_db_id"`
	GuildID     string            `json:"guild_id"`
	Reason      string            `json:"reason"`
	StartTime   time.Time         `json:"startTime"`
	IsActive    bool              `json:"is_active"`
	Attendees   []models.Attendee `json:"attendees"`
}

func GetSessions(c *gin.Context) {
	var sessions []models.Session
	if err := database.DB.Preload("Attendees").Find(&sessions).Error; err != nil {
		c.JSON(500, gin.H{"error": "Gagal mengambil data sesi"})
		return
	}

	var response []SessionListResponse

	for _, s := range sessions {

		realID := s.ID
		if len(s.ID) > 11 {
			realID = s.ID[:len(s.ID)-11]
		}

		data := SessionListResponse{
			SessionID:   realID,
			SessionDBID: s.ID,
			GuildID:     s.GuildID,
			Reason:      s.Reason,
			StartTime:   s.StartTime,
			IsActive:    s.IsActive,
			Attendees:   s.Attendees,
		}

		response = append(response, data)
	}
	c.JSON(http.StatusOK, response)
}
