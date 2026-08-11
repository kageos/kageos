package serverx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultHTTPReadHeaderTimeout = 5 * time.Second
	DefaultHTTPReadTimeout       = 30 * time.Second
	DefaultHTTPIdleTimeout       = 60 * time.Second
	DefaultHTTPShutdownTimeout   = 5 * time.Second
)

type HTTPServer struct {
	server   *http.Server
	listener net.Listener
	serveErr chan error
}

func StartHTTPServer(ctx context.Context, addr string, handler http.Handler) (*HTTPServer, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, fmt.Errorf("http server address is empty")
	}
	if handler == nil {
		return nil, fmt.Errorf("http server handler is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &HTTPServer{
		server: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: DefaultHTTPReadHeaderTimeout,
			ReadTimeout:       DefaultHTTPReadTimeout,
			IdleTimeout:       DefaultHTTPIdleTimeout,
		},
		listener: listener,
		serveErr: make(chan error, 1),
	}

	go func() {
		err := s.server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		s.serveErr <- err
		close(s.serveErr)
	}()

	return s, nil
}

func (s *HTTPServer) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

func (s *HTTPServer) Err() <-chan error {
	if s == nil || s.serveErr == nil {
		ch := make(chan error)
		close(ch)
		return ch
	}
	return s.serveErr
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	if ctx == nil || ctx.Err() != nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultHTTPShutdownTimeout)
		defer cancel()
	}

	err := s.server.Shutdown(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	if err != nil {
		_ = s.server.Close()
		return err
	}

	select {
	case serveErr, ok := <-s.Err():
		if !ok {
			return nil
		}
		return serveErr
	case <-ctx.Done():
		_ = s.server.Close()
		return ctx.Err()
	}
}
