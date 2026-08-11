package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWechatProvidersAreRegisteredForSettings(t *testing.T) {
	official, ok := LookupAuthProviderSeed(ProviderWechatOfficial)
	if !ok || official.Action != ProviderActionQRCode || official.AuthorizePath == "" || official.CallbackPath == "" {
		t.Fatalf("official account seed = %+v, registered=%v", official, ok)
	}
	open, ok := LookupAuthProviderSeed(ProviderWechatOpenOAuth)
	if !ok || open.Action != ProviderActionRedirect || open.AuthorizePath == "" || open.CallbackPath == "" {
		t.Fatalf("open platform seed = %+v, registered=%v", open, ok)
	}
	if _, ok := GetOAuthProvider(ProviderWechatOpenOAuth); !ok {
		t.Fatal("wechat open OAuth factory is not registered")
	}
}

func TestBuildWechatOpenAuthorizeURL(t *testing.T) {
	target, err := buildWechatOpenAuthorizeURL(map[string]string{
		"app_id":       "wx-app",
		"app_secret":   "secret",
		"redirect_url": "https://kageos.example/hr/api/v1/auth/wechat-open/callback",
	}, "state-value")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(strings.TrimSuffix(target, "#wechat_redirect"))
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if query.Get("appid") != "wx-app" || query.Get("scope") != "snsapi_login" || query.Get("state") != "state-value" {
		t.Fatalf("unexpected authorize query: %v", query)
	}
}

func TestWechatOpenAuthorizeUsesConfiguredProvider(t *testing.T) {
	db := openWechatServiceTestDB(t)
	providerService := NewAuthLoginProviderService(repository.NewAuthLoginProviderRepository(db))
	if err := providerService.SeedDefaults(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := providerService.UpdateConfig(ProviderWechatOpenOAuth, map[string]string{
		"app_id":       "wx-app",
		"app_secret":   "secret",
		"redirect_url": "https://kageos.example/hr/api/v1/auth/wechat-open/callback",
	}, "system"); err != nil {
		t.Fatal(err)
	}
	if _, err := providerService.SetEnabled(ProviderWechatOpenOAuth, true, "system"); err != nil {
		t.Fatal(err)
	}
	svc := &AuthOAuthService{
		providerService: providerService,
		stateRepo:       repository.NewAuthOAuthStateRepository(db),
	}
	target, err := svc.StartAuthorize(context.Background(), "wechat-open", "/workspace/demo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(target, "https://open.weixin.qq.com/connect/qrconnect?") || !strings.Contains(target, "state=") {
		t.Fatalf("unexpected authorize URL: %s", target)
	}
}

func TestWechatOfficialSignatureValidation(t *testing.T) {
	input := WechatCallbackInput{Timestamp: "1786439000", Nonce: "nonce"}
	parts := []string{"message-token", input.Timestamp, input.Nonce}
	sort.Strings(parts)
	sum := sha1.Sum([]byte(strings.Join(parts, "")))
	input.Signature = hex.EncodeToString(sum[:])
	if !validWechatSignature("message-token", input) {
		t.Fatal("valid callback signature was rejected")
	}
	input.Signature = strings.Repeat("0", 40)
	if validWechatSignature("message-token", input) {
		t.Fatal("invalid callback signature was accepted")
	}
}

func TestWechatOfficialCreateAttemptUsesEnabledSettings(t *testing.T) {
	db := openWechatServiceTestDB(t)
	providerRepo := repository.NewAuthLoginProviderRepository(db)
	providerService := NewAuthLoginProviderService(providerRepo)
	if err := providerService.SeedDefaults(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := providerService.UpdateConfig(ProviderWechatOfficial, map[string]string{
		"app_id":        "wx-app",
		"app_secret":    "secret",
		"message_token": "callback-token",
	}, "system"); err != nil {
		t.Fatal(err)
	}
	if _, err := providerService.SetEnabled(ProviderWechatOfficial, true, "system"); err != nil {
		t.Fatal(err)
	}
	methods, err := providerService.ListLoginMethods()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, method := range methods {
		if method.Provider == ProviderWechatOfficial && method.Action == ProviderActionQRCode {
			found = true
		}
	}
	if !found {
		t.Fatalf("enabled WeChat Official Account method is missing: %+v", methods)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/cgi-bin/token", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("appid") != "wx-app" || r.URL.Query().Get("secret") != "secret" {
			t.Errorf("unexpected token query: %v", r.URL.Query())
		}
		_, _ = w.Write([]byte(`{"access_token":"access-token","expires_in":7200}`))
	})
	mux.HandleFunc("/cgi-bin/qrcode/create", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("access_token") != "access-token" {
			t.Errorf("unexpected QR token: %v", r.URL.Query())
		}
		_, _ = w.Write([]byte(`{"ticket":"qr-ticket","expire_seconds":300}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := NewAuthWechatOfficialService(providerService, repository.NewAuthWechatLoginAttemptRepository(db), nil)
	svc.client = server.Client()
	svc.apiBaseURL = server.URL
	svc.qrBaseURL = server.URL + "/showqrcode"
	result, err := svc.CreateAttempt(context.Background(), "/workspace/demo")
	if err != nil {
		t.Fatal(err)
	}
	if result.AttemptToken == "" || result.QRCodeURL != server.URL+"/showqrcode?ticket=qr-ticket" {
		t.Fatalf("unexpected attempt: %+v", result)
	}
	var count int64
	if err := db.Model(&model.AuthWechatLoginAttempt{}).Where("redirect_after = ?", "/workspace/demo").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("stored attempts = %d, want 1", count)
	}
}

func TestWechatOfficialAttemptIsPendingThenSingleUse(t *testing.T) {
	db := openWechatServiceTestDB(t)
	repo := repository.NewAuthWechatLoginAttemptRepository(db)
	now := time.Now()
	if err := repo.Create(&model.AuthWechatLoginAttempt{
		TokenHash:    "token-hash",
		SceneHash:    "scene-hash",
		ProviderCode: ProviderWechatOfficial,
		ExpiresAt:    now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Consume("token-hash", now); !errors.Is(err, repository.ErrWechatLoginAttemptPending) {
		t.Fatalf("pending consume error = %v", err)
	}
	if err := repo.MarkScanned("scene-hash", "wx-app:openid", "微信用户", "", now); err != nil {
		t.Fatal(err)
	}
	attempt, err := repo.Consume("token-hash", now)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.ExternalID != "wx-app:openid" || attempt.UsedAt == nil {
		t.Fatalf("unexpected consumed attempt: %+v", attempt)
	}
	if _, err := repo.Consume("token-hash", now); !errors.Is(err, repository.ErrWechatLoginAttemptInvalid) {
		t.Fatalf("second consume error = %v", err)
	}
}

func openWechatServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AuthLoginProvider{}, &model.AuthOAuthState{}, &model.AuthWechatLoginAttempt{}); err != nil {
		t.Fatal(err)
	}
	return db
}
