package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"strings"

	"github.com/kageos/kageos/pkg/config"
	"golang.org/x/oauth2"
)

func (r *OAuthProviderRegistry) BuildAuthURL(provider config.ConnectorOAuthProviderConfig, redirectURL, state, codeChallenge string, scopes []string) string {
	return connectorAdapterFor(provider.Code).BuildAuthorizeURL(provider, redirectURL, state, codeChallenge, scopes)
}

func buildOAuthAuthorizeURL(provider config.ConnectorOAuthProviderConfig, redirectURL, state, codeChallenge string, scopes []string) string {
	conf := oauth2Config(provider, redirectURL, scopes)
	opts := []oauth2.AuthCodeOption{}
	if providerUsesAccessTypeOffline(provider) {
		opts = append(opts, oauth2.AccessTypeOffline)
	}
	if providerUsesPKCE(provider) {
		opts = append(opts,
			oauth2.SetAuthURLParam("code_challenge", codeChallenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		)
	}
	for key, value := range provider.ExtraAuthParams {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			opts = append(opts, oauth2.SetAuthURLParam(key, value))
		}
	}
	return conf.AuthCodeURL(state, opts...)
}

func newOAuthState() (string, error) {
	return randomBase64URL(32)
}

func newPKCEVerifier() (string, string, error) {
	verifier, err := randomBase64URL(32)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func randomBase64URL(size int) (string, error) {
	data := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}
