package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
	"github.com/kageos/kageos/dto"
)

const (
	ProviderGoogleOAuth = "google_oauth"
	ProviderGitHubOAuth = "github_oauth"

	ProviderActionRedirect = "redirect"

	ProviderStatusUnconfigured = "unconfigured"
	ProviderStatusDisabled     = "disabled"
	ProviderStatusEnabled      = "enabled"
)

type AuthLoginProviderService struct {
	repo *repository.AuthLoginProviderRepository
}

type AuthProviderFieldDef struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Secret      bool   `json:"secret"`
	Help        string `json:"help,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
}

type AuthProviderSeed struct {
	Code          string
	Name          string
	Description   string
	Action        string
	Fields        []AuthProviderFieldDef
	AuthorizePath string
	CallbackPath  string
	DocsURL       string
	SortOrder     int
}

type AuthLoginProviderRuntimeConfig struct {
	Code   string
	Action string
	Values map[string]string
}

var authProviderSeedRegistry = struct {
	sync.RWMutex
	seeds map[string]AuthProviderSeed
}{
	seeds: make(map[string]AuthProviderSeed),
}

func RegisterAuthProviderSeed(seed AuthProviderSeed) {
	seed.Code = normalizeProviderCode(seed.Code)
	if seed.Code == "" {
		panic("auth provider seed code is empty")
	}
	if strings.TrimSpace(seed.Name) == "" {
		panic(fmt.Sprintf("auth provider seed %s name is empty", seed.Code))
	}
	if strings.TrimSpace(seed.Action) == "" {
		panic(fmt.Sprintf("auth provider seed %s action is empty", seed.Code))
	}
	authProviderSeedRegistry.Lock()
	defer authProviderSeedRegistry.Unlock()
	if _, exists := authProviderSeedRegistry.seeds[seed.Code]; exists {
		panic(fmt.Sprintf("auth provider seed %s already registered", seed.Code))
	}
	authProviderSeedRegistry.seeds[seed.Code] = seed
}

func LookupAuthProviderSeed(code string) (AuthProviderSeed, bool) {
	code = normalizeProviderCode(code)
	authProviderSeedRegistry.RLock()
	defer authProviderSeedRegistry.RUnlock()
	seed, ok := authProviderSeedRegistry.seeds[code]
	return seed, ok
}

func RegisteredAuthProviderSeeds() []AuthProviderSeed {
	authProviderSeedRegistry.RLock()
	seeds := make([]AuthProviderSeed, 0, len(authProviderSeedRegistry.seeds))
	for _, seed := range authProviderSeedRegistry.seeds {
		seeds = append(seeds, seed)
	}
	authProviderSeedRegistry.RUnlock()
	sort.Slice(seeds, func(i, j int) bool {
		if seeds[i].SortOrder == seeds[j].SortOrder {
			return seeds[i].Code < seeds[j].Code
		}
		return seeds[i].SortOrder < seeds[j].SortOrder
	})
	return seeds
}

func NewAuthLoginProviderService(repo *repository.AuthLoginProviderRepository) *AuthLoginProviderService {
	return &AuthLoginProviderService{repo: repo}
}

func (s *AuthLoginProviderService) SeedDefaults(ctx context.Context) error {
	for _, seed := range RegisteredAuthProviderSeeds() {
		schema, err := json.Marshal(seed.Fields)
		if err != nil {
			return err
		}
		if err := s.repo.UpsertSeed(ctx, &model.AuthLoginProvider{
			Code:             seed.Code,
			Name:             seed.Name,
			Description:      seed.Description,
			Action:           seed.Action,
			Enabled:          false,
			Configured:       false,
			Status:           ProviderStatusUnconfigured,
			ConfigSchemaJSON: string(schema),
			ConfigValuesJSON: "{}",
			CallbackPath:     seed.CallbackPath,
			DocsURL:          seed.DocsURL,
			SortOrder:        seed.SortOrder,
			UpdatedBy:        "system",
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *AuthLoginProviderService) ListProviders() ([]*dto.AuthLoginProviderResp, error) {
	providers, err := s.repo.List(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]*dto.AuthLoginProviderResp, 0, len(providers))
	for _, provider := range providers {
		resp, err := s.toProviderResp(provider)
		if err != nil {
			return nil, err
		}
		out = append(out, resp)
	}
	return out, nil
}

func (s *AuthLoginProviderService) ListLoginMethods() ([]dto.LoginMethodResp, error) {
	providers, err := s.ListProviders()
	if err != nil {
		return nil, err
	}
	methods := make([]dto.LoginMethodResp, 0, len(providers))
	for _, provider := range providers {
		if provider.Enabled && provider.Configured && provider.Status == ProviderStatusEnabled {
			methods = append(methods, dto.LoginMethodResp{
				Provider:      provider.Code,
				Label:         provider.Name,
				Action:        provider.Action,
				Description:   provider.Description,
				AuthorizePath: providerAuthorizePath(provider.Code),
			})
		}
	}
	return methods, nil
}

func (s *AuthLoginProviderService) GetEnabledRuntimeConfig(code string) (*AuthLoginProviderRuntimeConfig, error) {
	provider, err := s.repo.GetByCode(context.Background(), normalizeProviderCode(code))
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, fmt.Errorf("login provider %q not found", code)
	}
	fields, err := parseProviderFields(provider.ConfigSchemaJSON)
	if err != nil {
		return nil, err
	}
	values := parseProviderValues(provider.ConfigValuesJSON)
	if !provider.Enabled || !providerConfigured(fields, values) {
		return nil, fmt.Errorf("login provider %q is not enabled", code)
	}
	return &AuthLoginProviderRuntimeConfig{
		Code:   provider.Code,
		Action: provider.Action,
		Values: values,
	}, nil
}

func (s *AuthLoginProviderService) UpdateConfig(code string, config map[string]string, updatedBy string) (*dto.AuthLoginProviderResp, error) {
	provider, err := s.repo.GetByCode(context.Background(), normalizeProviderCode(code))
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, fmt.Errorf("login provider %q not found", code)
	}

	fields, err := parseProviderFields(provider.ConfigSchemaJSON)
	if err != nil {
		return nil, err
	}
	values := parseProviderValues(provider.ConfigValuesJSON)
	for _, field := range fields {
		raw, exists := config[field.Key]
		if !exists {
			continue
		}
		value := strings.TrimSpace(raw)
		if field.Secret && value == "" {
			continue
		}
		if value == "" {
			delete(values, field.Key)
			continue
		}
		values[field.Key] = value
	}

	configured := providerConfigured(fields, values)
	provider.Configured = configured
	if !configured {
		provider.Enabled = false
		provider.Status = ProviderStatusUnconfigured
	} else if provider.Enabled {
		provider.Status = ProviderStatusEnabled
	} else {
		provider.Status = ProviderStatusDisabled
	}
	provider.UpdatedBy = strings.TrimSpace(updatedBy)
	valuesJSON, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	provider.ConfigValuesJSON = string(valuesJSON)
	if err := s.repo.UpdateConfig(context.Background(), provider); err != nil {
		return nil, err
	}
	updated, err := s.repo.GetByCode(context.Background(), provider.Code)
	if err != nil {
		return nil, err
	}
	return s.toProviderResp(updated)
}

func (s *AuthLoginProviderService) SetEnabled(code string, enabled bool, updatedBy string) (*dto.AuthLoginProviderResp, error) {
	provider, err := s.repo.GetByCode(context.Background(), normalizeProviderCode(code))
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, fmt.Errorf("login provider %q not found", code)
	}
	fields, err := parseProviderFields(provider.ConfigSchemaJSON)
	if err != nil {
		return nil, err
	}
	values := parseProviderValues(provider.ConfigValuesJSON)
	configured := providerConfigured(fields, values)
	if enabled && !configured {
		return nil, fmt.Errorf("login provider %q is not configured", code)
	}
	status := ProviderStatusDisabled
	if !configured {
		status = ProviderStatusUnconfigured
		enabled = false
	} else if enabled {
		status = ProviderStatusEnabled
	}
	if err := s.repo.UpdateEnabled(context.Background(), provider.Code, enabled, status, strings.TrimSpace(updatedBy)); err != nil {
		return nil, err
	}
	updated, err := s.repo.GetByCode(context.Background(), provider.Code)
	if err != nil {
		return nil, err
	}
	return s.toProviderResp(updated)
}

func (s *AuthLoginProviderService) toProviderResp(provider *model.AuthLoginProvider) (*dto.AuthLoginProviderResp, error) {
	fields, err := parseProviderFields(provider.ConfigSchemaJSON)
	if err != nil {
		return nil, err
	}
	values := parseProviderValues(provider.ConfigValuesJSON)
	configured := providerConfigured(fields, values)
	status := provider.Status
	if !configured {
		status = ProviderStatusUnconfigured
	} else if provider.Enabled {
		status = ProviderStatusEnabled
	} else if status == "" || status == ProviderStatusUnconfigured {
		status = ProviderStatusDisabled
	}

	fieldResp := make([]dto.AuthLoginProviderField, 0, len(fields))
	for _, field := range fields {
		value := strings.TrimSpace(values[field.Key])
		item := dto.AuthLoginProviderField{
			Key:         field.Key,
			Label:       field.Label,
			Type:        field.Type,
			Required:    field.Required,
			Secret:      field.Secret,
			Help:        field.Help,
			Placeholder: field.Placeholder,
			ValueSet:    value != "",
		}
		if !field.Secret {
			item.Value = value
		}
		fieldResp = append(fieldResp, item)
	}

	return &dto.AuthLoginProviderResp{
		Code:         provider.Code,
		Name:         provider.Name,
		Description:  provider.Description,
		Action:       provider.Action,
		Enabled:      provider.Enabled && configured,
		Configured:   configured,
		Status:       status,
		CallbackPath: provider.CallbackPath,
		DocsURL:      provider.DocsURL,
		Fields:       fieldResp,
		UpdatedBy:    provider.UpdatedBy,
		UpdatedAt:    time.Time(provider.UpdatedAt).Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func parseProviderFields(raw string) ([]AuthProviderFieldDef, error) {
	var fields []AuthProviderFieldDef
	if strings.TrimSpace(raw) == "" {
		return fields, nil
	}
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func parseProviderValues(raw string) map[string]string {
	values := map[string]string{}
	if strings.TrimSpace(raw) == "" {
		return values
	}
	_ = json.Unmarshal([]byte(raw), &values)
	return values
}

func providerConfigured(fields []AuthProviderFieldDef, values map[string]string) bool {
	for _, field := range fields {
		if field.Required && strings.TrimSpace(values[field.Key]) == "" {
			return false
		}
	}
	return true
}

func normalizeProviderCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

func providerAuthorizePath(code string) string {
	if seed, ok := LookupAuthProviderSeed(code); ok {
		return strings.TrimSpace(seed.AuthorizePath)
	}
	return ""
}
