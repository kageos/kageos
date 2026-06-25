package license

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Edition string

const (
	EditionCommunity  Edition = "community"
	EditionEnterprise Edition = "enterprise"
)

type Status string

const (
	StatusNone    Status = "none"
	StatusActive  Status = "active"
	StatusExpired Status = "expired"
)

type EffectiveMode string

const (
	EffectiveModeCommunity  EffectiveMode = "community"
	EffectiveModeEnterprise EffectiveMode = "enterprise"
)

type Tier string

const (
	TierCommunity  Tier = "community"
	TierEnterprise Tier = "enterprise"
)

type CapabilityName string

const (
	CapabilityAuditLog           CapabilityName = "audit_log"
	CapabilityApprovalWorkflow   CapabilityName = "approval_workflow"
	CapabilityPermissionRequest  CapabilityName = "permission_request"
	CapabilitySAML               CapabilityName = "saml"
	CapabilitySlackNotification  CapabilityName = "slack_notification"
	CapabilityEnterpriseWorkflow CapabilityName = "enterprise_workflow"
)

var ErrCapabilityDisabled = errors.New("capability disabled")

type Capability struct {
	Name        CapabilityName `json:"name"`
	Tier        Tier           `json:"tier"`
	DisplayName string         `json:"display_name,omitempty"`
	Description string         `json:"description,omitempty"`
}

type CapabilityInfo struct {
	Name        CapabilityName `json:"name"`
	Tier        Tier           `json:"tier"`
	DisplayName string         `json:"display_name,omitempty"`
	Description string         `json:"description,omitempty"`
	Enabled     bool           `json:"enabled"`
}

type Info struct {
	Service              string           `json:"service,omitempty"`
	Edition              Edition          `json:"edition"`
	LicenseStatus        Status           `json:"license_status"`
	EffectiveMode        EffectiveMode    `json:"effective_mode"`
	Capabilities         []CapabilityInfo `json:"capabilities"`
	EnabledCapabilities  []CapabilityName `json:"enabled_capabilities"`
	DisabledCapabilities []CapabilityName `json:"disabled_capabilities,omitempty"`
}

var state = struct {
	sync.RWMutex
	edition               Edition
	status                Status
	enabledCaps           map[CapabilityName]struct{}
	capabilityOrder       []CapabilityName
	capabilityDefinitions map[CapabilityName]Capability
}{
	edition:               EditionCommunity,
	status:                StatusNone,
	enabledCaps:           make(map[CapabilityName]struct{}),
	capabilityDefinitions: make(map[CapabilityName]Capability),
}

func SetEdition(edition Edition) {
	edition = normalizeEdition(edition)
	if !isValidEdition(edition) {
		panic(fmt.Sprintf("invalid edition: %s", edition))
	}
	state.Lock()
	defer state.Unlock()
	state.edition = edition
}

func EditionValue() Edition {
	state.RLock()
	defer state.RUnlock()
	return state.edition
}

func SetStatus(status Status, capabilities ...CapabilityName) {
	status = normalizeStatus(status)
	if !isValidStatus(status) {
		panic(fmt.Sprintf("invalid license status: %s", status))
	}
	normalizedCaps := make(map[CapabilityName]struct{}, len(capabilities))
	for _, capability := range capabilities {
		name := normalizeCapabilityName(capability)
		if name != "" {
			normalizedCaps[name] = struct{}{}
		}
	}

	state.Lock()
	defer state.Unlock()
	state.status = status
	state.enabledCaps = normalizedCaps
}

func Activate(capabilities ...CapabilityName) {
	SetStatus(StatusActive, capabilities...)
}

func Clear() {
	SetStatus(StatusNone)
}

func Expire() {
	SetStatus(StatusExpired)
}

func RegisterCapability(capability Capability) {
	capability.Name = normalizeCapabilityName(capability.Name)
	capability.Tier = normalizeTier(capability.Tier)
	if capability.Name == "" {
		panic("capability name is required")
	}
	if !isValidTier(capability.Tier) {
		panic(fmt.Sprintf("invalid capability tier: %s", capability.Tier))
	}

	state.Lock()
	defer state.Unlock()
	if _, exists := state.capabilityDefinitions[capability.Name]; exists {
		panic(fmt.Sprintf("capability %s already registered", capability.Name))
	}
	state.capabilityOrder = append(state.capabilityOrder, capability.Name)
	state.capabilityDefinitions[capability.Name] = capability
}

