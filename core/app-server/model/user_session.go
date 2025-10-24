package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"gorm.io/gorm"
)

// UserSession 用户会话记录
type UserSession struct {
	ID        int64          `json:"id" gorm:"primary_key"`
	CreatedAt models.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt models.Time    `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	UserID       int64       `json:"user_id" gorm:"column:user_id;type:bigint;not null"`
	Token        string      `json:"token" gorm:"column:token;type:varchar(500);uniqueIndex;not null"`
	RefreshToken string      `json:"refresh_token" gorm:"column:refresh_token;type:varchar(500);uniqueIndex;not null"`
	ExpiresAt    models.Time `json:"expires_at" gorm:"column:expires_at;type:datetime;not null"`
	UserAgent    string      `json:"user_agent" gorm:"column:user_agent;type:varchar(500)"`
	IPAddress    string      `json:"ip_address" gorm:"column:ip_address;type:varchar(45)"`
	IsActive     bool        `json:"is_active" gorm:"column:is_active;type:boolean;default:true"`

	// 关联字段
	User *User `json:"user" gorm:"foreignKey:UserID;references:ID"`
}

func (UserSession) TableName() string {
	return "user_session"
}

// CreateUserSession 创建用户会话
func CreateUserSession(userID int64, token, refreshToken string, expiresAt models.Time, userAgent, ipAddress string) error {
	session := UserSession{
		UserID:       userID,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		UserAgent:    userAgent,
		IPAddress:    ipAddress,
	}
	return DB.Create(&session).Error
}

// GetUserSessionByToken 根据token获取用户会话
func GetUserSessionByToken(token string) (*UserSession, error) {
	var session UserSession
	err := DB.Where("token = ? AND is_active = true AND expires_at > ?", token, models.Time{}).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetUserSessionByRefreshToken 根据refresh token获取用户会话
func GetUserSessionByRefreshToken(refreshToken string) (*UserSession, error) {
	var session UserSession
	err := DB.Where("refresh_token = ? AND is_active = true AND expires_at > ?", refreshToken, models.Time{}).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// DeactivateUserSession 停用用户会话
func DeactivateUserSession(token string) error {
	return DB.Model(&UserSession{}).Where("token = ?", token).Update("is_active", false).Error
}

// DeactivateAllUserSessions 停用用户的所有会话
func DeactivateAllUserSessions(userID int64) error {
	return DB.Model(&UserSession{}).Where("user_id = ?", userID).Update("is_active", false).Error
}

// DeleteExpiredSessions 删除过期的会话
func DeleteExpiredSessions() error {
	return DB.Where("expires_at < ?", models.Time{}).Delete(&UserSession{}).Error
}
