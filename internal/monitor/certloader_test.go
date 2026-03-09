package monitor

import (
	"testing"

	"github.com/fuomag9/uptime-kabomba/internal/models"
)

// Verify DBCertLoader satisfies CertLoader at compile time.
// The var _ assertion in certloader.go already does this, but this test
// serves as a clear documentation point.
func TestDBCertLoaderImplementsCertLoader(t *testing.T) {
	var _ CertLoader = (*DBCertLoader)(nil)
}

// Verify that a nil CertLoader is valid to store in HTTPMonitor.
func TestHTTPMonitorAcceptsNilCertLoader(t *testing.T) {
	m := NewHTTPMonitor(nil)
	if m == nil {
		t.Fatal("NewHTTPMonitor returned nil")
	}
}

// Ensure Certificate model has the fields we depend on.
func TestCertificateModelFields(t *testing.T) {
	cert := models.Certificate{
		ID:      1,
		UserID:  2,
		Name:    "test",
		CertPEM: "cert",
		KeyPEM:  "key",
		CAPEM:   "ca",
	}
	if cert.ID != 1 || cert.UserID != 2 {
		t.Fatal("Certificate model fields not accessible")
	}
}
