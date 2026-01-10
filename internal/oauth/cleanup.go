package oauth

import (
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/models"
)

// StartCleanupJob starts a background job to clean up expired OAuth sessions and linking tokens
func StartCleanupJob(db *gorm.DB) {
	log.Println("OAuth: Starting cleanup job (runs every 10 minutes)")

	// Run cleanup immediately on start
	go cleanupExpiredRecords(db)

	// Then run every 10 minutes
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		for range ticker.C {
			cleanupExpiredRecords(db)
		}
	}()
}

func cleanupExpiredRecords(db *gorm.DB) {
	now := time.Now()

	// Clean up expired OAuth sessions
	result := db.Where("expires_at < ?", now).Delete(&models.OAuthSession{})
	if result.Error != nil {
		log.Println("OAuth cleanup: Failed to delete expired sessions:", result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("OAuth cleanup: Deleted %d expired sessions", result.RowsAffected)
	}

	// Clean up expired linking tokens
	result = db.Where("expires_at < ?", now).Delete(&models.OAuthLinkingToken{})
	if result.Error != nil {
		log.Println("OAuth cleanup: Failed to delete expired linking tokens:", result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("OAuth cleanup: Deleted %d expired linking tokens", result.RowsAffected)
	}
}
