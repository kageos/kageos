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

// mergeControlServiceConfig 合并 Control Service 配置
// 如果服务配置了某个字段，使用服务配置；否则使用全局配置
func mergeControlServiceConfig(global ControlServiceClientConfig, service ControlServiceClientConfig) ControlServiceClientConfig {
	result := global
	// 注意：Enabled 字段的默认值是 true，所以需要特殊处理
	// 如果服务显式设置了 enabled=false，则使用服务配置
	// 否则，如果服务配置了其他字段（如 encryption_key），则认为服务想要启用
	if !service.Enabled && global.Enabled {
		// 服务显式禁用了，但全局启用了，使用服务配置
		result.Enabled = false
	} else if service.Enabled || service.EncryptionKey != "" || service.NatsURL != "" || service.KeyPath != "" {
		// 服务配置了相关字段，使用服务配置
		if service.EncryptionKey != "" {
			result.EncryptionKey = service.EncryptionKey
		}
		if service.NatsURL != "" {
			result.NatsURL = service.NatsURL
		}
		if service.KeyPath != "" {
			result.KeyPath = service.KeyPath
		}
		if service.Enabled {
			result.Enabled = service.Enabled
		}
	}
	return result
}

