package v1

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/pkg/gormx/models"
)

func TestUserDTOIncludesLastLogin(t *testing.T) {
	user := &model.User{Username: "alice"}
	if got := convertUserToDTO(user); got.LastLoginAt != nil {
		t.Fatal("unknown login time must remain null")
	}
	at := models.Time(time.Date(2026, 9, 10, 8, 30, 0, 0, time.FixedZone("CST", 8*3600)))
	user.LastLoginAt = &at
	got := convertUserToDTOWithDetails(user, nil, nil)
	if got.LastLoginAt == nil || *got.LastLoginAt != "2026-09-10T08:30:00+08:00" {
		t.Fatalf("invalid login timestamp: %+v", got.LastLoginAt)
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if result["last_login_at"] != "2026-09-10T08:30:00+08:00" {
		t.Fatalf("missing login timestamp: %s", data)
	}
}
