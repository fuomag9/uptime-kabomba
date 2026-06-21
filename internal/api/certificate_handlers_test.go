package api

import "testing"

func TestMonitorConfigReferencesCertificate(t *testing.T) {
	tests := []struct {
		name      string
		configRaw string
		certID    int
		want      bool
	}{
		{
			name:      "numeric certificate id matches",
			configRaw: `{"certificate_id":42}`,
			certID:    42,
			want:      true,
		},
		{
			name:      "string certificate id matches",
			configRaw: `{"certificate_id":"42"}`,
			certID:    42,
			want:      true,
		},
		{
			name:      "empty string does not match",
			configRaw: `{"certificate_id":""}`,
			certID:    42,
			want:      false,
		},
		{
			name:      "non numeric string does not match",
			configRaw: `{"certificate_id":"not-a-number"}`,
			certID:    42,
			want:      false,
		},
		{
			name:      "different certificate id does not match",
			configRaw: `{"certificate_id":7}`,
			certID:    42,
			want:      false,
		},
		{
			name:      "missing certificate id does not match",
			configRaw: `{"method":"GET"}`,
			certID:    42,
			want:      false,
		},
		{
			name:      "invalid JSON does not match",
			configRaw: `{`,
			certID:    42,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := monitorConfigReferencesCertificate(tt.configRaw, tt.certID)
			if got != tt.want {
				t.Fatalf("monitorConfigReferencesCertificate(%q, %d) = %v, want %v", tt.configRaw, tt.certID, got, tt.want)
			}
		})
	}
}
