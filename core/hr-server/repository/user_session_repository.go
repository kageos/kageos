package repository

import (
	"context"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/pkg/gormx/models"
	"gorm.io/gorm"
)

type UserSessionRepository struct {
	db *gorm.DB
}

func NewUserSessionRepository(db *gorm.DB) *UserSessionRepository {
	return &UserSessionRepository{db: db}
}

// CreateUserSession 创建用户会话
func (r *UserSessionRepository) CreateUserSession(ctx context.Context, userID int64, token, refreshToken string, expiresAt models.Time, userAgent, ipAddress string) error {
	session := model.UserSession{
		UserID:       userID,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		UserAgent:    userAgent,
		IPAddress:    ipAddress,
	}
	return r.db.WithContext(ctx).Create(&session).Error
}

// GetUserSessionByToken 根据token获取用户会话
func (r *UserSessionRepository) GetUserSessionByToken(ctx context.Context, token string) (*model.UserSession, error) {
	var session model.UserSession
	err := r.db.WithContext(ctx).Where("token = ? AND is_active = true AND expires_at > ?", token, models.Time(time.Now())).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetUserSessionByRefreshToken 根据refresh token获取用户会话
func (r *UserSessionRepository) GetUserSessionByRefreshToken(ctx context.Context, refreshToken string) (*model.UserSession, error) {
	var session model.UserSession
	err := r.db.WithContext(ctx).Where("refresh_token = ? AND is_active = true AND expires_at > ?", refreshToken, models.Time(time.Now())).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// DeactivateUserSession 停用用户会话
func (r *UserSessionRepository) DeactivateUserSession(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).Model(&model.UserSession{}).Where("token = ?", token).Update("is_active", false).Error
}

// DeactivateAllUserSessions 停用用户的所有会话
func (r *UserSessionRepository) DeactivateAllUserSessions(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).Model(&model.UserSession{}).Where("user_id = ?", userID).Update("is_active", false).Error
}

// DeleteExpiredSessions 删除过期的会话
func (r *UserSessionRepository) DeleteExpiredSessions(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("expires_at < ?", models.Time(time.Now())).Delete(&model.UserSession{}).Error
}

// UpdateUserSessionTokens 更新用户会话的token和refresh token
func (r *UserSessionRepository) UpdateUserSessionTokens(ctx context.Context, sessionID int64, token, refreshToken string) error {
	return r.db.WithContext(ctx).Model(&model.UserSession{}).Where("id = ?", sessionID).Updates(map[string]interface{}{
		"token":         token,
		"refresh_token": refreshToken,
	}).Error
}

// GetUserSessionByID 根据ID获取用户会话
func (r *UserSessionRepository) GetUserSessionByID(ctx context.Context, id int64) (*model.UserSession, error) {
	var session model.UserSession
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetUserSessionsByUserID 根据用户ID获取所有会话
func (r *UserSessionRepository) GetUserSessionsByUserID(ctx context.Context, userID int64) ([]*model.UserSession, error) {
	var sessions []*model.UserSession
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

// GetActiveSessionsByUserID 根据用户ID获取所有活跃会话
func (r *UserSessionRepository) GetActiveSessionsByUserID(ctx context.Context, userID int64) ([]*model.UserSession, error) {
	var sessions []*model.UserSession
	now := models.Time(time.Now())
	err := r.db.WithContext(ctx).Where("user_id = ? AND is_active = true AND expires_at > ?", userID, now).Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	return sessions, nil
}
