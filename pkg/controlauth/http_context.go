package controlauth

import (
	"context"
	"net/http"
)

// DelegatedHTTPRequestSigner is an in-process capability, not transport
// metadata. Its implementation decides which origin it may sign for.
type DelegatedHTTPRequestSigner func(req *http.Request, body []byte) (bool, error)

type delegatedHTTPRequestSignerContextKey struct{}

// WithDelegatedHTTPRequestSigner attaches a signing capability to a private,
// typed context key. Untrusted HTTP callers cannot manufacture this value.
func WithDelegatedHTTPRequestSigner(ctx context.Context, signer DelegatedHTTPRequestSigner) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if signer == nil {
		return ctx
	}
	return context.WithValue(ctx, delegatedHTTPRequestSignerContextKey{}, signer)
}

// PropagateDelegatedHTTPRequestSigner copies only the private delegation
// capability into a clean context. It deliberately does not inherit arbitrary
// string-key identity values or cancellation semantics from source.
func PropagateDelegatedHTTPRequestSigner(source, target context.Context) context.Context {
	if target == nil {
		target = context.Background()
	}
	if source == nil {
		return target
	}
	signer, _ := source.Value(delegatedHTTPRequestSignerContextKey{}).(DelegatedHTTPRequestSigner)
	return WithDelegatedHTTPRequestSigner(target, signer)
}

// ApplyDelegatedHTTPRequestSignature invokes the private capability carried by
// req.Context. It reports false when the request should remain unsigned.
func ApplyDelegatedHTTPRequestSignature(req *http.Request, body []byte) (bool, error) {
	if req == nil {
		return false, nil
	}
	signer, _ := req.Context().Value(delegatedHTTPRequestSignerContextKey{}).(DelegatedHTTPRequestSigner)
	if signer == nil {
		return false, nil
	}
	return signer(req, body)
}
