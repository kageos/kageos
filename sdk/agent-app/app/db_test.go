package app

import (
	"path/filepath"
	"testing"
)

func TestSanitizeDBName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "adds suffix", in: "orders", want: filepath.Join(getDataDir(), "orders.db")},
		{name: "keeps db suffix", in: "orders.db", want: filepath.Join(getDataDir(), "orders.db")},
		{name: "drops traversal directory", in: "../secrets/orders.db", want: filepath.Join(getDataDir(), "orders.db")},
		{name: "uses base name", in: "nested/orders.db", want: filepath.Join(getDataDir(), "orders.db")},
	}

	for _, tt := range tests {
		if got := sanitizeDBName(tt.in); got != tt.want {
			t.Fatalf("%s: want %s, got %s", tt.name, tt.want, got)
		}
	}
}

func TestDBLogFilePath(t *testing.T) {
	got := dbLogFilePath(filepath.Join("/tmp", "agent", "orders.db"))
	want := filepath.Join("/tmp", "agent", "orders.log")
	if got != want {
		t.Fatalf("want %s, got %s", want, got)
	}
}
