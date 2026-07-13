package service

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"

	"github.com/kageos/kageos/dto"
	"gorm.io/gorm"
)

func (s *AppDatabaseService) lockFor(key string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	lock := s.keyLocks[key]
	if lock == nil {
		lock = &sync.Mutex{}
		s.keyLocks[key] = lock
	}
	return lock
}

func normalizeAppDBPackagePath(packagePath string) (string, error) {
	cleanPath, err := cleanPackagePath(packagePath)
	if err != nil {
		return "", err
	}
	if cleanPath == "" {
		return appDBRootPackage, nil
	}
	return cleanPath, nil
}

func cleanPackagePath(packagePath string) (string, error) {
	if packagePath != strings.TrimSpace(packagePath) {
		return "", fmt.Errorf("invalid app database package path %q: leading or trailing spaces are not allowed", packagePath)
	}
	cleanPath := strings.Trim(packagePath, "/")
	if cleanPath == "" || cleanPath == appDBRootPackage {
		return "", nil
	}
	if strings.Contains(cleanPath, `\`) {
		return "", fmt.Errorf("invalid app database package path %q: backslash is not allowed", packagePath)
	}
	if strings.Contains(cleanPath, "//") {
		return "", fmt.Errorf("invalid app database package path %q: empty path segment is not allowed", packagePath)
	}
	if normalized := path.Clean(cleanPath); normalized != cleanPath {
		return "", fmt.Errorf("invalid app database package path %q: dot segments are not allowed", packagePath)
	}
	parts := strings.Split(cleanPath, "/")
	for _, part := range parts {
		if err := validateGoPackagePathSegment(part); err != nil {
			return "", fmt.Errorf("invalid app database package path %q: %w", packagePath, err)
		}
	}
	return strings.Join(parts, "/"), nil
}

func normalizeAppDBAccess(access string) (string, error) {
	access = strings.TrimSpace(strings.ToLower(access))
	if access == "" {
		return dto.AppDBAccessRuntime, nil
	}
	switch access {
	case dto.AppDBAccessRuntime, dto.AppDBAccessMigration:
		return access, nil
	default:
		return "", fmt.Errorf("unsupported app database access: %s", access)
	}
}

func runtimeDatabaseUserName(prefix, suffix string) string {
	return prefix + suffix
}

func migrationDatabaseUserName(prefix, suffix string) string {
	return prefix + "m_" + suffix
}

func packagePathFromRouter(router string) (string, error) {
	if router != strings.TrimSpace(router) {
		return "", fmt.Errorf("invalid app database router %q: leading or trailing spaces are not allowed", router)
	}
	cleanRouter := strings.Trim(router, "/")
	if cleanRouter == "" {
		return appDBRootPackage, nil
	}
	if strings.Contains(cleanRouter, `\`) {
		return "", fmt.Errorf("invalid app database router %q: backslash is not allowed", router)
	}
	if strings.Contains(cleanRouter, "//") {
		return "", fmt.Errorf("invalid app database router %q: empty path segment is not allowed", router)
	}
	if normalized := path.Clean(cleanRouter); normalized != cleanRouter {
		return "", fmt.Errorf("invalid app database router %q: dot segments are not allowed", router)
	}
	parts := strings.Split(cleanRouter, "/")
	for _, part := range parts {
		if part != strings.TrimSpace(part) {
			return "", fmt.Errorf("invalid app database router %q: path segment spaces are not allowed", router)
		}
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("invalid app database router %q: invalid path segment %q", router, part)
		}
	}
	if len(parts) <= 1 {
		return appDBRootPackage, nil
	}
	packageParts := parts[:len(parts)-1]
	for _, part := range packageParts {
		if err := validateGoPackagePathSegment(part); err != nil {
			return "", fmt.Errorf("invalid app database router %q: %w", router, err)
		}
	}
	return strings.Join(packageParts, "/"), nil
}

func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func base62Encode(n uint64) string {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 12)
	for n > 0 {
		buf = append([]byte{alphabet[n%62]}, buf...)
		n /= 62
	}
	return string(buf)
}

func quoteMySQLIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func quoteMySQLString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func isNoSuchGrantError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "no such grant") || strings.Contains(text, "there is no such grant")
}

func closeGORM(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}
