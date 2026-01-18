package apicall

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/serviceconfig"
)

// GetUserByUsername 根据用户名获取用户信息（app-server -> hr-server）
func GetUserByUsername(header *Header, username string) (*dto.UserInfo, error) {
	// 构建查询参数
	path := "/hr/api/v1/user/query"
	params := url.Values{}
	params.Set("username", username)

	// 构建完整 URL
	gatewayURL := serviceconfig.GetGatewayURL()
	fullURL := fmt.Sprintf("%s%s?%s", gatewayURL, path, params.Encode())

	result, err := callAPIWithURL[dto.QueryUserResp](http.MethodGet, fullURL, header, nil)
	if err != nil {
		return nil, err
	}
	return &result.Data.User, nil
}

// GetUsersByUsernames 批量获取用户信息（app-server -> hr-server）
func GetUsersByUsernames(header *Header, usernames []string) ([]dto.UserInfo, error) {
	req := &dto.GetUsersByUsernamesReq{
		Usernames: usernames,
	}

	result, err := callAPI[dto.GetUsersByUsernamesResp](http.MethodPost, "/hr/api/v1/users", header, req)
	if err != nil {
		return nil, err
	}
	return result.Data.Users, nil
}

// SearchUsersFuzzy 模糊查询用户（app-server -> hr-server）
func SearchUsersFuzzy(header *Header, keyword string, limit int) ([]dto.UserInfo, error) {
	// 构建查询参数
	path := "/hr/api/v1/user/search_fuzzy"
	params := url.Values{}
	params.Set("keyword", keyword)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	// 构建完整 URL
	gatewayURL := serviceconfig.GetGatewayURL()
	fullURL := fmt.Sprintf("%s%s?%s", gatewayURL, path, params.Encode())

	result, err := callAPIWithURL[dto.SearchUsersFuzzyResp](
		http.MethodGet,
		fullURL,
		header,
		nil,
	)
	if err != nil {
		return nil, err
	}
	return result.Data.Users, nil
}
