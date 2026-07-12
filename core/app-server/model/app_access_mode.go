package model

import "strings"

// AppAccessMode describes how authenticated users enter and use a workspace.
// Permissioned keeps the existing per-user/team authorization behavior.
// OpenCollaboration opens business data operations to every authenticated user,
// while control-plane operations continue to use the existing authorization.
type AppAccessMode string

const (
	AppAccessModePermissioned      AppAccessMode = "permissioned"
	AppAccessModeOpenCollaboration AppAccessMode = "open_collaboration"
)

func NormalizeAppAccessMode(mode AppAccessMode) AppAccessMode {
	normalized := AppAccessMode(strings.ToLower(strings.TrimSpace(string(mode))))
	if normalized == "" {
		return AppAccessModePermissioned
	}
	return normalized
}

func IsValidAppAccessMode(mode AppAccessMode) bool {
	switch NormalizeAppAccessMode(mode) {
	case AppAccessModePermissioned, AppAccessModeOpenCollaboration:
		return true
	default:
		return false
	}
}

func (m AppAccessMode) IsOpenCollaboration() bool {
	return NormalizeAppAccessMode(m) == AppAccessModeOpenCollaboration
}
