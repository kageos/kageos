package repository

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/hr-server/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"gorm.io/gorm"
)

type UserSessionRepository struct {
	db *gorm.DB
}

func NewUserSessionRepository(db *gorm.DB) *UserSessionRepository {
	return &UserSessionRepository{db: db}
}

// CreateUserSession 创建用户会话
func (r *UserSessionRepository) CreateUserSession(userID int64, token, refreshToken string, expiresAt models.Time, userAgent, ipAddress string) error {
	session := model.UserSession{
		UserID:       userID,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		UserAgent:    userAgent,
		IPAddress:    ipAddress,
	}
	return r.db.Create(&session).Error
}

// GetUserSessionByToken 根据token获取用户会话
func (r *UserSessionRepository) GetUserSessionByToken(token string) (*model.UserSession, error) {
	var session model.UserSession
	err := r.db.Where("token = ? AND is_active = true AND expires_at > ?", token, models.Time{}).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetUserSessionByRefreshToken 根据refresh token获取用户会话
func (r *UserSessionRepository) GetUserSessionByRefreshToken(refreshToken string) (*model.UserSession, error) {
	var session model.UserSession
	err := r.db.Where("refresh_token = ? AND is_active = true AND expires_at > ?", refreshToken, models.Time{}).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// DeactivateUserSession 停用用户会话
func (r *UserSessionRepository) DeactivateUserSession(token string) error {
	return r.db.Model(&model.UserSession{}).Where("token = ?", token).Update("is_active", false).Error
}

// DeactivateAllUserSessions 停用用户的所有会话
func (r *UserSessionRepository) DeactivateAllUserSessions(userID int64) error {
	return r.db.Model(&model.UserSession{}).Where("user_id = ?", userID).Update("is_active", false).Error
}

// DeleteExpiredSessions 删除过期的会话
func (r *UserSessionRepository) DeleteExpiredSessions() error {
	return r.db.Where("expires_at < ?", models.Time{}).Delete(&model.UserSession{}).Error
}

// UpdateUserSessionTokens 更新用户会话的token和refresh token
func (r *UserSessionRepository) UpdateUserSessionTokens(sessionID int64, token, refreshToken string) error {
	return r.db.Model(&model.UserSession{}).Where("id = ?", sessionID).Updates(map[string]interface{}{
		"token":         token,
		"refresh_token": refreshToken,
	}).Error
}

// GetUserSessionByID 根据ID获取用户会话
func (r *UserSessionRepository) GetUserSessionByID(id int64) (*model.UserSession, error) {
	var session model.UserSession
	err := r.db.Where("id = ?", id).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetUserSessionsByUserID 根据用户ID获取所有会话
func (r *UserSessionRepository) GetUserSessionsByUserID(userID int64) ([]*model.UserSession, error) {
	var sessions []*model.UserSession
	err := r.db.Where("user_id = ?", userID).Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

// GetActiveSessionsByUserID 根据用户ID获取所有活跃会话
func (r *UserSessionRepository) GetActiveSessionsByUserID(userID int64) ([]*model.UserSession, error) {
	var sessions []*model.UserSession
	now := models.Time(time.Now())
	err := r.db.Where("user_id = ? AND is_active = true AND expires_at > ?", userID, now).Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	return sessions, nil
}
