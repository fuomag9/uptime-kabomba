package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"

	"github.com/fuomag9/uptime-kuma-go/internal/models"
)

// HandleGetStatusPages returns all status pages for the current user
func HandleGetStatusPages(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		var pages []models.StatusPage
		query := `SELECT id, user_id, slug, title, description, published, show_powered_by, theme, custom_css, created_at, updated_at
		          FROM status_pages WHERE user_id = ? ORDER BY created_at DESC`

		err := db.Select(&pages, query, user.ID)
		if err != nil {
			http.Error(w, "Failed to fetch status pages", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pages)
	}
}

// HandleGetStatusPage returns a single status page by ID
func HandleGetStatusPage(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		pageID := chi.URLParam(r, "id")

		var page models.StatusPage
		query := `SELECT id, user_id, slug, title, description, published, show_powered_by, theme, custom_css, created_at, updated_at
		          FROM status_pages WHERE id = ? AND user_id = ?`

		err := db.Get(&page, query, pageID, user.ID)
		if err != nil {
			http.Error(w, "Status page not found", http.StatusNotFound)
			return
		}

		// Get monitors for this status page
		var monitors []models.Monitor
		monitorQuery := `
			SELECT m.id, m.user_id, m.name, m.type, m.url, m.interval, m.timeout, m.active, m.config, m.created_at, m.updated_at
			FROM monitors m
			INNER JOIN status_page_monitors spm ON m.id = spm.monitor_id
			WHERE spm.status_page_id = ?
			ORDER BY spm.display_order ASC
		`
		db.Select(&monitors, monitorQuery, pageID)

		result := models.StatusPageWithMonitors{
			StatusPage: page,
			Monitors:   monitors,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// HandleCreateStatusPage creates a new status page
func HandleCreateStatusPage(db *sqlx.DB) http.HandlerFunc {
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
		var count int
		db.Get(&count, "SELECT COUNT(*) FROM status_pages WHERE slug = ?", req.Slug)
		if count > 0 {
			http.Error(w, "Slug already exists", http.StatusConflict)
			return
		}

		// Hash password if provided
		var passwordHash sql.NullString
		if req.Password != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				http.Error(w, "Failed to hash password", http.StatusInternalServerError)
				return
			}
			passwordHash = sql.NullString{String: string(hash), Valid: true}
		}

		// Insert status page
		query := `
			INSERT INTO status_pages (user_id, slug, title, description, published, show_powered_by, theme, custom_css, password, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			RETURNING id
		`

		now := time.Now()
		var pageID int
		err := db.QueryRow(query,
			user.ID, req.Slug, req.Title, req.Description, req.Published,
			req.ShowPoweredBy, req.Theme, req.CustomCSS, passwordHash, now, now,
		).Scan(&pageID)

		if err != nil {
			http.Error(w, "Failed to create status page", http.StatusInternalServerError)
			return
		}

		// Add monitors to status page
		if len(req.MonitorIDs) > 0 {
			for i, monitorID := range req.MonitorIDs {
				_, err := db.Exec(
					"INSERT INTO status_page_monitors (status_page_id, monitor_id, display_order) VALUES (?, ?, ?)",
					pageID, monitorID, i,
				)
				if err != nil {
					// Log error but continue
					continue
				}
			}
		}

		// Return created page
		var page models.StatusPage
		db.Get(&page, "SELECT id, user_id, slug, title, description, published, show_powered_by, theme, custom_css, created_at, updated_at FROM status_pages WHERE id = ?", pageID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(page)
	}
}

// HandleUpdateStatusPage updates an existing status page
func HandleUpdateStatusPage(db *sqlx.DB) http.HandlerFunc {
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
		var count int
		db.Get(&count, "SELECT COUNT(*) FROM status_pages WHERE id = ? AND user_id = ?", pageID, user.ID)
		if count == 0 {
			http.Error(w, "Status page not found", http.StatusNotFound)
			return
		}

		// Check slug uniqueness (excluding current page)
		db.Get(&count, "SELECT COUNT(*) FROM status_pages WHERE slug = ? AND id != ?", req.Slug, pageID)
		if count > 0 {
			http.Error(w, "Slug already exists", http.StatusConflict)
			return
		}

		// Hash password if provided
		var passwordHash sql.NullString
		if req.Password != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				http.Error(w, "Failed to hash password", http.StatusInternalServerError)
				return
			}
			passwordHash = sql.NullString{String: string(hash), Valid: true}
		}

		// Update status page
		query := `
			UPDATE status_pages
			SET slug = ?, title = ?, description = ?, published = ?, show_powered_by = ?,
			    theme = ?, custom_css = ?, password = ?, updated_at = ?
			WHERE id = ? AND user_id = ?
		`

		_, err := db.Exec(query,
			req.Slug, req.Title, req.Description, req.Published, req.ShowPoweredBy,
			req.Theme, req.CustomCSS, passwordHash, time.Now(), pageID, user.ID,
		)

		if err != nil {
			http.Error(w, "Failed to update status page", http.StatusInternalServerError)
			return
		}

		// Update monitors
		// First, delete all existing monitor associations
		db.Exec("DELETE FROM status_page_monitors WHERE status_page_id = ?", pageID)

		// Add new monitors
		if len(req.MonitorIDs) > 0 {
			for i, monitorID := range req.MonitorIDs {
				db.Exec(
					"INSERT INTO status_page_monitors (status_page_id, monitor_id, display_order) VALUES (?, ?, ?)",
					pageID, monitorID, i,
				)
			}
		}

		// Return updated page
		var page models.StatusPage
		db.Get(&page, "SELECT id, user_id, slug, title, description, published, show_powered_by, theme, custom_css, created_at, updated_at FROM status_pages WHERE id = ?", pageID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(page)
	}
}

