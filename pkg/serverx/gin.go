package serverx

import "github.com/gin-gonic/gin"

// GinOption configures a shared Gin bootstrap.
type GinOption func(*ginOptions)

type ginOptions struct {
	mode       *string
	middleware []gin.HandlerFunc
}

// NewGin creates a Gin engine with optional mode and middleware setup.
func NewGin(opts ...GinOption) *gin.Engine {
	cfg := ginOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	if cfg.mode != nil {
		gin.SetMode(*cfg.mode)
	}

	engine := gin.New()
	if len(cfg.middleware) > 0 {
		engine.Use(cfg.middleware...)
	}
	return engine
}

// WithMode sets the Gin global mode before the engine is created.
func WithMode(mode string) GinOption {
	return func(cfg *ginOptions) {
		cfg.mode = &mode
	}
}

// WithDebug switches between debug and release mode.
func WithDebug(debug bool) GinOption {
	if debug {
		return WithMode(gin.DebugMode)
	}
	return WithMode(gin.ReleaseMode)
}

// WithRecovery adds Gin's default recovery middleware.
func WithRecovery() GinOption {
	return WithMiddleware(gin.Recovery())
}

// WithLogger adds Gin's default access logger middleware.
func WithLogger() GinOption {
	return WithMiddleware(gin.Logger())
}

// WithMiddleware appends middleware in order.
func WithMiddleware(middleware ...gin.HandlerFunc) GinOption {
	return func(cfg *ginOptions) {
		cfg.middleware = append(cfg.middleware, middleware...)
	}
}
