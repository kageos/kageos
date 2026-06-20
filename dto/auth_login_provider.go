package dto

type AuthLoginProviderField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Secret      bool   `json:"secret"`
	Help        string `json:"help,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Value       string `json:"value,omitempty"`
	ValueSet    bool   `json:"value_set"`
}

type AuthLoginProviderResp struct {
	Code         string                   `json:"code"`
	Name         string                   `json:"name"`
	Description  string                   `json:"description"`
	Action       string                   `json:"action"`
	Enabled      bool                     `json:"enabled"`
	Configured   bool                     `json:"configured"`
	Status       string                   `json:"status"`
	CallbackPath string                   `json:"callback_path,omitempty"`
	DocsURL      string                   `json:"docs_url,omitempty"`
	Fields       []AuthLoginProviderField `json:"fields"`
	UpdatedBy    string                   `json:"updated_by,omitempty"`
	UpdatedAt    string                   `json:"updated_at,omitempty"`
}

type ListAuthLoginProvidersResp struct {
	Providers []*AuthLoginProviderResp `json:"providers"`
}

type UpdateAuthLoginProviderConfigReq struct {
	Config map[string]string `json:"config"`
}

type UpdateAuthLoginProviderEnabledReq struct {
	Enabled bool `json:"enabled"`
}

type LoginMethodResp struct {
	Provider      string `json:"provider"`
	Label         string `json:"label"`
	Action        string `json:"action"`
	Description   string `json:"description,omitempty"`
	AuthorizePath string `json:"authorize_path,omitempty"`
}

type ListLoginMethodsResp struct {
	Methods []LoginMethodResp `json:"methods"`
}
