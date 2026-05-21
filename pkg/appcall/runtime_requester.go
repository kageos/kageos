package appcall

import (
	"context"
	"time"

	"github.com/kageos/kageos/pkg/msgx"
)

// runtimeRequester 封装 app-runtime 管理类 request-reply 调用。
type runtimeRequester struct {
	connProvider ConnProvider
	timeout      time.Duration
}

func newRuntimeRequester(connProvider ConnProvider, timeout time.Duration) *runtimeRequester {
	return &runtimeRequester{
		connProvider: connProvider,
		timeout:      timeout,
	}
}

func (r *runtimeRequester) requestByHost(ctx context.Context, hostID int64, subject string, req, resp interface{}) error {
	conn, err := r.connProvider.GetConnByHost(hostID)
	if err != nil {
		return err
	}

	_, err = msgx.RequestJSON(ctx, conn, subject, req, resp, r.timeout)
	return err
}
