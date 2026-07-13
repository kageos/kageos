package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/connector-server/model"
	"github.com/kageos/kageos/core/connector-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/gormx/models"
	"gorm.io/gorm"
)

const (
	connectorGlobalResourcePath = "/"
	connectorGlobalScopePart    = "*"
)

type ConnectorService struct {
	repo         *repository.ConnectorRepository
	oauth        *OAuthProviderRegistry
	tokenVault   *TokenVault
	callbackBase string
	stateTTL     time.Duration
	admins       map[string]struct{}
	oauthInitErr error
}

type ConnectorServiceOption func(*ConnectorService)

func NewConnectorService(repo *repository.ConnectorRepository, opts ...ConnectorServiceOption) *ConnectorService {
	svc := &ConnectorService{repo: repo}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

func (s *ConnectorService) CreateConnection(ctx context.Context, req dto.CreateConnectorConnectionReq) (*dto.ConnectorConnectionInfo, error) {
	owner := contextx.GetRequestUser(ctx)
	if owner == "" {
		return nil, fmt.Errorf("未提供用户信息")
	}
	return s.createConnectionForOwner(ctx, owner, req)
}

func (s *ConnectorService) createConnectionForOwner(ctx context.Context, owner string, req dto.CreateConnectorConnectionReq) (*dto.ConnectorConnectionInfo, error) {
	provider := normalizeProvider(req.Provider)
	if provider == "" {
		return nil, fmt.Errorf("provider 不能为空")
	}
	authType, err := normalizeConnectorAuthType(req.AuthType)
	if err != nil {
		return nil, err
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = provider
	}
	metadata, err := marshalMetadata(req.Metadata)
	if err != nil {
		return nil, err
	}
	connectionID, err := newConnectionID()
	if err != nil {
		return nil, err
	}
	conn := &model.ConnectorConnection{
		ConnectionID:      connectionID,
		OwnerUsername:     owner,
		Provider:          provider,
		AuthType:          authType,
		DisplayName:       displayName,
		ExternalAccountID: strings.TrimSpace(req.ExternalAccountID),
		Status:            model.ConnectorStatusActive,
		Metadata:          metadata,
	}
	conn.CreatedBy = owner
	conn.UpdatedBy = owner
	if err := s.repo.CreateConnection(ctx, conn); err != nil {
		return nil, err
	}
	return connectionToInfo(conn), nil
}

func (s *ConnectorService) updateConnectionProfileForOwner(ctx context.Context, owner, connectionID, provider, displayName, externalAccountID string, metadata map[string]interface{}) (*dto.ConnectorConnectionInfo, error) {
	provider = normalizeProvider(provider)
	if provider == "" {
		return nil, fmt.Errorf("provider 不能为空")
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = provider
	}
	metadataText, err := marshalMetadata(metadata)
	if err != nil {
		return nil, err
	}
	conn, err := s.repo.UpdateOwnedConnectionProfile(ctx, owner, connectionID, displayName, externalAccountID, metadataText, owner)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("连接器不存在或不属于当前用户")
		}
		return nil, err
	}
	if normalizeProvider(conn.Provider) != provider {
		return nil, fmt.Errorf("连接器 provider 不匹配: got %s, want %s", conn.Provider, provider)
	}
	return connectionToInfo(conn), nil
}

func (s *ConnectorService) ListConnections(ctx context.Context, provider string) ([]dto.ConnectorConnectionInfo, error) {
	owner := contextx.GetRequestUser(ctx)
	if owner == "" {
		return nil, fmt.Errorf("未提供用户信息")
	}
	rows, err := s.repo.ListConnections(ctx, owner, normalizeProvider(provider))
	if err != nil {
		return nil, err
	}
	items := make([]dto.ConnectorConnectionInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, *connectionToInfo(row))
	}
	return items, nil
}

func (s *ConnectorService) DeleteConnection(ctx context.Context, connectionID string) error {
	owner := contextx.GetRequestUser(ctx)
	if owner == "" {
		return fmt.Errorf("未提供用户信息")
	}
	connectionID = strings.TrimSpace(connectionID)
	if connectionID == "" {
		return fmt.Errorf("connection_id 不能为空")
	}
	if err := s.repo.DeleteOwnedConnection(ctx, owner, connectionID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("连接器不存在或无权删除")
		}
		return err
	}
	return nil
}

