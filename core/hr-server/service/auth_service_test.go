package service

import "testing"

func TestResolveRefreshTokenExpireSeconds(t *testing.T) {
	const (
		oneDay     = 24 * 3600
		thirtyDays = 30 * 24 * 3600
		ninetyDays = 90 * 24 * 3600
	)

	tests := []struct {
		name       string
		configured int
		remember   bool
		want       int
	}{
		{name: "uses configured ttl", configured: ninetyDays, remember: false, want: ninetyDays},
		{name: "remember never shortens configured ttl", configured: ninetyDays, remember: true, want: ninetyDays},
		{name: "remember extends short configured ttl", configured: oneDay, remember: true, want: thirtyDays},
		{name: "default ttl when missing", configured: 0, remember: false, want: oneDay},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveRefreshTokenExpireSeconds(tt.configured, tt.remember); got != tt.want {
				t.Fatalf("resolveRefreshTokenExpireSeconds(%d, %v) = %d, want %d", tt.configured, tt.remember, got, tt.want)
			}
		})
	}
}
