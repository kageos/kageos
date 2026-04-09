package websearch

import (
	"context"
	"net/http"
	"time"
)

var searchHTTPClient = &http.Client{
	Timeout: 15 * time.Second,
}

var followURLHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return http.ErrUseLastResponse
		}
		return nil
	},
}

func doGET(ctx context.Context, client *http.Client, rawURL string, configure func(*http.Request)) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if configure != nil {
		configure(req)
	}
	return client.Do(req)
}
