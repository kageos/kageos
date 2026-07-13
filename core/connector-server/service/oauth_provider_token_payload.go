package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

func oauth2TokenPayload(token *oauth2.Token, scopes []string) *OAuthTokenPayload {
	if token == nil {
		return nil
	}
	var expiry *time.Time
	if !token.Expiry.IsZero() {
		t := token.Expiry
		expiry = &t
	}
	scopeText := strings.Join(scopes, " ")
	if value := token.Extra("scope"); value != nil {
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				scopeText = typed
			}
		case []string:
			if len(typed) > 0 {
				scopeText = strings.Join(typed, " ")
			}
		}
	}
	raw := map[string]interface{}{
		"token_type":      token.TokenType,
		"expiry":          token.Expiry,
		"scopes":          scopeText,
		"access_present":  token.AccessToken != "",
		"refresh_present": token.RefreshToken != "",
	}
	rawData, _ := json.Marshal(raw)
	return &OAuthTokenPayload{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Scopes:       scopeText,
		Expiry:       expiry,
		RawResponse:  string(rawData),
	}
}

func parseTokenPayload(data []byte, scopes []string) (*OAuthTokenPayload, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	tokenRoot := raw
	if dataObj, ok := raw["data"].(map[string]interface{}); ok {
		tokenRoot = dataObj
	}
	accessToken := stringField(tokenRoot, "access_token", "accessToken")
	if accessToken == "" {
		return nil, fmt.Errorf("oauth token response 缺少 access_token")
	}
	expiry := tokenExpiry(tokenRoot)
	scopeText := stringField(tokenRoot, "scope", "scopes")
	if scopeText == "" {
		scopeText = strings.Join(scopes, " ")
	}
	return &OAuthTokenPayload{
		AccessToken:  accessToken,
		RefreshToken: stringField(tokenRoot, "refresh_token", "refreshToken"),
		TokenType:    stringField(tokenRoot, "token_type", "tokenType"),
		Scopes:       scopeText,
		Expiry:       expiry,
		RawResponse:  string(data),
	}, nil
}

func tokenExpiry(raw map[string]interface{}) *time.Time {
	seconds := numberField(raw, "expires_in", "expiresIn", "expire_in", "expireIn")
	if seconds <= 0 {
		return nil
	}
	t := time.Now().Add(time.Duration(seconds) * time.Second)
	return &t
}

func stringField(raw map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := raw[key]; ok {
			if value := oauthValueString(v); value != "" {
				return value
			}
		}
	}
	return ""
}

func numberField(raw map[string]interface{}, keys ...string) int64 {
	for _, key := range keys {
		if v, ok := raw[key]; ok {
			switch typed := v.(type) {
			case float64:
				return int64(typed)
			case int64:
				return typed
			case int:
				return int64(typed)
			case string:
				n, _ := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
				return n
			}
		}
	}
	return 0
}
