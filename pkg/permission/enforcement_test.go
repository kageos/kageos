package permission

import "testing"

func TestEnforcementEnabledIsDisabledByDefault(t *testing.T) {
	if EnforcementEnabled() {
		t.Fatal("permission enforcement should be disabled")
	}
}
