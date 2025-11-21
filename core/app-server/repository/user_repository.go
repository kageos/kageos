package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetUserByID 根据用户ID获取用户信息
func (r *UserRepository) GetUserByID(id int64) (*model.User, error) {
	var user model.User
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户信息
func (r *UserRepository) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail 根据邮箱获取用户信息
func (r *UserRepository) GetUserByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserWithHostAndNats 根据用户ID获取用户信息，并预加载host和nats信息
func (r *UserRepository) GetUserWithHostAndNats(id int64) (*model.User, error) {
	var user model.User
	err := r.db.Preload("Host.Nats").Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByUsernameWithHostAndNats 根据用户名获取用户信息，并预加载host和nats信息
func (r *UserRepository) GetUserByUsernameWithHostAndNats(username string) (*model.User, error) {
	var user model.User
	err := r.db.Preload("Host.Nats").Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByThirdPartyID 根据第三方平台ID和注册方式获取用户信息
func (r *UserRepository) GetUserByThirdPartyID(thirdPartyID, registerType string) (*model.User, error) {
	var user model.User
	err := r.db.Where("third_party_id = ? AND register_type = ?", thirdPartyID, registerType).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmailAndRegisterType 根据邮箱和注册方式获取用户信息
func (r *UserRepository) GetUserByEmailAndRegisterType(email, registerType string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ? AND register_type = ?", email, registerType).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUser 创建用户
func (r *UserRepository) CreateUser(user *model.User) error {
	return r.db.Create(user).Error
}

// UpdateUser 更新用户
func (r *UserRepository) UpdateUser(user *model.User) error {
	return r.db.Save(user).Error
}

// DeleteUser 删除用户
func (r *UserRepository) DeleteUser(id int64) error {
	return r.db.Delete(&model.User{}, id).Error
}

// SearchUsersFuzzy 模糊查询用户（根据用户名、邮箱或昵称）
func (r *UserRepository) SearchUsersFuzzy(keyword string, limit int) ([]*model.User, error) {
	var users []*model.User
	query := r.db.Where("username LIKE ? OR email LIKE ? OR nickname LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&users).Error
	return users, err
}

// GetUsersByUsernames 根据用户名列表批量获取用户信息
func (r *UserRepository) GetUsersByUsernames(usernames []string) ([]*model.User, error) {
	if len(usernames) == 0 {
		return []*model.User{}, nil
	}
	var users []*model.User
	err := r.db.Where("username IN ?", usernames).Find(&users).Error
	return users, err
}
