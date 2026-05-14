package permission

// EnforcementEnabled is the global switch for RBAC enforcement.
//
// The product currently runs with permission control disabled so MVP flows are
// not blocked by role setup. Keep the checks wired through this function so
// re-enabling RBAC later is an explicit product decision.
func EnforcementEnabled() bool {
	return false
}
