package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/models"
	"github.com/fuomag9/uptime-kabomba/internal/monitor"
)

// HandleGetSnapshots returns a paginated list of page change snapshots for a monitor.
func HandleGetSnapshots(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		monitorIDStr := chi.URLParam(r, "id")

		monitorID, err := strconv.Atoi(monitorIDStr)
		if err != nil {
			http.Error(w, "Invalid monitor ID", http.StatusBadRequest)
			return
		}

		// Verify user owns monitor
		var count int64
		db.Model(&models.Monitor{}).Where("id = ? AND user_id = ?", monitorID, user.ID).Count(&count)
		if count == 0 {
			http.Error(w, "Monitor not found", http.StatusNotFound)
			return
		}

		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
				limit = parsed
			}
		}

		offset := 0
		if o := r.URL.Query().Get("offset"); o != "" {
			if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
				offset = parsed
			}
		}

		var snapshots []monitor.PageChangeSnapshot
		err = db.Where("monitor_id = ?", monitorID).
			Order("created_at DESC").
			Limit(limit).
			Offset(offset).
			Find(&snapshots).Error

		if err != nil {
			http.Error(w, "Failed to fetch snapshots", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(snapshots)
	}
}

// HandleGetScreenshot serves the current screenshot image for a snapshot.
func HandleGetScreenshot(db *gorm.DB, storagePath string) http.HandlerFunc {
	return serveSnapshotFile(db, storagePath, func(s *monitor.PageChangeSnapshot) string {
		return s.ScreenshotPath
	})
}

// HandleGetScreenshotDiff serves the diff image for a snapshot.
func HandleGetScreenshotDiff(db *gorm.DB, storagePath string) http.HandlerFunc {
	return serveSnapshotFile(db, storagePath, func(s *monitor.PageChangeSnapshot) string {
		return s.DiffPath
	})
}

// HandleGetScreenshotBaseline serves the baseline screenshot for a snapshot.
func HandleGetScreenshotBaseline(db *gorm.DB, storagePath string) http.HandlerFunc {
	return serveSnapshotFile(db, storagePath, func(s *monitor.PageChangeSnapshot) string {
		return s.BaselinePath
	})
}

// serveSnapshotFile is a helper that serves a PNG file from a snapshot record.
func serveSnapshotFile(db *gorm.DB, storagePath string, getPath func(*monitor.PageChangeSnapshot) string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		monitorIDStr := chi.URLParam(r, "id")
		snapshotIDStr := chi.URLParam(r, "snapshotId")

		monitorID, err := strconv.Atoi(monitorIDStr)
		if err != nil {
			http.Error(w, "Invalid monitor ID", http.StatusBadRequest)
			return
		}

		snapshotID, err := strconv.Atoi(snapshotIDStr)
		if err != nil {
			http.Error(w, "Invalid snapshot ID", http.StatusBadRequest)
			return
		}

		// Verify user owns monitor
		var count int64
		db.Model(&models.Monitor{}).Where("id = ? AND user_id = ?", monitorID, user.ID).Count(&count)
		if count == 0 {
			http.Error(w, "Monitor not found", http.StatusNotFound)
			return
		}

		// Get snapshot
		var snapshot monitor.PageChangeSnapshot
		err = db.Where("id = ? AND monitor_id = ?", snapshotID, monitorID).First(&snapshot).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Snapshot not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to fetch snapshot", http.StatusInternalServerError)
			}
			return
		}

		relPath := getPath(&snapshot)
		if relPath == "" {
			http.Error(w, "No image available for this snapshot", http.StatusNotFound)
			return
		}

		absPath := filepath.Clean(filepath.Join(storagePath, relPath))

		// Verify the file path doesn't escape the storage directory
		cleanStorage := filepath.Clean(storagePath)
		if len(absPath) < len(cleanStorage) || absPath[:len(cleanStorage)] != cleanStorage {
			http.Error(w, "Invalid file path", http.StatusBadRequest)
			return
		}

		data, err := os.ReadFile(absPath)
		if err != nil {
			http.Error(w, "Screenshot file not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(data)
	}
}
