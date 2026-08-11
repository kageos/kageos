package service

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestKageOSAuthProviderUsesFirstPartyEndpointsAndPKCE(t *testing.T) {
	factory, ok := GetOAuthProvider(ProviderKageOSAuth)
	if !ok || !factory.UsePKCE {
		t.Fatal("KageOS Auth provider must be registered with PKCE")
	}
	config, err := factory.OAuth2Config(map[string]string{
		"client_id": "kageos-app", "client_secret": strings.Repeat("s", 32),
		"redirect_url": "https://app.kageos.com/hr/api/v1/auth/kageos/callback",
	})
	if err != nil {
		t.Fatal(err)
	}
	if config.Endpoint.AuthURL != "https://auth.kageos.com/api/v1/oauth/authorize" || config.Endpoint.TokenURL != "https://auth.kageos.com/api/v1/oauth/token" {
		t.Fatalf("unexpected KageOS Auth endpoints: %#v", config.Endpoint)
	}
	if config.Endpoint.AuthStyle != oauth2.AuthStyleInHeader {
		t.Fatalf("auth style = %v, want HTTP Basic", config.Endpoint.AuthStyle)
	}
}

func TestRegisterOAuthLoginProviderRegistersSeedAndFactory(t *testing.T) {
	restoreOAuthProviderRegistriesForTest(t)

	RegisterOAuthLoginProvider(OAuthLoginProvider{
		Seed: AuthProviderSeed{
			Code:          "test_oidc",
			Name:          "Test OIDC",
			Action:        ProviderActionRedirect,
			AuthorizePath: "/hr/api/v1/auth/test_oidc/authorize",
			CallbackPath:  "/hr/api/v1/auth/test_oidc/callback",
			Fields: []AuthProviderFieldDef{
				{Key: "issuer_url", Label: "Issuer URL", Type: "url", Required: true},
			},
		},
		Factory: OAuthProviderFactory{
			OAuth2Config: func(map[string]string) (*oauth2.Config, error) {
				return &oauth2.Config{}, nil
			},
			FetchProfile: func(context.Context, *http.Client) (*OAuthProfile, error) {
				return &OAuthProfile{ExternalID: "subject"}, nil
			},
		},
		Aliases: []string{"test-oidc"},
	})

	if _, ok := GetOAuthProvider("test_oidc"); !ok {
		t.Fatal("OAuth provider factory was not registered")
	}
	if code, ok := LookupOAuthProviderCode("test-oidc"); !ok || code != "test_oidc" {
		t.Fatalf("alias lookup = %q, %v; want test_oidc, true", code, ok)
	}
	seed, ok := LookupAuthProviderSeed("test_oidc")
	if !ok {
		t.Fatal("auth provider seed was not registered")
	}
	if seed.AuthorizePath != "/hr/api/v1/auth/test_oidc/authorize" || len(seed.Fields) != 1 {
		t.Fatalf("unexpected seed: %#v", seed)
	}
}

func TestRegisterOAuthLoginProviderRejectsDuplicateCode(t *testing.T) {
	restoreOAuthProviderRegistriesForTest(t)

	provider := OAuthLoginProvider{
		Seed: AuthProviderSeed{
			Code:   "test_oidc",
			Name:   "Test OIDC",
			Action: ProviderActionRedirect,
		},
		Factory: OAuthProviderFactory{
			OAuth2Config: func(map[string]string) (*oauth2.Config, error) {
				return &oauth2.Config{}, nil
			},
			FetchProfile: func(context.Context, *http.Client) (*OAuthProfile, error) {
				return &OAuthProfile{ExternalID: "subject"}, nil
			},
		},
	}
	RegisterOAuthLoginProvider(provider)
	mustPanic(t, func() {
		RegisterOAuthLoginProvider(provider)
	})
}

func restoreOAuthProviderRegistriesForTest(t *testing.T) {
	t.Helper()

	oauthProviderRegistry.RLock()
	providers := make(map[string]OAuthProviderFactory, len(oauthProviderRegistry.providers))
	for code, factory := range oauthProviderRegistry.providers {
		providers[code] = factory
	}
	aliases := make(map[string]string, len(oauthProviderRegistry.aliases))
	for alias, code := range oauthProviderRegistry.aliases {
		aliases[alias] = code
	}
	oauthProviderRegistry.RUnlock()

	authProviderSeedRegistry.RLock()
	seeds := make(map[string]AuthProviderSeed, len(authProviderSeedRegistry.seeds))
	for code, seed := range authProviderSeedRegistry.seeds {
		seeds[code] = seed
	}
	authProviderSeedRegistry.RUnlock()

	t.Cleanup(func() {
		oauthProviderRegistry.Lock()
		oauthProviderRegistry.providers = providers
		oauthProviderRegistry.aliases = aliases
		oauthProviderRegistry.Unlock()

		authProviderSeedRegistry.Lock()
		authProviderSeedRegistry.seeds = seeds
		authProviderSeedRegistry.Unlock()
	})
}

func mustPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}
