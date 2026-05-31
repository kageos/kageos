package dto

type ConnectorEndpoint struct {
	Provider       string   `json:"provider,omitempty"`
	Method         string   `json:"method,omitempty"`
	URL            string   `json:"url,omitempty"`
	Name           string   `json:"name,omitempty"`
	Desc           string   `json:"desc,omitempty"`
	RequiredScopes []string `json:"required_scopes,omitempty"`
}
