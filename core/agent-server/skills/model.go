package skills

type SkillMeta struct {
	ID               string   `json:"id" yaml:"id"`
	Name             string   `json:"name" yaml:"name"`
	Description      string   `json:"description" yaml:"description"`
	Triggers         []string `json:"triggers,omitempty" yaml:"triggers"`
	Modes            []string `json:"modes,omitempty" yaml:"modes"`
	RequiredDocs     []string `json:"required_docs,omitempty" yaml:"required_docs"`
	RecommendedDemos []string `json:"recommended_demos,omitempty" yaml:"recommended_demos"`
	Capabilities     []string `json:"capabilities,omitempty" yaml:"capabilities"`
	AllowedTools     []string `json:"allowed_tools,omitempty" yaml:"allowed_tools"`
	Completion       []string `json:"completion,omitempty" yaml:"completion"`
	Path             string   `json:"path,omitempty" yaml:"-"`
}

type Skill struct {
	Meta SkillMeta `json:"meta"`
	Body string    `json:"body"`
}

type SearchOptions struct {
	Keyword string
	Mode    string
	Limit   int
}

type SearchResult struct {
	Meta  SkillMeta `json:"meta"`
	Score int       `json:"score"`
}
