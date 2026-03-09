package monitor

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/models"
)

// CertLoader loads a certificate for a given user.
type CertLoader interface {
	LoadCertificate(ctx context.Context, certID int, userID int) (*models.Certificate, error)
}

// DBCertLoader is the production implementation backed by GORM.
type DBCertLoader struct {
	db *gorm.DB
}

var _ CertLoader = (*DBCertLoader)(nil)

// NewDBCertLoader creates a new DBCertLoader.
func NewDBCertLoader(db *gorm.DB) *DBCertLoader {
	return &DBCertLoader{db: db}
}

// LoadCertificate loads a certificate scoped to the given user.
// Returns an error if the cert does not exist or belongs to a different user.
func (l *DBCertLoader) LoadCertificate(ctx context.Context, certID int, userID int) (*models.Certificate, error) {
	var cert models.Certificate
	err := l.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", certID, userID).
		First(&cert).Error
	if err != nil {
		return nil, fmt.Errorf("certificate %d not found: %w", certID, err)
	}
	return &cert, nil
}
