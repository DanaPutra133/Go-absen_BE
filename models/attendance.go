package models

import "time"

// --- Tabel Database ---
type Session struct {
	ID        string     `gorm:"primaryKey" json:"session_id"`
	GuildID   string     `json:"guild_id"`
	Reason    string     `json:"reason"`
	StartTime time.Time  `json:"startTime"`
	IsActive  bool       `json:"is_active" gorm:"default:true"`
	Attendees []Attendee `gorm:"foreignKey:SessionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"attendees"`
}

type Attendee struct {
	ID        uint      `gorm:"primaryKey" json:"row_id"`
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}

type Streak struct {
	ID             uint   `gorm:"primaryKey" json:"row_id"`
	GuildID        string `json:"guild_id"`
	UserID         string `json:"user_id"`  
	CurrentStreak  int    `json:"currentStreak"`
	LastAttendance string `json:"lastAttendance"` // YYYY-MM-DD
}

type AttendanceRequest struct {
	GuildID   string    `json:"guild_id" binding:"required"`
	UserID    string    `json:"user_id" binding:"required"`
	SessionID string    `json:"session_id" binding:"required"` 
	Timestamp time.Time `json:"timestamp"` 
	Reason    string    `json:"reason"`    
}


type TrafficStat struct {
	Timestamp int64 `gorm:"primaryKey" json:"timestamp"` 
	GET       int   `json:"GET"`
	POST      int   `json:"POST"`
	PUT       int   `json:"PUT"`
	DELETE    int   `json:"DELETE" gorm:"column:delete_req"`
}