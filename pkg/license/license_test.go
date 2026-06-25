package license

import (
	"errors"
	"testing"
)

func TestDefaultSnapshotUsesCommunityMode(t *testing.T) {
	resetStateForTest(t)

	info := Snapshot("app-server")

	if info.Service != "app-server" {
		t.Fatalf("service = %q, want app-server", info.Service)
	}
	if info.Edition != EditionCommunity {
		t.Fatalf("edition = %q, want %q", info.Edition, EditionCommunity)
	}
	if info.LicenseStatus != StatusNone {
		t.Fatalf("license status = %q, want %q", info.LicenseStatus, StatusNone)
	}
	if info.EffectiveMode != EffectiveModeCommunity {
		t.Fatalf("effective mode = %q, want %q", info.EffectiveMode, EffectiveModeCommunity)
	}
	if len(info.Capabilities) != 0 {
		t.Fatalf("capabilities = %v, want empty", info.Capabilities)
	}
}

func TestCommunityCapabilityIsEnabledWithoutLicense(t *testing.T) {
	resetStateForTest(t)

	RegisterCapability(Capability{Name: "base_activity", Tier: TierCommunity})

	if !Enabled("base_activity") {
		t.Fatal("community capability should be enabled")
	}
	info := Snapshot()
	if len(info.EnabledCapabilities) != 1 || info.EnabledCapabilities[0] != "base_activity" {
		t.Fatalf("enabled capabilities = %v, want [base_activity]", info.EnabledCapabilities)
	}
}

func TestEnterpriseCapabilityRequiresActiveEnterpriseLicense(t *testing.T) {
	resetStateForTest(t)

	RegisterCapability(Capability{Name: CapabilityAuditLog, Tier: TierEnterprise})

	if Enabled(CapabilityAuditLog) {
		t.Fatal("enterprise capability should not be enabled in community edition")
	}

	SetEdition(EditionEnterprise)
	if Enabled(CapabilityAuditLog) {
		t.Fatal("enterprise capability should not be enabled without active license")
	}

	Activate(CapabilityAuditLog)
	if !Enabled(CapabilityAuditLog) {
		t.Fatal("enterprise capability should be enabled after active enterprise license")
	}

	info := Snapshot()
	if info.EffectiveMode != EffectiveModeEnterprise {
		t.Fatalf("effective mode = %q, want %q", info.EffectiveMode, EffectiveModeEnterprise)
	}
}

func TestRequireReturnsCapabilityDisabledError(t *testing.T) {
	resetStateForTest(t)

	RegisterCapability(Capability{Name: CapabilityApprovalWorkflow, Tier: TierEnterprise})

	if err := Require(CapabilityApprovalWorkflow); !errors.Is(err, ErrCapabilityDisabled) {
		t.Fatalf("Require error = %v, want ErrCapabilityDisabled", err)
	}
}

func TestRegisterCapabilityRejectsDuplicate(t *testing.T) {
	resetStateForTest(t)

	RegisterCapability(Capability{Name: "audit_log", Tier: TierEnterprise})

	defer func() {
		if recover() == nil {
			t.Fatal("RegisterCapability duplicate did not panic")
		}
	}()
	RegisterCapability(Capability{Name: "audit_log", Tier: TierEnterprise})
}

func resetStateForTest(t *testing.T) {
	t.Helper()
	state.Lock()
	state.edition = EditionCommunity
	state.status = StatusNone
	state.enabledCaps = make(map[CapabilityName]struct{})
	state.capabilityOrder = nil
	state.capabilityDefinitions = make(map[CapabilityName]Capability)
	state.Unlock()
	t.Cleanup(func() {
		state.Lock()
		state.edition = EditionCommunity
		state.status = StatusNone
		state.enabledCaps = make(map[CapabilityName]struct{})
		state.capabilityOrder = nil
		state.capabilityDefinitions = make(map[CapabilityName]Capability)
		state.Unlock()
	})
}
