package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/models"
)

func HandleGetCertificates(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		var certs []models.Certificate
		if err := db.Where("user_id = ?", user.ID).Order("created_at DESC").Find(&certs).Error; err != nil {
			http.Error(w, "Failed to fetch certificates", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(certs)
	}
}

func HandleGetCertificate(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		var cert models.Certificate
		if err := db.Where("id = ? AND user_id = ?", id, user.ID).First(&cert).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Certificate not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to fetch certificate", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cert)
	}
}

func HandleCreateCertificate(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		var req struct {
			Name    string `json:"name"`
			CertPEM string `json:"cert_pem"`
			KeyPEM  string `json:"key_pem"`
			CAPEM   string `json:"ca_pem"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.CertPEM == "" || req.KeyPEM == "" {
			http.Error(w, "name, cert_pem, and key_pem are required", http.StatusBadRequest)
			return
		}

		cert := models.Certificate{
			UserID:    user.ID,
			Name:      req.Name,
			CertPEM:   req.CertPEM,
			KeyPEM:    req.KeyPEM,
			CAPEM:     req.CAPEM,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := db.Create(&cert).Error; err != nil {
			http.Error(w, "Failed to create certificate", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(cert)
	}
}

func HandleUpdateCertificate(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		var req struct {
			Name    string `json:"name"`
			CertPEM string `json:"cert_pem"`
			KeyPEM  string `json:"key_pem"`
			CAPEM   string `json:"ca_pem"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.CertPEM == "" {
			http.Error(w, "name and cert_pem are required", http.StatusBadRequest)
			return
		}

		// Note: ca_pem is always updated (send "" to clear it).
		// key_pem is only updated if a new value is provided (cannot be retrieved from the API).
		updates := map[string]interface{}{
			"name":       req.Name,
			"cert_pem":   req.CertPEM,
			"ca_pem":     req.CAPEM,
			"updated_at": time.Now(),
		}
		// Only update key_pem if a new one is provided
		if req.KeyPEM != "" {
			updates["key_pem"] = req.KeyPEM
		}

		result := db.Model(&models.Certificate{}).
			Where("id = ? AND user_id = ?", id, user.ID).
			Updates(updates)
		if result.Error != nil {
			http.Error(w, "Failed to update certificate", http.StatusInternalServerError)
			return
		}
		if result.RowsAffected == 0 {
			http.Error(w, "Certificate not found", http.StatusNotFound)
			return
		}

		var cert models.Certificate
		if err := db.Where("id = ? AND user_id = ?", id, user.ID).First(&cert).Error; err != nil {
			http.Error(w, "Failed to fetch updated certificate", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cert)
	}
}

func HandleDeleteCertificate(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		count, err := countCertificateUsage(db, user.ID, id)
		if err != nil {
			http.Error(w, "Failed to check certificate usage", http.StatusInternalServerError)
			return
		}
		if count > 0 {
			http.Error(w, "Certificate is in use by one or more monitors", http.StatusConflict)
			return
		}

		result := db.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.Certificate{})
		if result.Error != nil {
			http.Error(w, "Failed to delete certificate", http.StatusInternalServerError)
			return
		}
		if result.RowsAffected == 0 {
			http.Error(w, "Certificate not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func countCertificateUsage(db *gorm.DB, userID, certID int) (int64, error) {
	var monitorConfigs []struct {
		ConfigRaw string `gorm:"column:config"`
	}
	if err := db.Table("monitors").Select("config").Where("user_id = ?", userID).Find(&monitorConfigs).Error; err != nil {
		return 0, err
	}

	var count int64
	for _, monitorConfig := range monitorConfigs {
		if monitorConfigReferencesCertificate(monitorConfig.ConfigRaw, certID) {
			count++
		}
	}

	return count, nil
}

func monitorConfigReferencesCertificate(configRaw string, certID int) bool {
	if configRaw == "" {
		return false
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configRaw), &config); err != nil {
		return false
	}

	certIDRaw, ok := config["certificate_id"]
	if !ok {
		return false
	}

	switch v := certIDRaw.(type) {
	case float64:
		return v == float64(certID)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		return err == nil && parsed == certID
	case int:
		return v == certID
	case json.Number:
		parsed, err := strconv.Atoi(v.String())
		return err == nil && parsed == certID
	default:
		return false
	}
}
