package repository

import (
	"context"
	"strings"

	"github.com/kageos/kageos/core/hr-server/model"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetUserByID 根据用户ID获取用户信息
func (r *UserRepository) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户信息
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail 根据邮箱获取用户信息
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CountUsers 统计用户总数（不包括已删除的用户）
func (r *UserRepository) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Count(&count).Error
	return count, err
}

// ⚠️ 注意：GetUserWithHostAndNats 和 GetUserByUsernameWithHostAndNats 方法不迁移
// 因为 Host 模型在 app-server，如果需要可以通过 API 调用获取

// GetUserByThirdPartyID 根据第三方平台ID和注册方式获取用户信息
func (r *UserRepository) GetUserByThirdPartyID(ctx context.Context, thirdPartyID, registerType string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("third_party_id = ? AND register_type = ?", thirdPartyID, registerType).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmailAndRegisterType 根据邮箱和注册方式获取用户信息
func (r *UserRepository) GetUserByEmailAndRegisterType(ctx context.Context, email, registerType string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ? AND register_type = ?", email, registerType).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUser 创建用户
func (r *UserRepository) CreateUser(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// UpdateUser 更新用户
func (r *UserRepository) UpdateUser(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// DeleteUser 删除用户
func (r *UserRepository) DeleteUser(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.User{}, id).Error
}

// SearchUsersFuzzy 模糊查询用户（根据用户名、邮箱或昵称）
func (r *UserRepository) SearchUsersFuzzy(ctx context.Context, keyword string, limit int) ([]*model.User, error) {
	var users []*model.User
	query := r.db.WithContext(ctx).Where("username LIKE ? OR email LIKE ? OR nickname LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&users).Error
	return users, err
}

// SearchUsersFuzzyByCompany 模糊查询同企业用户（根据用户名、邮箱或昵称）。
func (r *UserRepository) SearchUsersFuzzyByCompany(ctx context.Context, companyCode, keyword string, limit int) ([]*model.User, error) {
	var users []*model.User
	companyCode = strings.TrimSpace(companyCode)
	keyword = strings.TrimSpace(keyword)
	query := r.db.WithContext(ctx).Where("company_code = ?", companyCode)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("(username LIKE ? OR email LIKE ? OR nickname LIKE ?)", like, like, like)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&users).Error
	return users, err
}

// ListUsersForSystem 分页查询全局用户，仅供 system 管理入口使用。
func (r *UserRepository) ListUsersForSystem(ctx context.Context, keyword, companyCode, status, registerType string, page, pageSize int) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	keyword = strings.ToLower(strings.TrimSpace(keyword))
	companyCode = strings.TrimSpace(companyCode)
	status = strings.TrimSpace(status)
	registerType = strings.TrimSpace(registerType)

	query := r.db.WithContext(ctx).Model(&model.User{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("(LOWER(username) LIKE ? OR LOWER(email) LIKE ? OR LOWER(nickname) LIKE ?)", like, like, like)
	}
	if companyCode != "" {
		query = query.Where("company_code = ?", companyCode)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if registerType != "" {
		query = query.Where("register_type = ?", registerType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*model.User{}, 0, nil
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// GetUsersByUsernames 根据用户名列表批量获取用户信息
func (r *UserRepository) GetUsersByUsernames(ctx context.Context, usernames []string) ([]*model.User, error) {
	if len(usernames) == 0 {
		return []*model.User{}, nil
	}
	var users []*model.User
	err := r.db.WithContext(ctx).Where("username IN ?", usernames).Find(&users).Error
	return users, err
}

func (r *UserRepository) GetUsersByUsernamesAndCompany(ctx context.Context, usernames []string, companyCode string) ([]*model.User, error) {
	if len(usernames) == 0 {
		return []*model.User{}, nil
	}
	var users []*model.User
	err := r.db.WithContext(ctx).Where("company_code = ? AND username IN ?", strings.TrimSpace(companyCode), usernames).Find(&users).Error
	return users, err
}

// CountUsersByDepartmentFullPath 根据部门完整路径统计用户数量
func (r *UserRepository) CountUsersByDepartmentFullPath(ctx context.Context, departmentFullPath string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("department_full_path = ?", departmentFullPath).Count(&count).Error
	return count, err
}

// GetUsersByDepartmentFullPath 根据部门完整路径获取用户列表
func (r *UserRepository) GetUsersByDepartmentFullPath(ctx context.Context, departmentFullPath string) ([]*model.User, error) {
	var users []*model.User
	err := r.db.WithContext(ctx).Where("department_full_path = ?", departmentFullPath).Find(&users).Error
	return users, err
}
