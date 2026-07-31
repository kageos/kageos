package apicall

import (
	"context"

	"github.com/kageos/kageos/dto"
)

// GetUserByUsername 根据用户名获取用户信息（app-server -> hr-server）
func GetUserByUsername(ctx context.Context, req *dto.QueryUserReq) (*dto.UserInfo, error) {
	result, err := GetAPI[*dto.QueryUserResp](ctx, "/hr/api/v1/user/query", buildQueryParams(
		withTrimmedQueryValue("username", req.Username),
	))
	if err != nil {
		return nil, err
	}
	return &result.User, nil
}

// GetUsersByUsernames 批量获取当前实例内的用户信息。
func GetUsersByUsernames(ctx context.Context, usernames []string) ([]dto.UserInfo, error) {
	if len(usernames) == 0 {
		return nil, nil
	}
	result, err := PostAPI[*dto.GetUsersByUsernamesReq, *dto.GetUsersByUsernamesResp](ctx, "/hr/api/v1/users", &dto.GetUsersByUsernamesReq{
		Usernames: usernames,
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.Users, nil
}

// GetDepartmentsByPaths 根据部门 full_code_path 列表批量获取部门信息（app-server -> hr-server）
// 用于解析部门中文名称路径（FullNamePath）供展示；存储/逻辑仍用 full_code_path。
func GetDepartmentsByPaths(ctx context.Context, fullCodePaths []string) (*dto.GetDepartmentsByPathsResp, error) {
	if len(fullCodePaths) == 0 {
		return &dto.GetDepartmentsByPathsResp{Departments: nil}, nil
	}
	return GetAPI[*dto.GetDepartmentsByPathsResp](ctx, "/hr/api/v1/departments", buildQueryParams(
		withCSVQueryValue("full_code_paths", fullCodePaths),
	))
}
