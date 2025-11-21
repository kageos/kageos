package dto

// QueryUserReq 查询用户请求
type QueryUserReq struct {
	Username string `json:"username" form:"username" binding:"required" example:"beiluo"` // 用户名
}

// QueryUserResp 查询用户响应
type QueryUserResp struct {
	User UserInfo `json:"user"` // 用户信息
}

// SearchUsersFuzzyReq 模糊查询用户请求
type SearchUsersFuzzyReq struct {
	Keyword string `json:"keyword" form:"keyword" binding:"required" example:"bei"` // 搜索关键词
	Limit   int    `json:"limit" form:"limit" example:"10"`                        // 返回数量限制，默认10，最大100
}

// SearchUsersFuzzyResp 模糊查询用户响应
type SearchUsersFuzzyResp struct {
	Users []UserInfo `json:"users"` // 用户列表
}

// GetUsersByUsernamesReq 批量获取用户请求
type GetUsersByUsernamesReq struct {
	Usernames []string `json:"usernames" binding:"required,min=1,max=100" example:"[\"user1\",\"user2\"]"` // 用户名列表，最多100个
}

// GetUsersByUsernamesResp 批量获取用户响应
type GetUsersByUsernamesResp struct {
	Users []UserInfo `json:"users"` // 用户列表
}

// UpdateUserReq 更新用户信息请求（所有字段都是可选的）
type UpdateUserReq struct {
	Nickname  *string `json:"nickname,omitempty" example:"北落"`                           // 昵称（可选，传值则更新，不传则不更新）
	Signature *string `json:"signature,omitempty" example:"这个人很懒，什么都没有留下"`         // 个人签名/简介（可选，传值则更新，不传则不更新）
	Avatar    *string `json:"avatar,omitempty" example:"https://avatar.com/1.jpg"`        // 头像URL（可选，传值则更新，不传则不更新）
	Gender    *string `json:"gender,omitempty" example:"male"`                             // 性别（可选，传值则更新，不传则不更新）: male(男), female(女), other(其他)
}

// UpdateUserResp 更新用户信息响应
type UpdateUserResp struct {
	User UserInfo `json:"user"` // 更新后的用户信息
}

