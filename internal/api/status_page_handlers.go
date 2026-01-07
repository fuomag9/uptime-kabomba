package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/models"
)

// HandleGetStatusPages returns all status pages for the current user
func HandleGetStatusPages(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		var pages []models.StatusPage
		err := db.Where("user_id = ?", user.ID).
			Order("created_at DESC").
			Find(&pages).Error

		if err != nil {
			http.Error(w, "Failed to fetch status pages", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pages)
	}
}

// HandleGetStatusPage returns a single status page by ID
func HandleGetStatusPage(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		pageID := chi.URLParam(r, "id")

		var page models.StatusPage
		err := db.Where("id = ? AND user_id = ?", pageID, user.ID).
			First(&page).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Status page not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to fetch status page", http.StatusInternalServerError)
			}
			return
		}

		// Get monitors for this status page
		var monitors []models.Monitor
		db.Joins("INNER JOIN status_page_monitors spm ON monitors.id = spm.monitor_id").
			Where("spm.status_page_id = ?", pageID).
			Order("monitors.name ASC").
			Find(&monitors)

		result := models.StatusPageWithMonitors{
			StatusPage: page,
			Monitors:   monitors,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// HandleCreateStatusPage creates a new status page
func HandleCreateStatusPage(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		var req struct {
			Slug          string `json:"slug"`
			Title         string `json:"title"`
			Description   string `json:"description"`
			Published     bool   `json:"published"`
			ShowPoweredBy bool   `json:"show_powered_by"`
			Theme         string `json:"theme"`
			CustomCSS     string `json:"custom_css"`
			Password      string `json:"password"`
			MonitorIDs    []int  `json:"monitor_ids"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate slug is unique
		var count int64
		db.Model(&models.StatusPage{}).
			Where("slug = ?", req.Slug).
			Count(&count)
		if count > 0 {
			http.Error(w, "Slug already exists", http.StatusConflict)
			return
		}

		// Create status page
		now := time.Now()
		page := models.StatusPage{
			UserID:        user.ID,
			Slug:          req.Slug,
			Title:         req.Title,
			Description:   req.Description,
			Published:     req.Published,
			ShowPoweredBy: req.ShowPoweredBy,
			Theme:         req.Theme,
			CustomCSS:     req.CustomCSS,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		err := db.Transaction(func(tx *gorm.DB) error {
			// Create status page
			if err := tx.Create(&page).Error; err != nil {
				return err
			}

			// Add monitors to status page
			if len(req.MonitorIDs) > 0 {
				for _, monitorID := range req.MonitorIDs {
					spm := models.StatusPageMonitor{
						StatusPageID: page.ID,
						MonitorID:    monitorID,
					}
					if err := tx.Create(&spm).Error; err != nil {
						// Log error but continue
						continue
					}
				}
			}

			return nil
		})

		if err != nil {
			http.Error(w, "Failed to create status page", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(page)
	}
}

// HandleUpdateStatusPage updates an existing status page
func HandleUpdateStatusPage(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		pageID := chi.URLParam(r, "id")

		var req struct {
			Slug          string `json:"slug"`
			Title         string `json:"title"`
			Description   string `json:"description"`
			Published     bool   `json:"published"`
			ShowPoweredBy bool   `json:"show_powered_by"`
			Theme         string `json:"theme"`
			CustomCSS     string `json:"custom_css"`
			Password      string `json:"password"`
			MonitorIDs    []int  `json:"monitor_ids"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Verify ownership
		var count int64
		db.Model(&models.StatusPage{}).
			Where("id = ? AND user_id = ?", pageID, user.ID).
			Count(&count)
		if count == 0 {
			http.Error(w, "Status page not found", http.StatusNotFound)
			return
		}

		// Check slug uniqueness (excluding current page)
		db.Model(&models.StatusPage{}).
			Where("slug = ? AND id != ?", req.Slug, pageID).
			Count(&count)
		if count > 0 {
			http.Error(w, "Slug already exists", http.StatusConflict)
			return
		}

		// Update status page using transaction
		err := db.Transaction(func(tx *gorm.DB) error {
			// Update status page
			err := tx.Model(&models.StatusPage{}).
				Where("id = ? AND user_id = ?", pageID, user.ID).
				Updates(map[string]interface{}{
					"slug":            req.Slug,
					"title":           req.Title,
					"description":     req.Description,
					"published":       req.Published,
					"show_powered_by": req.ShowPoweredBy,
					"theme":           req.Theme,
					"custom_css":      req.CustomCSS,
					"updated_at":      time.Now(),
				}).Error
			if err != nil {
				return err
			}

			// Delete all existing monitor associations
			if err := tx.Where("status_page_id = ?", pageID).
				Delete(&models.StatusPageMonitor{}).Error; err != nil {
				return err
			}

			// Add new monitors
			if len(req.MonitorIDs) > 0 {
				pageIDInt, _ := strconv.Atoi(pageID)
				for _, monitorID := range req.MonitorIDs {
					spm := models.StatusPageMonitor{
						StatusPageID: pageIDInt,
						MonitorID:    monitorID,
					}
					if err := tx.Create(&spm).Error; err != nil {
						return err
					}
				}
			}

			return nil
		})

		if err != nil {
			http.Error(w, "Failed to update status page", http.StatusInternalServerError)
			return
		}

		// Return updated page
		var page models.StatusPage
		db.Where("id = ?", pageID).First(&page)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(page)
	}
}

// HandleDeleteStatusPage deletes a status page
func HandleDeleteStatusPage(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		pageID := chi.URLParam(r, "id")

		// Delete from database
		result := db.Where("id = ? AND user_id = ?", pageID, user.ID).
			Delete(&models.StatusPage{})

		if result.Error != nil {
			http.Error(w, "Failed to delete status page", http.StatusInternalServerError)
			return
		}

		if result.RowsAffected == 0 {
			http.Error(w, "Status page not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// HandleGetPublicStatusPage returns a public status page by slug (no auth required)
func HandleGetPublicStatusPage(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")

		var page models.StatusPage
		err := db.Where("slug = ? AND published = ?", slug, true).
			First(&page).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Status page not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to fetch status page", http.StatusInternalServerError)
			}
			return
		}

		// Password protection not implemented yet (no password column in schema)

		// Get monitors with their latest heartbeat
		type MonitorWithStatus struct {
			models.Monitor
			LastHeartbeat *models.Heartbeat `json:"last_heartbeat"`
		}

		var monitors []models.Monitor
		db.Joins("INNER JOIN status_page_monitors spm ON monitors.id = spm.monitor_id").
			Where("spm.status_page_id = ?", page.ID).
			Order("monitors.name ASC").
			Find(&monitors)

		monitorsWithStatus := make([]MonitorWithStatus, len(monitors))
		for i, monitor := range monitors {
			monitorsWithStatus[i].Monitor = monitor

			// Get latest heartbeat
			var heartbeat models.Heartbeat
			if err := db.Where("monitor_id = ?", monitor.ID).
				Order("time DESC").
				Limit(1).
				First(&heartbeat).Error; err == nil {
				monitorsWithStatus[i].LastHeartbeat = &heartbeat
			}
		}

		// Get recent incidents
		var incidents []models.Incident
		db.Where("status_page_id = ?", page.ID).
			Order("pin DESC, created_at DESC").
			Limit(10).
			Find(&incidents)

		result := map[string]interface{}{
			"page":      page,
			"monitors":  monitorsWithStatus,
			"incidents": incidents,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// HandleGetIncidents returns all incidents for a status page
func HandleGetIncidents(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		pageID := chi.URLParam(r, "id")

		// Verify ownership
		var count int64
		db.Model(&models.StatusPage{}).
			Where("id = ? AND user_id = ?", pageID, user.ID).
			Count(&count)
		if count == 0 {
			http.Error(w, "Status page not found", http.StatusNotFound)
			return
		}

		var incidents []models.Incident
		err := db.Where("status_page_id = ?", pageID).
			Order("pin DESC, created_at DESC").
			Find(&incidents).Error

		if err != nil {
			http.Error(w, "Failed to fetch incidents", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(incidents)
	}
}

// HandleCreateIncident creates a new incident
func HandleCreateIncident(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		pageID := chi.URLParam(r, "id")

		// Verify ownership
		var count int64
		db.Model(&models.StatusPage{}).
			Where("id = ? AND user_id = ?", pageID, user.ID).
			Count(&count)
		if count == 0 {
			http.Error(w, "Status page not found", http.StatusNotFound)
			return
		}

		var req struct {
			Title   string `json:"title"`
			Content string `json:"content"`
			Style   string `json:"style"`
			Pin     bool   `json:"pin"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		now := time.Now()
		pageIDInt, _ := strconv.Atoi(pageID)
		incident := models.Incident{
			StatusPageID: pageIDInt,
			Title:        req.Title,
			Content:      req.Content,
			Style:        req.Style,
			Pin:          req.Pin,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		err := db.Create(&incident).Error
		if err != nil {
			http.Error(w, "Failed to create incident", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(incident)
	}
}

// HandleDeleteIncident deletes an incident
func HandleDeleteIncident(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		pageID := chi.URLParam(r, "id")
		incidentID := chi.URLParam(r, "incidentId")

		// Verify ownership
		var count int64
		db.Model(&models.StatusPage{}).
			Where("id = ? AND user_id = ?", pageID, user.ID).
			Count(&count)
		if count == 0 {
			http.Error(w, "Status page not found", http.StatusNotFound)
			return
		}

		result := db.Where("id = ? AND status_page_id = ?", incidentID, pageID).
			Delete(&models.Incident{})

		if result.Error != nil {
			http.Error(w, "Failed to delete incident", http.StatusInternalServerError)
			return
		}

		if result.RowsAffected == 0 {
			http.Error(w, "Incident not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
