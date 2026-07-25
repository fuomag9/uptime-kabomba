package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	cssscanner "github.com/gorilla/css/scanner"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/models"
)

// allowedAtRules are the only at-rules custom status-page CSS may use.
// Everything else - known-dangerous (@import, @namespace) or simply unknown
// - is stripped. Defaulting to deny (rather than blocking a rule blocklist)
// matters here: an at-keyword the scanner doesn't recognize as one of these
// is dropped rather than passed through, which is what closes comment-split
// obfuscation like "@im/**/port \"https://evil.example/x.css\";" - "@im" is
// not an allowed at-rule, so everything up to the next ";"/"{" (including
// the reassembled "port" and the string) is discarded with it, rather than
// individually surviving as "harmless" tokens that reassemble into a working
// @import once the comment between them is removed.
var allowedAtRules = map[string]bool{
	"@media":             true,
	"@supports":          true,
	"@font-face":         true,
	"@keyframes":         true,
	"@-webkit-keyframes": true,
	"@-moz-keyframes":    true,
	"@page":              true,
	"@container":         true,
	"@layer":             true,
	"@property":          true,
	"@scope":             true,
	"@starting-style":    true,
}

// sanitizeCustomCSS tokenizes the provided CSS (rather than pattern-matching
// on the raw string) and drops constructs that can turn a public status
// page's custom CSS into an exfiltration or tracking vector: @import and any
// other non-allowlisted at-rule, url()/url-prefix()/domain() (loads external
// resources, beaconing the visitor's IP/UA to an attacker-chosen host), and
// legacy script-execution vectors (expression(), -moz-binding, behavior).
// Tokenizing - rather than a regex over the raw string - means the filter
// follows CSS grammar instead of literal text, so comment-splitting tricks
// like "@im/**/port" can't reassemble into something the checks below would
// have allowed through individually. Any token that still carries a
// backslash escape is dropped outright rather than decoded, since legitimate
// status-page styling never needs escaped identifiers and escapes are the
// classic way to smuggle a keyword like "url(" past a naive filter.
func sanitizeCustomCSS(css string) string {
	if css == "" {
		return ""
	}

	s := cssscanner.New(css)
	var out strings.Builder
	skipStatement := false

	for {
		tok := s.Next()
		if tok.Type == cssscanner.TokenEOF || tok.Type == cssscanner.TokenError {
			break
		}

		if strings.Contains(tok.Value, `\`) {
			continue
		}

		if skipStatement {
			if tok.Type == cssscanner.TokenChar && (tok.Value == ";" || tok.Value == "{") {
				skipStatement = false
			}
			continue
		}

		lower := strings.ToLower(tok.Value)

		switch tok.Type {
		case cssscanner.TokenComment, cssscanner.TokenCDO, cssscanner.TokenCDC:
			continue
		case cssscanner.TokenAtKeyword:
			if !allowedAtRules[lower] {
				skipStatement = true
				continue
			}
		case cssscanner.TokenURI:
			// The whole url(...) - including its contents - is a single
			// token here; drop it entirely.
			continue
		case cssscanner.TokenFunction:
			name := strings.TrimSuffix(lower, "(")
			if name == "url" || name == "url-prefix" || name == "domain" || name == "expression" {
				depth := 1
				for depth > 0 {
					t := s.Next()
					if t.Type == cssscanner.TokenEOF || t.Type == cssscanner.TokenError {
						break
					}
					if t.Type == cssscanner.TokenChar && t.Value == "(" {
						depth++
					} else if t.Type == cssscanner.TokenChar && t.Value == ")" {
						depth--
					}
				}
				continue
			}
		case cssscanner.TokenIdent, cssscanner.TokenString:
			if strings.Contains(lower, "javascript:") || strings.Contains(lower, "expression") ||
				strings.Contains(lower, "-moz-binding") || strings.Contains(lower, "behavior") {
				continue
			}
		}

		out.WriteString(tok.Value)
	}

	result := out.String()
	// Defense in depth against anything the tokenizer pass above missed.
	result = strings.ReplaceAll(result, "<", "")
	result = strings.ReplaceAll(result, ">", "")
	return result
}

// verifyMonitorOwnership returns an error unless every ID in monitorIDs
// belongs to userID. Status pages are the only unauthenticated read surface
// in the app, so letting a caller attach a monitor they don't own would turn
// one authenticated account into a permanent, unauthenticated, cross-tenant
// oracle for that monitor's data via the public status page endpoints.
func verifyMonitorOwnership(db *gorm.DB, monitorIDs []int, userID int) error {
	if len(monitorIDs) == 0 {
		return nil
	}

	unique := make(map[int]struct{}, len(monitorIDs))
	for _, id := range monitorIDs {
		unique[id] = struct{}{}
	}
	ids := make([]int, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}

	var ownedCount int64
	if err := db.Model(&models.Monitor{}).
		Where("id IN ? AND user_id = ?", ids, userID).
		Count(&ownedCount).Error; err != nil {
		return fmt.Errorf("failed to verify monitor ownership")
	}
	if int(ownedCount) != len(ids) {
		return fmt.Errorf("one or more monitors not found")
	}
	return nil
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

		// Every monitor_id must belong to the caller - otherwise an
		// authenticated user could attach (and later, via the unauthenticated
		// public read endpoints, leak) another tenant's monitor data by
		// guessing/enumerating monitor IDs.
		if err := verifyMonitorOwnership(db, req.MonitorIDs, user.ID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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

		// Every monitor_id must belong to the caller (see HandleCreateStatusPage).
		if err := verifyMonitorOwnership(db, req.MonitorIDs, user.ID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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
				"custom_css":      sanitizeCustomCSS(req.CustomCSS),
				"updated_at":      time.Now(),
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

		// Re-sanitize on every read (not just at write time) so any row
		// written before this sanitizer existed is cleaned up retroactively.
		page.CustomCSS = sanitizeCustomCSS(page.CustomCSS)

		// Get monitors with their latest heartbeat
		type StatusHistoryBucket struct {
			Start  time.Time `json:"start"`
			Status int       `json:"status"`
		}
		type MonitorWithStatus struct {
			models.Monitor
			LastHeartbeat *models.Heartbeat     `json:"last_heartbeat"`
			History       []StatusHistoryBucket `json:"history"`
		}

		// Scope by the page owner as well as the association, so a monitor
		// that was linked before ownership was enforced at write time (or
		// whose owner changed since) can never leak through a page it no
		// longer belongs to.
		var monitors []models.Monitor
		db.Joins("INNER JOIN status_page_monitors spm ON monitors.id = spm.monitor_id").
			Where("spm.status_page_id = ? AND monitors.user_id = ?", page.ID, page.UserID).
			Order("monitors.name ASC").
			Find(&monitors)

		monitorsWithStatus := make([]MonitorWithStatus, len(monitors))
		monitorIDs := make([]int, 0, len(monitors))
		for i, monitor := range monitors {
			monitorsWithStatus[i].Monitor = monitor
			monitorIDs = append(monitorIDs, monitor.ID)
		}

		// Get latest heartbeat per monitor using individual indexed lookups
		lastHeartbeatStatusByMonitor := make(map[int]int, len(monitorIDs))
		if len(monitorIDs) > 0 {
			latestByMonitor := make(map[int]models.Heartbeat, len(monitorIDs))
			for _, mid := range monitorIDs {
				var hb models.Heartbeat
				if err := db.Where("monitor_id = ?", mid).
					Order("time DESC").
					Limit(1).
					First(&hb).Error; err == nil {
					latestByMonitor[mid] = hb
					lastHeartbeatStatusByMonitor[mid] = hb.Status
				}
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
			lastStatusByMonitor := make(map[int]int, len(monitorIDs))
			for _, mid := range monitorIDs {
				var row lastStatusRow
				if err := db.Raw(`
					SELECT monitor_id, status
					FROM heartbeats
					WHERE monitor_id = ? AND time < ?
					ORDER BY time DESC
					LIMIT 1
				`, mid, start).Scan(&row).Error; err == nil && row.MonitorID != 0 {
					lastStatusByMonitor[mid] = row.Status
				}
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

		// Ensure the monitor belongs to this page AND is still owned by the
		// page's owner (see HandleGetPublicStatusPage).
		var count int64
		db.Table("status_page_monitors").
			Joins("INNER JOIN monitors ON monitors.id = status_page_monitors.monitor_id").
			Where("status_page_monitors.status_page_id = ? AND status_page_monitors.monitor_id = ? AND monitors.user_id = ?", page.ID, monitorID, page.UserID).
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
