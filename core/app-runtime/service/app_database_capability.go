package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/dto"
)

func (s *AppDatabaseService) IssueCapability(user, app, version, router string) (*dto.AppDBCapability, error) {
	if !s.IsEnabled() {
		return nil, nil
	}
	nonce, err := randomToken(18)
	if err != nil {
		return nil, err
	}
	capability := &dto.AppDBCapability{
		User:      user,
		App:       app,
		Version:   version,
		Router:    router,
		ExpiresAt: time.Now().Add(appDBCapabilityTTL).Unix(),
		Nonce:     nonce,
	}
	capability.Signature = s.signCapability(capability)
	return capability, nil
}

func (s *AppDatabaseService) Resolve(ctx context.Context, req *dto.AppDBResolveReq) (*dto.AppDBResolveResp, error) {
	if !s.IsEnabled() {
		return nil, ErrAppDatabaseDisabled
	}
	if req == nil {
		return nil, fmt.Errorf("resolve request is nil")
	}
	access, err := normalizeAppDBAccess(req.Access)
	if err != nil {
		return nil, err
	}
	if err := s.validateCapability(req); err != nil {
		return nil, err
	}

	packagePath, err := normalizeAppDBPackagePath(req.PackagePath)
	if err != nil {
		return nil, err
	}

	record, passwords, err := s.ensurePackageDatabase(ctx, req.User, req.App, packagePath)
	if err != nil {
		return nil, err
	}

	databaseUser := record.DatabaseUser
	password := passwords.runtime
	if access == dto.AppDBAccessMigration {
		databaseUser = record.MigrationDatabaseUser
		password = passwords.migration
	}
	if strings.TrimSpace(databaseUser) == "" || strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("app database %s credentials are unavailable", access)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		databaseUser, password, s.cfg.Host, s.cfg.Port, record.DatabaseName)
	return &dto.AppDBResolveResp{
		Dialect:      "mysql",
		Access:       access,
		DatabaseName: record.DatabaseName,
		DSN:          dsn,
		MaxOpenConns: s.cfg.MaxOpenConns,
		MaxIdleConns: s.cfg.MaxIdleConns,
		MaxIdleTime:  s.cfg.MaxIdleTime,
		MaxLifetime:  s.cfg.MaxLifetime,
	}, nil
}

func (s *AppDatabaseService) validateCapability(req *dto.AppDBResolveReq) error {
	capability := req.Capability
	if capability == nil {
		return fmt.Errorf("missing app database capability")
	}
	if capability.User != req.User || capability.App != req.App {
		return fmt.Errorf("app database capability scope mismatch")
	}
	if capability.ExpiresAt <= time.Now().Unix() {
		return fmt.Errorf("app database capability expired")
	}
	if !hmac.Equal([]byte(capability.Signature), []byte(s.signCapability(capability))) {
		return fmt.Errorf("invalid app database capability signature")
	}
	access, err := normalizeAppDBAccess(req.Access)
	if err != nil {
		return err
	}
	if access == dto.AppDBAccessMigration && capability.Router != "" {
		return fmt.Errorf("app database migration access requires lifecycle capability")
	}
	if capability.Router != "" {
		expectedPackagePath, err := packagePathFromRouter(capability.Router)
		if err != nil {
			return err
		}
		actualPackagePath, err := normalizeAppDBPackagePath(req.PackagePath)
		if err != nil {
			return err
		}
		if expectedPackagePath != actualPackagePath {
			return fmt.Errorf("app database capability package mismatch")
		}
	}
	return nil
}

func (s *AppDatabaseService) signCapability(capability *dto.AppDBCapability) string {
	copy := *capability
	copy.Signature = ""
	payload, _ := json.Marshal(copy)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