func RegisteredCapabilities() []Capability {
	state.RLock()
	defer state.RUnlock()
	out := make([]Capability, 0, len(state.capabilityOrder))
	for _, name := range state.capabilityOrder {
		out = append(out, state.capabilityDefinitions[name])
	}
	return out
}

func Enabled(name CapabilityName) bool {
	name = normalizeCapabilityName(name)
	if name == "" {
		return false
	}
	state.RLock()
	defer state.RUnlock()
	return capabilityEnabledLocked(name)
}

func Require(name CapabilityName) error {
	name = normalizeCapabilityName(name)
	if Enabled(name) {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrCapabilityDisabled, name)
}

func Snapshot(service ...string) Info {
	state.RLock()
	defer state.RUnlock()

	info := Info{
		Edition:       state.edition,
		LicenseStatus: state.status,
		EffectiveMode: effectiveModeLocked(),
		Capabilities:  make([]CapabilityInfo, 0, len(state.capabilityOrder)),
	}
	if len(service) > 0 {
		info.Service = strings.TrimSpace(service[0])
	}

	enabled := make([]CapabilityName, 0)
	disabled := make([]CapabilityName, 0)
	for _, name := range state.capabilityOrder {
		definition := state.capabilityDefinitions[name]
		isEnabled := capabilityEnabledLocked(name)
		info.Capabilities = append(info.Capabilities, CapabilityInfo{
			Name:        definition.Name,
			Tier:        definition.Tier,
			DisplayName: definition.DisplayName,
			Description: definition.Description,
			Enabled:     isEnabled,
		})
		if isEnabled {
			enabled = append(enabled, name)
		} else {
			disabled = append(disabled, name)
		}
	}
	info.EnabledCapabilities = enabled
	info.DisabledCapabilities = disabled
	return info
}

func Summary(service ...string) string {
	info := Snapshot(service...)
	enabled := make([]string, 0, len(info.EnabledCapabilities))
	for _, name := range info.EnabledCapabilities {
		enabled = append(enabled, string(name))
	}
	sort.Strings(enabled)
	summary := fmt.Sprintf(
		"edition=%s license_status=%s effective_mode=%s enabled_capabilities=%v",
		info.Edition,
		info.LicenseStatus,
		info.EffectiveMode,
		enabled,
	)
	if info.Service != "" {
		summary = fmt.Sprintf("service=%s %s", info.Service, summary)
	}
	return summary
}

func capabilityEnabledLocked(name CapabilityName) bool {
	definition, exists := state.capabilityDefinitions[name]
	if !exists {
		return false
	}
	if definition.Tier == TierCommunity {
		return true
	}
	if state.edition != EditionEnterprise || state.status != StatusActive {
		return false
	}
	_, activated := state.enabledCaps[name]
	return activated
}

func effectiveModeLocked() EffectiveMode {
	if state.edition == EditionEnterprise && state.status == StatusActive {
		return EffectiveModeEnterprise
	}
	return EffectiveModeCommunity
}

func normalizeEdition(edition Edition) Edition {
	return Edition(strings.ToLower(strings.TrimSpace(string(edition))))
}

func isValidEdition(edition Edition) bool {
	return edition == EditionCommunity || edition == EditionEnterprise
}

func normalizeStatus(status Status) Status {
	return Status(strings.ToLower(strings.TrimSpace(string(status))))
}

func isValidStatus(status Status) bool {
	return status == StatusNone || status == StatusActive || status == StatusExpired
}

func normalizeTier(tier Tier) Tier {
	if tier == "" {
		return TierEnterprise
	}
	return Tier(strings.ToLower(strings.TrimSpace(string(tier))))
}

func isValidTier(tier Tier) bool {
	return tier == TierCommunity || tier == TierEnterprise
}

func normalizeCapabilityName(name CapabilityName) CapabilityName {
	return CapabilityName(strings.ToLower(strings.TrimSpace(string(name))))
}
