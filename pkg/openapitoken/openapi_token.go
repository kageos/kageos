package openapitoken

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/gormx/models"
	"gorm.io/gorm"
)

const TokenPrefix = "kgos_"

type Store struct {
	db *gorm.DB
}

type OpenAPIToken struct {
	ID                int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt         models.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         models.Time    `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
	OwnerUserID       int64          `json:"owner_user_id" gorm:"column:owner_user_id;type:bigint;index"`
	OwnerUsername     string         `json:"owner_username" gorm:"column:owner_username;type:varchar(255);not null;index"`
	Name              string         `json:"name" gorm:"column:name;type:varchar(120);not null"`
	TokenPrefix       string         `json:"token_prefix" gorm:"column:token_prefix;type:varchar(32);not null;index"`
	TokenHash         string         `json:"-" gorm:"column:token_hash;type:char(64);not null;uniqueIndex"`
	ExpiresAt         *time.Time     `json:"expires_at,omitempty" gorm:"column:expires_at"`
	RevokedAt         *time.Time     `json:"revoked_at,omitempty" gorm:"column:revoked_at;index"`
	LastUsedAt        *time.Time     `json:"last_used_at,omitempty" gorm:"column:last_used_at"`
	LastUsedIP        string         `json:"last_used_ip,omitempty" gorm:"column:last_used_ip;type:varchar(80)"`
	LastUsedUserAgent string         `json:"last_used_user_agent,omitempty" gorm:"column:last_used_user_agent;type:varchar(500)"`
}

func (OpenAPIToken) TableName() string {
	return "openapi_tokens"
}

type CreateInput struct {
	OwnerUsername      string
	OwnerUserID        int64
	OwnerEmail         string
	CompanyCode        string
	CompanyName        string
	CompanyLogoURL     string
	DepartmentFullPath string
	LeaderUsername     string
	Name               string
	ExpiresAt          *time.Time
}

type CreateResult struct {
	Token  OpenAPIToken
	Secret string
}

type Principal struct {
	TokenID            int64
	UserID             int64
	Username           string
	Email              string
	CompanyCode        string
	CompanyName        string
	CompanyLogoURL     string
	DepartmentFullPath string
}

func NewStore(database *gorm.DB) (*Store, error) {
	if database == nil {
		return nil, errors.New("openapi token db is nil")
	}
	if err := Migrate(database); err != nil {
		return nil, err
	}
	return &Store{db: database}, nil
}

func Migrate(database *gorm.DB) error {
	return database.AutoMigrate(&OpenAPIToken{})
}

func (s *Store) Create(input CreateInput) (*CreateResult, error) {
	database := s.database()
	if database == nil {
		return nil, errors.New("openapi token db is not configured")
	}
	ownerUsername := strings.TrimSpace(input.OwnerUsername)
	if ownerUsername == "" {
		return nil, errors.New("owner username is required")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("token name is required")
	}

	secret, err := auth.NewJWTService().GenerateOpenAPITokenWithContext(auth.UserTokenContext{
		UserID:             input.OwnerUserID,
		Username:           ownerUsername,
		Email:              strings.TrimSpace(input.OwnerEmail),
		CompanyCode:        strings.TrimSpace(input.CompanyCode),
		CompanyName:        strings.TrimSpace(input.CompanyName),
		CompanyLogoURL:     strings.TrimSpace(input.CompanyLogoURL),
		DepartmentFullPath: strings.TrimSpace(input.DepartmentFullPath),
		LeaderUsername:     strings.TrimSpace(input.LeaderUsername),
	}, input.ExpiresAt)
	if err != nil {
		return nil, err
	}
	record := OpenAPIToken{
		OwnerUserID:   input.OwnerUserID,
		OwnerUsername: ownerUsername,
		Name:          name,
		TokenPrefix:   displayPrefix(secret),
		TokenHash:     hashToken(secret),
		ExpiresAt:     input.ExpiresAt,
	}
	if err := database.Create(&record).Error; err != nil {
		return nil, err
	}
	return &CreateResult{Token: record, Secret: secret}, nil
}

func (s *Store) List(ownerUsername string) ([]OpenAPIToken, error) {
	database := s.database()
	if database == nil {
		return nil, errors.New("openapi token db is not configured")
	}
	var tokens []OpenAPIToken
	err := database.
		Where("owner_username = ?", strings.TrimSpace(ownerUsername)).
		Order("created_at DESC").
		Find(&tokens).Error
	return tokens, err
}

func (s *Store) Revoke(ownerUsername string, id int64) error {
	database := s.database()
	if database == nil {
		return errors.New("openapi token db is not configured")
	}
	now := time.Now()
	result := database.Model(&OpenAPIToken{}).
		Where("id = ? AND owner_username = ? AND revoked_at IS NULL", id, strings.TrimSpace(ownerUsername)).
		Update("revoked_at", &now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Store) Validate(rawToken, ip, userAgent string) (*Principal, error) {
	rawToken = strings.TrimSpace(rawToken)
	claims, err := auth.NewJWTService().ValidateToken(rawToken)
	if err != nil {
		return nil, err
	}
	principal := &Principal{
		TokenID:        0,
		UserID:         claims.UserID,
		Username:       claims.Username,
		Email:          claims.Email,
		CompanyCode:    claims.CompanyCode,
		CompanyName:    claims.CompanyName,
		CompanyLogoURL: claims.CompanyLogoURL,
	}
	if claims.DepartmentFullPath != nil {
		principal.DepartmentFullPath = *claims.DepartmentFullPath
	}
	database := s.database()
	if database == nil {
		return principal, nil
	}
	tokenPrefix := displayPrefix(rawToken)
	var candidates []OpenAPIToken
	if err := database.Where("token_prefix = ? AND revoked_at IS NULL", tokenPrefix).Find(&candidates).Error; err != nil {
		return principal, nil
	}
	rawHash := hashToken(rawToken)
	for _, candidate := range candidates {
		if subtle.ConstantTimeCompare([]byte(candidate.TokenHash), []byte(rawHash)) != 1 {
			continue
		}
		if candidate.ExpiresAt != nil && time.Now().After(*candidate.ExpiresAt) {
			return nil, errors.New("openapi token expired")
		}
		now := time.Now()
		_ = database.Model(&OpenAPIToken{}).Where("id = ?", candidate.ID).Updates(map[string]interface{}{
			"last_used_at":         &now,
			"last_used_ip":         strings.TrimSpace(ip),
			"last_used_user_agent": truncate(strings.TrimSpace(userAgent), 500),
		}).Error
		principal.TokenID = candidate.ID
		return principal, nil
	}
	return principal, nil
}

func BearerToken(authorizationHeader string) string {
	header := strings.TrimSpace(authorizationHeader)
	if len(header) < 7 || !strings.EqualFold(header[:7], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(header[7:])
}

func (s *Store) database() *gorm.DB {
	if s == nil {
		return nil
	}
	return s.db
}

func displayPrefix(token string) string {
	if len(token) <= 14 {
		return token
	}
	return token[:14]
}

func hashToken(token string) string {
	secret := strings.TrimSpace(config.GetGlobalSharedConfig().JWT.Secret)
	sum := sha256.Sum256([]byte(secret + ":" + token))
	return hex.EncodeToString(sum[:])
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
