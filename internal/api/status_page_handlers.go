package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/models"
)

func isAdminUser(userID int) bool {
	return true
}

func sanitizeCustomCSS(css string) string {
	if css == "" {
		return ""
	}
	css = strings.ReplaceAll(css, "<", "")
	css = strings.ReplaceAll(css, ">", "")
	return css
}

func hasValidStatusPagePassword(r *http.Request, page *models.StatusPage) bool {
	if page.Password == "" {
		return true
	}
	provided := r.Header.Get("X-Status-Page-Password")
	if provided == "" {
		return false
	}
	if strings.HasPrefix(page.Password, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(page.Password), []byte(provided)) == nil
	}
	return page.Password == provided
}

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

		if !isAdminUser(user.ID) {
			for i := range pages {
				pages[i].CustomCSS = ""
			}
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
		if !isAdminUser(user.ID) {
			result.StatusPage.CustomCSS = ""
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

		if !isAdminUser(user.ID) && req.CustomCSS != "" {
			http.Error(w, "Custom CSS requires admin access", http.StatusForbidden)
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
			CustomCSS:     sanitizeCustomCSS(req.CustomCSS),
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if req.Password != "" {
			hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				http.Error(w, "Failed to set password", http.StatusInternalServerError)
				return
			}
			page.Password = string(hashed)
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

		if !isAdminUser(user.ID) && req.CustomCSS != "" {
			http.Error(w, "Custom CSS requires admin access", http.StatusForbidden)
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
			updates := map[string]interface{}{
				"slug":            req.Slug,
				"title":           req.Title,
				"description":     req.Description,
				"published":       req.Published,
				"show_powered_by": req.ShowPoweredBy,
				"theme":           req.Theme,
				"updated_at":      time.Now(),
			}

			if isAdminUser(user.ID) {
				updates["custom_css"] = sanitizeCustomCSS(req.CustomCSS)
			}

			if req.Password != "" {
				hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
				if err != nil {
					return err
				}
				updates["password"] = string(hashed)
			}

			err := tx.Model(&models.StatusPage{}).
				Where("id = ? AND user_id = ?", pageID, user.ID).
				Updates(updates).Error
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

		if !hasValidStatusPagePassword(r, &page) {
			http.Error(w, "Status page password required", http.StatusUnauthorized)
			return
		}

		if !isAdminUser(page.UserID) {
			page.CustomCSS = ""
		} else {
			page.CustomCSS = sanitizeCustomCSS(page.CustomCSS)
		}

		// Get monitors with their latest heartbeat
		type StatusHistoryBucket struct {
			Start  time.Time `json:"start"`
			Status int       `json:"status"`
		}
		type MonitorWithStatus struct {
			models.Monitor
			LastHeartbeat *models.Heartbeat `json:"last_heartbeat"`
			History       []StatusHistoryBucket `json:"history"`
		}

		var monitors []models.Monitor
		db.Joins("INNER JOIN status_page_monitors spm ON monitors.id = spm.monitor_id").
			Where("spm.status_page_id = ?", page.ID).
			Order("monitors.name ASC").
			Find(&monitors)

		monitorsWithStatus := make([]MonitorWithStatus, len(monitors))
		monitorIDs := make([]int, 0, len(monitors))
		for i, monitor := range monitors {
			monitorsWithStatus[i].Monitor = monitor
			monitorIDs = append(monitorIDs, monitor.ID)
		}

		// Get latest heartbeat per monitor in one query
		lastHeartbeatStatusByMonitor := make(map[int]int, len(monitorIDs))
		if len(monitorIDs) > 0 {
			var latest []models.Heartbeat
			db.Raw(`
				SELECT DISTINCT ON (monitor_id) *
				FROM heartbeats
				WHERE monitor_id IN ?
				ORDER BY monitor_id, time DESC
			`, monitorIDs).Scan(&latest)

			latestByMonitor := make(map[int]models.Heartbeat, len(latest))
			for _, hb := range latest {
				latestByMonitor[hb.MonitorID] = hb
				lastHeartbeatStatusByMonitor[hb.MonitorID] = hb.Status
			}

			for i, monitor := range monitors {
				if hb, ok := latestByMonitor[monitor.ID]; ok {
					monitorsWithStatus[i].LastHeartbeat = &hb
				}
			}
		}

		// Build last-1h status history per monitor (bucket size = max(60s, monitor interval))
		now := time.Now().UTC()
		start := now.Add(-1 * time.Hour)
		historyByMonitor := make(map[int][]StatusHistoryBucket, len(monitors))
		intervalByMonitor := make(map[int]time.Duration, len(monitors))

		monitorIntervalByID := make(map[int]int, len(monitors))
		for _, monitor := range monitors {
			monitorIntervalByID[monitor.ID] = monitor.Interval
		}

		for _, monitorID := range monitorIDs {
			intervalSeconds := 60
			if interval, ok := monitorIntervalByID[monitorID]; ok && interval > 0 {
				if interval > intervalSeconds {
					intervalSeconds = interval
				}
			}

			interval := time.Duration(intervalSeconds) * time.Second
			intervalByMonitor[monitorID] = interval

			bucketCount := int((1*time.Hour + interval - 1) / interval)
			buckets := make([]StatusHistoryBucket, bucketCount)
			for i := 0; i < bucketCount; i++ {
				buckets[i] = StatusHistoryBucket{
					Start:  start.Add(time.Duration(i) * interval),
					Status: -1,
				}
			}
			historyByMonitor[monitorID] = buckets
		}

		if len(monitorIDs) > 0 {
			type bucketRow struct {
				MonitorID int   `gorm:"column:monitor_id"`
				Bucket    int64 `gorm:"column:bucket"`
				MaxWeight int   `gorm:"column:max_weight"`
			}

			var rows []bucketRow
			db.Raw(`
				SELECT
					h.monitor_id,
					FLOOR(EXTRACT(EPOCH FROM h.time) / GREATEST(60, m.interval)) AS bucket,
					MAX(
						CASE h.status
							WHEN 0 THEN 4
							WHEN 3 THEN 3
							WHEN 2 THEN 2
							WHEN 1 THEN 1
							ELSE 0
						END
					) AS max_weight
				FROM heartbeats h
				JOIN monitors m ON m.id = h.monitor_id
				WHERE h.monitor_id IN ? AND h.time >= ?
				GROUP BY h.monitor_id, bucket, GREATEST(60, m.interval)
				ORDER BY h.monitor_id, bucket ASC
			`, monitorIDs, start).Scan(&rows)

			type lastStatusRow struct {
				MonitorID int `gorm:"column:monitor_id"`
				Status    int `gorm:"column:status"`
			}
			var lastStatusRows []lastStatusRow
			db.Raw(`
				SELECT DISTINCT ON (monitor_id) monitor_id, status
				FROM heartbeats
				WHERE monitor_id IN ? AND time < ?
				ORDER BY monitor_id, time DESC
			`, monitorIDs, start).Scan(&lastStatusRows)

			lastStatusByMonitor := make(map[int]int, len(lastStatusRows))
			for _, row := range lastStatusRows {
				lastStatusByMonitor[row.MonitorID] = row.Status
			}

			bucketStatusByMonitor := make(map[int]map[int]int, len(monitorIDs))
			for _, row := range rows {
				buckets, ok := historyByMonitor[row.MonitorID]
				if !ok || len(buckets) == 0 {
					continue
				}

				interval := intervalByMonitor[row.MonitorID]
				if interval <= 0 {
					interval = time.Minute
				}
				intervalSeconds := int64(interval.Seconds())
				if intervalSeconds <= 0 {
					continue
				}

				bucketStartIndex := start.Unix() / intervalSeconds
				idx := int(row.Bucket - bucketStartIndex)
				if idx < 0 || idx >= len(buckets) {
					continue
				}

				status := -1
				switch row.MaxWeight {
				case 4:
					status = 0
				case 3:
					status = 3
				case 2:
					status = 2
				case 1:
					status = 1
				}
				if _, ok := bucketStatusByMonitor[row.MonitorID]; !ok {
					bucketStatusByMonitor[row.MonitorID] = make(map[int]int)
				}
				bucketStatusByMonitor[row.MonitorID][idx] = status
			}

			for _, monitorID := range monitorIDs {
				buckets, ok := historyByMonitor[monitorID]
				if !ok || len(buckets) == 0 {
					continue
				}

				lastStatus, hasLast := lastStatusByMonitor[monitorID]
				if !hasLast {
					if hbStatus, ok := lastHeartbeatStatusByMonitor[monitorID]; ok {
						lastStatus = hbStatus
						hasLast = true
					}
				}
				for i := range buckets {
					if statusMap, ok := bucketStatusByMonitor[monitorID]; ok {
						if status, ok := statusMap[i]; ok {
							buckets[i].Status = status
							lastStatus = status
							hasLast = true
							continue
						}
					}

					if hasLast {
						buckets[i].Status = lastStatus
					}
				}
			}
		}

		for i, monitor := range monitors {
			if history, ok := historyByMonitor[monitor.ID]; ok {
				monitorsWithStatus[i].History = history
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

// HandleGetPublicStatusPageHeartbeats returns monitor heartbeats for a public status page
func HandleGetPublicStatusPageHeartbeats(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		monitorID := chi.URLParam(r, "id")

		// Ensure status page exists and is published
		var page models.StatusPage
		if err := db.Where("slug = ? AND published = ?", slug, true).
			First(&page).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Status page not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to fetch status page", http.StatusInternalServerError)
			}
			return
		}

		if !hasValidStatusPagePassword(r, &page) {
			http.Error(w, "Status page password required", http.StatusUnauthorized)
			return
		}

		// Ensure monitor belongs to page
		var count int64
		db.Table("status_page_monitors").
			Where("status_page_id = ? AND monitor_id = ?", page.ID, monitorID).
			Count(&count)
		if count == 0 {
			http.Error(w, "Monitor not found", http.StatusNotFound)
			return
		}

		// Get query params
		limitStr := r.URL.Query().Get("limit")
		period := r.URL.Query().Get("period")

		limit := 200
		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 2000 {
				limit = l
			}
		}

		query := db.Where("monitor_id = ?", monitorID)
		if period == "" {
			period = "1h"
		}

		endTime := time.Now()
		switch period {
		case "1h":
			query = query.Where("time >= ? AND time <= ?", endTime.Add(-1*time.Hour), endTime)
		default:
			query = query.Where("time >= ? AND time <= ?", endTime.Add(-1*time.Hour), endTime)
		}

		var heartbeats []models.Heartbeat
		if err := query.Order("time DESC").
			Limit(limit).
			Find(&heartbeats).Error; err != nil {
			http.Error(w, "Failed to fetch heartbeats", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(heartbeats)
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
