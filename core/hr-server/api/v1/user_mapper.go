package v1

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
)

// extractUsernameFromDisplayName 从显示名称中提取用户名
// 原因：前端可能传递的是显示名称格式（如 "sina(新那)"），其中括号前是用户名，括号内是昵称
// 为了支持用户通过前端展示的名称进行查询，需要提取括号前的用户名部分进行实际查询
// 示例：
//   - "sina(新那)" -> "sina"
//   - "sina" -> "sina"（没有括号时返回原字符串）
func extractUsernameFromDisplayName(displayName string) string {
	return strings.TrimSpace(strings.Split(displayName, "(")[0])
}

func (u *User) ensureSameCompany(c *gin.Context, target *model.User) error {
	requester := contextx.GetRequestUser(c)
	if requester == "" {
		return fmt.Errorf("未提供用户信息")
	}
	current, err := u.userService.GetUserByUsername(requester)
	if err != nil {
		return fmt.Errorf("当前用户不存在: %w", err)
	}
	if current.CompanyCode != target.CompanyCode {
		return fmt.Errorf("用户不存在")
	}
	return nil
}

// convertUserToDTO 将model.User转换为dto.UserInfo（基础版本，不包含详细信息）
func convertUserToDTO(user *model.User) *dto.UserInfo {
	return convertUserToDTOWithDetails(user, nil, nil)
}

// convertUserToDTOWithDetails 将model.User转换为dto.UserInfo（包含详细信息：部门名称和Leader显示名称）
// deptMap: 部门信息映射表，key 为 fullCodePath，value 为 Department（可选）
// leaderMap: Leader 信息映射表，key 为 username，value 为 User（可选）
func convertUserToDTOWithDetails(user *model.User, deptMap map[string]*model.Department, leaderMap map[string]*model.User) *dto.UserInfo {
	userInfo := &dto.UserInfo{
		ID:            user.ID,
		Username:      user.Username,
		Email:         user.Email,
		CompanyCode:   user.CompanyCode,
		RegisterType:  user.RegisterType,
		Avatar:        user.Avatar,
		Nickname:      user.Nickname,
		Signature:     user.Signature,
		Gender:        user.Gender,
		EmailVerified: user.EmailVerified,
		Status:        user.Status,
		CreatedAt:     time.Time(user.CreatedAt).Format(time.RFC3339),
	}

	// 组织架构信息（如果存在）
	if user.DepartmentFullPath != "" {
		userInfo.DepartmentFullPath = user.DepartmentFullPath

		// 从缓存中获取部门名称和完整名称路径
		if deptMap != nil {
			if dept, ok := deptMap[user.DepartmentFullPath]; ok {
				userInfo.DepartmentName = dept.Name
				// 如果 FullNamePath 为空，使用 Name 作为后备
				if dept.FullNamePath != "" {
					userInfo.DepartmentFullNamePath = dept.FullNamePath
				} else {
					// 如果 FullNamePath 为空（可能是旧数据），使用 Name 作为后备
					userInfo.DepartmentFullNamePath = dept.Name
				}
			}
		}
	}
	if user.LeaderUsername != "" {
		userInfo.LeaderUsername = user.LeaderUsername

		// 从缓存中获取 Leader 显示名称
		if leaderMap != nil {
			if leader, ok := leaderMap[user.LeaderUsername]; ok {
				// 构建显示名称：username(nickname) 或 username
				if leader.Nickname != "" {
					userInfo.LeaderDisplayName = fmt.Sprintf("%s(%s)", leader.Username, leader.Nickname)
				} else {
					userInfo.LeaderDisplayName = leader.Username
				}
			}
		}
	}

	return userInfo
}

// convertUsersToDTOBatch 批量转换用户列表为 DTO（包含详细信息）
func convertUsersToDTOBatch(ctx context.Context, users []*model.User, userService *service.UserService, departmentService *service.DepartmentService) []*dto.UserInfo {
	if len(users) == 0 {
		return []*dto.UserInfo{}
	}

	// 收集所有需要查询的部门路径和 Leader 用户名
	deptPaths := make([]string, 0)
	leaderUsernames := make([]string, 0)
	companyCodes := make([]string, 0)

	for _, user := range users {
		if user.DepartmentFullPath != "" {
			deptPaths = append(deptPaths, user.DepartmentFullPath)
		}
		if user.LeaderUsername != "" {
			leaderUsernames = append(leaderUsernames, user.LeaderUsername)
		}
		if user.CompanyCode != "" {
			companyCodes = append(companyCodes, user.CompanyCode)
		}
	}

	companyMap := make(map[string]*model.Company)
	if len(companyCodes) > 0 && userService != nil {
		uniqueCodes := make(map[string]bool)
		codeList := make([]string, 0, len(companyCodes))
		for _, code := range companyCodes {
			if !uniqueCodes[code] {
				uniqueCodes[code] = true
				codeList = append(codeList, code)
			}
		}
		companies, err := userService.GetCompaniesByCodes(codeList)
		if err != nil {
			logger.Warnf(ctx, "[convertUsersToDTOBatch] 批量查询企业信息失败: %v", err)
		} else {
			for _, company := range companies {
				companyMap[company.Code] = company
			}
		}
	}

	// 批量查询部门信息（从缓存）
	var deptMap map[string]*model.Department
	if len(deptPaths) > 0 && departmentService != nil {
		var err error
		deptMap, err = departmentService.GetDepartmentsByFullCodePaths(ctx, deptPaths)
		if err != nil {
			logger.Warnf(ctx, "[convertUsersToDTOBatch] 批量查询部门信息失败: %v", err)
			deptMap = nil
		}
	}

	// 批量查询 Leader 信息
	var leaderMap map[string]*model.User
	if len(leaderUsernames) > 0 && userService != nil {
		// 去重
		uniqueUsernames := make(map[string]bool)
		for _, username := range leaderUsernames {
			uniqueUsernames[username] = true
		}
		usernameList := make([]string, 0, len(uniqueUsernames))
		for username := range uniqueUsernames {
			usernameList = append(usernameList, username)
		}

		leaders, err := userService.GetUsersByUsernames(usernameList)
		if err != nil {
			logger.Warnf(ctx, "[convertUsersToDTOBatch] 批量查询 Leader 信息失败: %v", err)
			leaderMap = nil
		} else {
			leaderMap = make(map[string]*model.User, len(leaders))
			for _, leader := range leaders {
				leaderMap[leader.Username] = leader
			}
		}
	}

	// 批量转换
	userInfos := make([]*dto.UserInfo, 0, len(users))
	for _, user := range users {
		userInfo := convertUserToDTOWithDetails(user, deptMap, leaderMap)
		if company := companyMap[user.CompanyCode]; company != nil {
			userInfo.CompanyName = company.Name
			userInfo.CompanyLogoURL = company.LogoURL
		}
		userInfos = append(userInfos, userInfo)
	}

	return userInfos
}
