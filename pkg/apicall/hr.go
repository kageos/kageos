package apicall

import (
	"context"
	"net/url"
	"strconv"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

// GetUserByUsername 根据用户名获取用户信息（app-server -> hr-server）
func GetUserByUsername(ctx context.Context, req *dto.QueryUserReq) (*dto.UserInfo, error) {
	// 构建查询参数
	path := "/hr/api/v1/user/query"
	params := url.Values{}
	params.Set("username", req.Username)

	result, err := GetAPI[*dto.QueryUserResp](ctx, path, params)
	if err != nil {
		return nil, err
	}
	return &result.User, nil
}

// GetUsersByUsernames 批量获取用户信息（app-server -> hr-server）
func GetUsersByUsernames(ctx context.Context, req *dto.GetUsersByUsernamesReq) ([]dto.UserInfo, error) {
	result, err := PostAPI[*dto.GetUsersByUsernamesReq, *dto.GetUsersByUsernamesResp](ctx, "/hr/api/v1/users", req)
	if err != nil {
		return nil, err
	}
	return result.Users, nil
}

// SearchUsersFuzzy 模糊查询用户（app-server -> hr-server）
func SearchUsersFuzzy(ctx context.Context, req *dto.SearchUsersFuzzyReq) ([]dto.UserInfo, error) {
	// 构建查询参数
	path := "/hr/api/v1/user/search_fuzzy"
	params := url.Values{}
	params.Set("keyword", req.Keyword)
	if req.Limit > 0 {
		params.Set("limit", strconv.Itoa(req.Limit))
	}

	result, err := GetAPI[*dto.SearchUsersFuzzyResp](ctx, path, params)
	if err != nil {
		return nil, err
	}
	return result.Users, nil
}
