package dto

const (
	AppDBAccessRuntime   = "runtime"
	AppDBAccessMigration = "migration"
)

// AppDBCapability is an internal, short-lived capability carried from
// app-runtime to SDK app requests. It is intentionally not exposed through SDK
// getters; SDK uses it only inside GetGormDB.
type AppDBCapability struct {
	User      string `json:"user,omitempty" swaggerignore:"true"`
	App       string `json:"app,omitempty" swaggerignore:"true"`
	Version   string `json:"version,omitempty" swaggerignore:"true"`
	Router    string `json:"router,omitempty" swaggerignore:"true"`
	ExpiresAt int64  `json:"expires_at,omitempty" swaggerignore:"true"`
	Nonce     string `json:"nonce,omitempty" swaggerignore:"true"`
	Signature string `json:"signature,omitempty" swaggerignore:"true"`
}

// AppDBResolveReq asks app-runtime for the current package database DSN.
// It is used only by SDK internals over NATS.
type AppDBResolveReq struct {
	Capability  *AppDBCapability `json:"capability"`
	User        string           `json:"user"`
	App         string           `json:"app"`
	Version     string           `json:"version,omitempty"`
	PackagePath string           `json:"package_path"`
	Access      string           `json:"access,omitempty"`
}

// AppDBResolveResp returns a low-privilege DSN for one package database.
type AppDBResolveResp struct {
	Dialect      string `json:"dialect"`
	Access       string `json:"access,omitempty"`
	DatabaseName string `json:"database_name"`
	DSN          string `json:"dsn"`
	MaxOpenConns int    `json:"max_open_conns,omitempty"`
	MaxIdleConns int    `json:"max_idle_conns,omitempty"`
	MaxIdleTime  int    `json:"max_idle_time,omitempty"` // seconds
	MaxLifetime  int    `json:"max_lifetime,omitempty"`  // seconds
}
