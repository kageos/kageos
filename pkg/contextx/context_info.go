package contextx

import (
	"net/http"
	"strings"
)

// TraceIdHeader HTTP Header 中的 TraceId key（统一使用此名称）
const TraceIdHeader = "X-Trace-Id"

// RequestUserHeader HTTP Header 中的 RequestUser key（统一使用此名称）
const RequestUserHeader = "X-Request-User"

// UsernameHeader is a legacy identity header accepted by some downstream
// middleware. Gateway boundaries must treat it as trusted identity metadata,
// not as a client-controlled fallback.
const UsernameHeader = "X-Username"

// DepartmentFullPathHeader HTTP Header 和 Context 中的 DepartmentFullPath key（统一使用此名称）
// ⭐ 统一使用此常量，不要硬编码字符串（既用于 HTTP Header，也用于 Context）
const DepartmentFullPathHeader = "X-Department-Full-Path"

const CompanyCodeHeader = "X-Company-Code"
const CompanyNameHeader = "X-Company-Name"
const CompanyLogoURLHeader = "X-Company-Logo-Url"
const UserIDHeader = "X-User-Id"
const UserEmailHeader = "X-User-Email"
const LeaderUsernameHeader = "X-Leader-Username"

// TokenHeader HTTP Header 中的 Token key（统一使用此名称）
const TokenHeader = "X-Token"

// ClientSourceHeader HTTP Header 中的客户端来源 key（统一使用此名称）
const ClientSourceHeader = "X-Client-Source"

const (
	ClientSourceBrowser       = "browser"
	ClientSourceAgent         = "agent"
	ClientSourceOpenAPI       = "openapi"
	ClientSourcePublicShare   = "public_share"
	ClientSourceScheduledTask = "scheduled_task"
	ClientSourceUnknown       = "unknown"
)

// SourceTypeHeader / SourceRefHeader 标记后台自动化、函数触发等调用来源。
// 定时 Agent 会话先埋 ref，后续工具白名单可基于这些字段在工具入口统一控制。
const SourceTypeHeader = "X-Source-Type"
const SourceRefHeader = "X-Source-Ref"
const SourcePathHeader = "X-Source-Path"
const SourceTitleHeader = "X-Source-Title"
const SourceParentPathHeader = "X-Source-Parent-Path"
const SourceParentTitleHeader = "X-Source-Parent-Title"
const SourceTemplateTypeHeader = "X-Source-Template-Type"

const WorkspaceSessionIDHeader = "X-Workspace-Session-Id"
const WorkspaceSessionTitleHeader = "X-Workspace-Session-Title"
const WorkspaceRoleHeader = "X-Workspace-Role"
const InitiatorUserHeader = "X-Initiator-User"
const WorkspaceMessageIDHeader = "X-Workspace-Message-Id"
const ToolCallIDHeader = "X-Tool-Call-Id"
const ToolNameHeader = "X-Tool-Name"

const (
	SourceTypeOpenAPIToken  = "openapi_token"
	SourceTypePublicShare   = "public_share"
	SourceTypeAgentTool     = "agent_tool"
	SourceTypeScheduledTask = "scheduled_task"
)

const PubKeyHerder = "X-Pub-Key"

// trustedIdentityHeaderNames is the complete set of HTTP headers whose values
// affect authorization identity or audit provenance. Requests crossing an
// external trust boundary must clear these values before verified credentials
// (or a verified internal request signature) are allowed to rebuild them.
//
// Keep this list centralized: adding a new identity/provenance header without
// adding it here would reintroduce a header-spoofing path at the gateway.
var trustedIdentityHeaderNames = [...]string{
	RequestUserHeader,
	UsernameHeader,
	DepartmentFullPathHeader,
	CompanyCodeHeader,
	CompanyNameHeader,
	CompanyLogoURLHeader,
	UserIDHeader,
	UserEmailHeader,
	LeaderUsernameHeader,
	ClientSourceHeader,
	SourceTypeHeader,
	SourceRefHeader,
	SourcePathHeader,
	SourceTitleHeader,
	SourceParentPathHeader,
	SourceParentTitleHeader,
	SourceTemplateTypeHeader,
	WorkspaceSessionIDHeader,
	WorkspaceSessionTitleHeader,
	WorkspaceRoleHeader,
	InitiatorUserHeader,
	WorkspaceMessageIDHeader,
	ToolCallIDHeader,
	ToolNameHeader,
}

var trustedIdentityHeaderNameSet = func() map[string]struct{} {
	set := make(map[string]struct{}, len(trustedIdentityHeaderNames))
	for _, name := range trustedIdentityHeaderNames {
		set[strings.ToLower(name)] = struct{}{}
	}
	return set
}()

// TrustedIdentityHeaderNames returns a copy of all trusted identity and audit
// provenance header names. Callers may sort or modify the returned slice.
func TrustedIdentityHeaderNames() []string {
	names := make([]string, len(trustedIdentityHeaderNames))
	copy(names, trustedIdentityHeaderNames[:])
	return names
}

// CaptureTrustedIdentityHeaders takes a canonical, single-value snapshot of
// trusted identity headers. It is intended for verified internal-call signing
// and gateway verification; it must never itself be treated as proof of trust.
func CaptureTrustedIdentityHeaders(header http.Header) map[string]string {
	values := make(map[string]string, len(trustedIdentityHeaderNames))
	for _, name := range trustedIdentityHeaderNames {
		if value := strings.TrimSpace(header.Get(name)); value != "" {
			values[name] = value
		}
	}
	return values
}

// ClearTrustedIdentityHeaders removes all client-supplied identity and audit
// provenance before a request crosses into trusted backend services.
func ClearTrustedIdentityHeaders(header http.Header) {
	for name := range header {
		if _, trusted := trustedIdentityHeaderNameSet[strings.ToLower(name)]; trusted {
			delete(header, name)
		}
	}
}

// ApplyTrustedIdentityHeaders replaces trusted identity metadata with a
// previously verified snapshot. Existing values are always removed first.
func ApplyTrustedIdentityHeaders(header http.Header, values map[string]string) {
	ClearTrustedIdentityHeaders(header)
	for _, name := range trustedIdentityHeaderNames {
		if value := strings.TrimSpace(values[name]); value != "" {
			header.Set(name, value)
		}
	}
}
