package config

// mergeDBConfig 合并数据库配置
// 如果服务配置了某个字段，使用服务配置；否则使用全局配置
func mergeDBConfig(global DBConfig, service DBConfig) DBConfig {
	result := global

	// 服务配置覆盖全局配置
	if service.Type != "" {
		result.Type = service.Type
	}
	if service.Host != "" {
		result.Host = service.Host
	}
	if service.Port != 0 {
		result.Port = service.Port
	}
	if service.User != "" {
		result.User = service.User
	}
	if service.Password != "" {
		result.Password = service.Password
	}
	if service.Name != "" {
		result.Name = service.Name
	}
	if service.MaxIdleConns != 0 {
		result.MaxIdleConns = service.MaxIdleConns
	}
	if service.MaxOpenConns != 0 {
		result.MaxOpenConns = service.MaxOpenConns
	}
	if service.MaxLifetime != 0 {
		result.MaxLifetime = service.MaxLifetime
	}
	if service.LogLevel != "" {
		result.LogLevel = service.LogLevel
	}
	if service.SlowThreshold != 0 {
		result.SlowThreshold = service.SlowThreshold
	}

	return result
}

// mergeNatsConfig 合并 NATS 配置
// 如果服务配置了 URL，使用服务配置；否则使用全局配置
func mergeNatsConfig(global NatsConfig, service NatsConfig) NatsConfig {
	result := global
	if service.URL != "" {
		result.URL = service.URL
	}
	return result
}

// mergeJWTConfig 合并 JWT 配置
// 如果服务配置了某个字段，使用服务配置；否则使用全局配置
func mergeJWTConfig(global JWTConfig, service JWTConfig) JWTConfig {
	result := global
	if service.Secret != "" {
		result.Secret = service.Secret
	}
	if service.Issuer != "" {
		result.Issuer = service.Issuer
	}
	if service.AccessTokenExpire != 0 {
		result.AccessTokenExpire = service.AccessTokenExpire
	}
	if service.RefreshTokenExpire != 0 {
		result.RefreshTokenExpire = service.RefreshTokenExpire
	}
	return result
}
