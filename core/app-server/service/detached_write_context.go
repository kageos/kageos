package service

import (
	"context"
	"time"
)

const detachedWriteTimeout = 5 * time.Second

func newDetachedWriteContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(parent), detachedWriteTimeout)
}