func (s *ConnectorService) RevokeConnection(ctx context.Context, connectionID string) error {
	owner := contextx.GetRequestUser(ctx)
	if owner == "" {
		return fmt.Errorf("未提供用户信息")
	}
	connectionID = strings.TrimSpace(connectionID)
	if connectionID == "" {
		return fmt.Errorf("connection_id 不能为空")
	}
	if err := s.repo.RevokeOwnedConnection(ctx, owner, connectionID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("连接器不存在或无权撤销")
		}
		return err
	}
	return nil
}

func (s *ConnectorService) BindDirectory(ctx context.Context, req dto.BindConnectorDirectoryReq) (*dto.ConnectorDirectoryBindingInfo, error) {
	owner := contextx.GetRequestUser(ctx)
	if owner == "" {
		return nil, fmt.Errorf("未提供用户信息")
	}
	provider := normalizeProvider(req.Provider)
	if provider == "" {
		return nil, fmt.Errorf("provider 不能为空")
	}
	if hasConnectorWildcardResourcePath(req.ResourcePath) {
		return nil, fmt.Errorf("resource_path 不支持通配符，请使用 / 表示全局连接器")
	}
	resourcePath := normalizeConnectorResourcePath(req.ResourcePath)
	if resourcePath == "" {
		return nil, fmt.Errorf("resource_path 不能为空")
	}
	tenantUser, app, err := parseConnectorBindingScope(resourcePath)
	if err != nil {
		return nil, err
	}
	conn, err := s.repo.GetOwnedConnection(ctx, owner, req.ConnectionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("连接器不存在或不属于当前用户")
		}
		return nil, err
	}
	if conn.Provider != provider {
		return nil, fmt.Errorf("连接器 provider 不匹配: got %s, want %s", conn.Provider, provider)
	}
	binding := &model.ConnectorDirectoryBinding{
		OwnerUsername: owner,
		TenantUser:    tenantUser,
		App:           app,
		ResourcePath:  resourcePath,
		Provider:      provider,
		ConnectionID:  conn.ConnectionID,
	}
	binding.CreatedBy = owner
	binding.UpdatedBy = owner
	if err := s.repo.UpsertDirectoryBinding(ctx, binding); err != nil {
		return nil, err
	}
	if saved, err := s.repo.FindDirectoryBinding(ctx, owner, resourcePath, provider); err == nil {
		binding = saved
	}
	return bindingToInfo(binding, conn), nil
}

func (s *ConnectorService) ListDirectoryBindings(ctx context.Context, resourcePath, provider string) ([]dto.ConnectorDirectoryBindingInfo, error) {
	owner := contextx.GetRequestUser(ctx)
	if owner == "" {
		return nil, fmt.Errorf("未提供用户信息")
	}
	if hasConnectorWildcardResourcePath(resourcePath) {
		return nil, fmt.Errorf("resource_path 不支持通配符，请使用 / 表示全局连接器")
	}
	resourcePath = normalizeConnectorResourcePath(resourcePath)
	rows, err := s.repo.ListDirectoryBindings(ctx, owner, resourcePath, normalizeProvider(provider))
	if err != nil {
		return nil, err
	}
	connectionIDs := make([]string, 0, len(rows))
	seenConnectionIDs := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if row == nil || strings.TrimSpace(row.ConnectionID) == "" {
			continue
		}
		if _, exists := seenConnectionIDs[row.ConnectionID]; exists {
			continue
		}
		seenConnectionIDs[row.ConnectionID] = struct{}{}
		connectionIDs = append(connectionIDs, row.ConnectionID)
	}
	connections, err := s.repo.GetOwnedConnections(ctx, owner, connectionIDs)
	if err != nil {
		return nil, err
	}
	items := make([]dto.ConnectorDirectoryBindingInfo, 0, len(rows))
	for _, row := range rows {
		item := bindingToInfo(row, nil)
		if conn := connections[row.ConnectionID]; conn != nil {
			item.Connection = connectionToInfo(conn)
		}
		items = append(items, *item)
	}
	return items, nil
}

func (s *ConnectorService) DeleteDirectoryBinding(ctx context.Context, resourcePath, provider string) error {
	owner := contextx.GetRequestUser(ctx)
	if owner == "" {
		return fmt.Errorf("未提供用户信息")
	}
	if hasConnectorWildcardResourcePath(resourcePath) {
		return fmt.Errorf("resource_path 不支持通配符，请使用 / 表示全局连接器")
	}
	resourcePath = normalizeConnectorResourcePath(resourcePath)
	provider = normalizeProvider(provider)
	if resourcePath == "" || provider == "" {
		return fmt.Errorf("resource_path 和 provider 不能为空")
	}
	rows, err := s.repo.DeleteDirectoryBinding(ctx, owner, resourcePath, provider)
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("绑定不存在或无权删除")
	}
	return nil
}