// HandleDeleteStatusPage deletes a status page
func HandleDeleteStatusPage(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		pageID := chi.URLParam(r, "id")

		// Delete from database
		result, err := db.Exec("DELETE FROM status_pages WHERE id = ? AND user_id = ?", pageID, user.ID)
		if err != nil {
			http.Error(w, "Failed to delete status page", http.StatusInternalServerError)
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			http.Error(w, "Status page not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// HandleGetPublicStatusPage returns a public status page by slug (no auth required)
func HandleGetPublicStatusPage(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")

		var page models.StatusPage
		query := `SELECT id, user_id, slug, title, description, published, show_powered_by, theme, custom_css, password, created_at, updated_at
		          FROM status_pages WHERE slug = ? AND published = 1`

		err := db.Get(&page, query, slug)
		if err != nil {
			http.Error(w, "Status page not found", http.StatusNotFound)
			return
		}

		// Check if password protected
		if page.Password != "" {
			// Get password from header or query param
			providedPassword := r.Header.Get("X-Status-Page-Password")
			if providedPassword == "" {
				providedPassword = r.URL.Query().Get("password")
			}

			if providedPassword == "" {
				http.Error(w, "Password required", http.StatusUnauthorized)
				return
			}

			// Verify password
			if err := bcrypt.CompareHashAndPassword([]byte(page.Password), []byte(providedPassword)); err != nil {
				http.Error(w, "Invalid password", http.StatusUnauthorized)
				return
			}
		}

		// Get monitors with their latest heartbeat
		type MonitorWithStatus struct {
			models.Monitor
			LastHeartbeat *models.Heartbeat `json:"last_heartbeat"`
		}

		monitorQuery := `
			SELECT m.id, m.user_id, m.name, m.type, m.url, m.interval, m.timeout, m.active, m.config, m.created_at, m.updated_at
			FROM monitors m
			INNER JOIN status_page_monitors spm ON m.id = spm.monitor_id
			WHERE spm.status_page_id = ?
			ORDER BY spm.display_order ASC
		`

		var monitors []models.Monitor
		db.Select(&monitors, monitorQuery, page.ID)

		monitorsWithStatus := make([]MonitorWithStatus, len(monitors))
		for i, monitor := range monitors {
			monitorsWithStatus[i].Monitor = monitor

			// Get latest heartbeat
			var heartbeat models.Heartbeat
			heartbeatQuery := `SELECT id, monitor_id, status, ping, important, message, time
			                   FROM heartbeats WHERE monitor_id = ? ORDER BY time DESC LIMIT 1`
			if err := db.Get(&heartbeat, heartbeatQuery, monitor.ID); err == nil {
				monitorsWithStatus[i].LastHeartbeat = &heartbeat
			}
		}

		// Get recent incidents
		var incidents []models.Incident
		incidentQuery := `SELECT id, status_page_id, title, content, style, pin, created_at, updated_at
		                  FROM incidents WHERE status_page_id = ? ORDER BY pin DESC, created_at DESC LIMIT 10`
		db.Select(&incidents, incidentQuery, page.ID)

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
func HandleGetIncidents(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		pageID := chi.URLParam(r, "id")

		// Verify ownership
		var count int
		db.Get(&count, "SELECT COUNT(*) FROM status_pages WHERE id = ? AND user_id = ?", pageID, user.ID)
		if count == 0 {
			http.Error(w, "Status page not found", http.StatusNotFound)
			return
		}

		var incidents []models.Incident
		query := `SELECT id, status_page_id, title, content, style, pin, created_at, updated_at
		          FROM incidents WHERE status_page_id = ? ORDER BY pin DESC, created_at DESC`

		err := db.Select(&incidents, query, pageID)
		if err != nil {
			http.Error(w, "Failed to fetch incidents", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(incidents)
	}
}

// HandleCreateIncident creates a new incident
func HandleCreateIncident(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		pageID := chi.URLParam(r, "id")

		// Verify ownership
		var count int
		db.Get(&count, "SELECT COUNT(*) FROM status_pages WHERE id = ? AND user_id = ?", pageID, user.ID)
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

		query := `
			INSERT INTO incidents (status_page_id, title, content, style, pin, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			RETURNING id
		`

		now := time.Now()
		var incidentID int
		err := db.QueryRow(query, pageID, req.Title, req.Content, req.Style, req.Pin, now, now).Scan(&incidentID)

		if err != nil {
			http.Error(w, "Failed to create incident", http.StatusInternalServerError)
			return
		}

		var incident models.Incident
		db.Get(&incident, "SELECT id, status_page_id, title, content, style, pin, created_at, updated_at FROM incidents WHERE id = ?", incidentID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(incident)
	}
}

// HandleDeleteIncident deletes an incident
func HandleDeleteIncident(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		pageID := chi.URLParam(r, "id")
		incidentID := chi.URLParam(r, "incidentId")

		// Verify ownership
		var count int
		db.Get(&count, "SELECT COUNT(*) FROM status_pages WHERE id = ? AND user_id = ?", pageID, user.ID)
		if count == 0 {
			http.Error(w, "Status page not found", http.StatusNotFound)
			return
		}

		result, err := db.Exec("DELETE FROM incidents WHERE id = ? AND status_page_id = ?", incidentID, pageID)
		if err != nil {
			http.Error(w, "Failed to delete incident", http.StatusInternalServerError)
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			http.Error(w, "Incident not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
