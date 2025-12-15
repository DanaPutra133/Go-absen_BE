package services

import (
	"log"
	"time"
)

func LogSessionAccessAsync(
	sessionID string,
	guildID string,
	found bool,
) {
	status := "NOT_FOUND"
	if found {
		status = "FOUND"
	}

	log.Printf(
		"[SESSION_ACCESS] session=%s guild=%s status=%s time=%s",
		sessionID,
		guildID,
		status,
		time.Now().Format(time.RFC3339),
	)
}
