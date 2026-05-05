package skills

import "embed"

// skillFS embeds the built-in AgentOS skills. These files are local seed
// assets, not service-tree docs; runtime can layer editable skills later.
//
//go:embed sop/*/SKILL.md sdk/*/SKILL.md system/*/SKILL.md system/*/*/SKILL.md
var skillFS embed.FS
