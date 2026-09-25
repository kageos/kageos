package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/message-server/model"
	"github.com/kageos/kageos/core/message-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInboxSearchHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.MessageEntry{}, &model.MessageRecipient{}); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewMessageRepository(db)
	for _, title := range []string{"会议提醒", "库存提醒"} {
		if _, err := repo.Create(context.Background(), dto.MessageSendMeta{From: "system", SourcePath: "/owner/app/notify.form"}, dto.MessageSendPayload{Title: title, Content: title}, []string{"alice"}); err != nil {
			t.Fatal(err)
		}
	}
	server := &Server{messageRepo: repo}
	for _, tc := range []struct {
		name, query, user string
		wantCode          int
		wantTotal         int64
	}{
		{"keyword", "q=" + url.QueryEscape("会议"), "alice", 0, 1},
		{"other recipient", "q=" + url.QueryEscape("会议"), "bob", 0, 0},
		{"future time", "since=2099-01-01T00:00:00Z", "alice", 0, 0},
		{"invalid time", "since=invalid", "alice", 7, 0},
		{"long keyword", "q=" + url.QueryEscape(strings.Repeat("字", 201)), "alice", 7, 0},
		{"no authentication", "q=hello", "", 7, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/message/api/v1/inbox?"+tc.query, nil)
			if tc.user != "" {
				c.Request.Header.Set(contextx.RequestUserHeader, tc.user)
			}
			server.listInboxMessages(c)
			var result struct {
				Code int                      `json:"code"`
				Data dto.MessageInboxListResp `json:"data"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Code != tc.wantCode || result.Data.Total != tc.wantTotal {
				t.Fatalf("response = %s", recorder.Body.String())
			}
		})
	}
}