func (s *ConnectorService) ResolveDirectoryBinding(ctx context.Context, resourcePath, provider string) (*dto.ResolveConnectorBindingResp, error) {
	return s.ResolveDirectoryBindingWithScopes(ctx, resourcePath, provider, nil)
}

func (s *ConnectorService) ResolveDirectoryBindingWithScopes(ctx context.Context, resourcePath, provider string, requiredScopes []string) (*dto.ResolveConnectorBindingResp, error) {
	owner := contextx.GetRequestUser(ctx)
	if owner == "" {
		return nil, fmt.Errorf("未提供用户信息")
	}
	provider = normalizeProvider(provider)
	if provider == "" {
		return nil, fmt.Errorf("provider 不能为空")
	}
	if hasConnectorWildcardResourcePath(resourcePath) {
		return nil, fmt.Errorf("resource_path 不支持通配符，请使用 / 表示全局连接器")
	}
	for _, candidate := range resourcePathCandidates(resourcePath) {
		binding, err := s.repo.FindDirectoryBinding(ctx, owner, candidate, provider)
		if err == nil {
			conn, err := s.repo.GetOwnedConnection(ctx, owner, binding.ConnectionID)
			if err != nil {
				return nil, fmt.Errorf("绑定连接器不可用: %w", err)
			}
			tokenInfo, grantedScopes, err := s.oauthTokenInfoForConnection(ctx, owner, conn.ConnectionID)
			if err != nil {
				return nil, err
			}
			requiredScopes = cleanScopes(requiredScopes)
			missing := connectorAdapterFor(provider).MissingScopes(grantedScopes, requiredScopes)
			return &dto.ResolveConnectorBindingResp{
				Binding:        *bindingToInfo(binding, conn),
				Connection:     *connectionToInfo(conn),
				Token:          tokenInfo,
				ResolvedFrom:   binding.ResourcePath,
				RequestedPath:  defaultConnectorResourcePath(resourcePath),
				RequiredScopes: requiredScopes,
				GrantedScopes:  grantedScopes,
				MissingScopes:  missing,
				ScopeSatisfied: len(missing) == 0,
			}, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("当前用户没有可用于 %s 的 %s 连接器绑定", defaultConnectorResourcePath(resourcePath), provider)
}

func (s *ConnectorService) oauthTokenInfoForConnection(ctx context.Context, owner, connectionID string) (*dto.ConnectorTokenInfo, []string, error) {
	tokenRow, err := s.repo.GetOwnedOAuthToken(ctx, owner, connectionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	grantedScopes := splitScopes(tokenRow.Scopes)
	return tokenToInfo(tokenRow), grantedScopes, nil
}

func connectionToInfo(conn *model.ConnectorConnection) *dto.ConnectorConnectionInfo {
	if conn == nil {
		return nil
	}
	return &dto.ConnectorConnectionInfo{
		ID:                conn.ID,
		ConnectionID:      conn.ConnectionID,
		OwnerUsername:     conn.OwnerUsername,
		Provider:          conn.Provider,
		AuthType:          defaultConnectorAuthType(conn.AuthType),
		DisplayName:       conn.DisplayName,
		ExternalAccountID: conn.ExternalAccountID,
		Status:            conn.Status,
		Metadata:          conn.Metadata,
		Profile:           connectionProfileFromMetadata(conn.Metadata),
		CreatedAt:         formatModelTime(conn.CreatedAt),
		UpdatedAt:         formatModelTime(conn.UpdatedAt),
	}
}

func bindingToInfo(binding *model.ConnectorDirectoryBinding, conn *model.ConnectorConnection) *dto.ConnectorDirectoryBindingInfo {
	if binding == nil {
		return nil
	}
	info := &dto.ConnectorDirectoryBindingInfo{
		ID:            binding.ID,
		OwnerUsername: binding.OwnerUsername,
		TenantUser:    binding.TenantUser,
		App:           binding.App,
		ResourcePath:  binding.ResourcePath,
		Provider:      binding.Provider,
		ConnectionID:  binding.ConnectionID,
		CreatedAt:     formatModelTime(binding.CreatedAt),
		UpdatedAt:     formatModelTime(binding.UpdatedAt),
	}
	if conn != nil {
		info.Connection = connectionToInfo(conn)
	}
	return info
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func normalizeConnectorAuthType(authType string) (string, error) {
	authType = strings.ToLower(strings.TrimSpace(authType))
	if authType == "" {
		return model.ConnectorAuthTypeOAuth2User, nil
	}
	if authType != model.ConnectorAuthTypeOAuth2User {
		return "", fmt.Errorf("auth_type 目前仅支持 %s", model.ConnectorAuthTypeOAuth2User)
	}
	return authType, nil
}

func defaultConnectorAuthType(authType string) string {
	normalized, err := normalizeConnectorAuthType(authType)
	if err != nil {
		return model.ConnectorAuthTypeOAuth2User
	}
	return normalized
}

func normalizeConnectorResourcePath(resourcePath string) string {
	raw := strings.TrimSpace(resourcePath)
	if raw == "" {
		return ""
	}
	trimmed := strings.Trim(raw, "/")
	if trimmed == "" {
		return connectorGlobalResourcePath
	}
	return access.NormalizeResourcePath(raw)
}

func hasConnectorWildcardResourcePath(resourcePath string) bool {
	return strings.Contains(strings.TrimSpace(resourcePath), "*")
}

func defaultConnectorResourcePath(resourcePath string) string {
	path := normalizeConnectorResourcePath(resourcePath)
	if path == "" {
		return connectorGlobalResourcePath
	}
	return path
}

func parseConnectorBindingScope(resourcePath string) (tenantUser, app string, err error) {
	if normalizeConnectorResourcePath(resourcePath) == connectorGlobalResourcePath {
		return connectorGlobalScopePart, connectorGlobalScopePart, nil
	}
	return access.ParseUserApp(resourcePath)
}

func marshalMetadata(metadata map[string]interface{}) (string, error) {
	if len(metadata) == 0 {
		return "", nil
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("metadata 必须是可序列化对象: %w", err)
	}
	return string(data), nil
}

func connectionProfileFromMetadata(raw string) *dto.ConnectorConnectionProfile {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return nil
	}
	profileValue, ok := metadata["profile"]
	if !ok || profileValue == nil {
		return legacyConnectionProfileFromMetadata(metadata)
	}
	data, err := json.Marshal(profileValue)
	if err != nil {
		return legacyConnectionProfileFromMetadata(metadata)
	}
	var profile dto.ConnectorConnectionProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return legacyConnectionProfileFromMetadata(metadata)
	}
	if profile == (dto.ConnectorConnectionProfile{}) {
		return nil
	}
	return &profile
}

func legacyConnectionProfileFromMetadata(metadata map[string]interface{}) *dto.ConnectorConnectionProfile {
	profile := &dto.ConnectorConnectionProfile{
		Provider:      oauthValueString(metadata["provider"]),
		AccountID:     oauthValueString(metadata["external_id"]),
		AccountName:   oauthValueString(metadata["username"]),
		AvatarURL:     oauthValueString(metadata["avatar_url"]),
		AccountURL:    oauthValueString(metadata["provider_account_url"]),
		WorkspaceID:   oauthValueString(metadata["workspace_id"]),
		WorkspaceName: oauthValueString(metadata["workspace_name"]),
		WorkspaceIcon: oauthValueString(metadata["workspace_icon"]),
	}
	profile.DisplayName = firstNonEmpty(profile.WorkspaceName, profile.AccountName)
	if *profile == (dto.ConnectorConnectionProfile{}) {
		return nil
	}
	return profile
}

func newConnectionID() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "conn_" + hex.EncodeToString(b[:]), nil
}

func formatModelTime(t models.Time) string {
	return time.Time(t).Format(time.RFC3339)
}

func resourcePathCandidates(resourcePath string) []string {
	path := defaultConnectorResourcePath(resourcePath)
	if path == "" {
		return nil
	}
	if path == connectorGlobalResourcePath {
		return []string{connectorGlobalResourcePath}
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	candidates := make([]string, 0, len(parts)+1)
	candidates = append(candidates, connectorGlobalResourcePath)
	for i := len(parts); i >= 2; i-- {
		candidates = append(candidates, "/"+strings.Join(parts[:i], "/"))
	}
	return candidates
}
