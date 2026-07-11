package plaidly

import (
	"context"
	"net/http"

	"github.com/plaidly/plaidly-go/generated/plaidlyapi"
)

type RequestOption func(*requestOptions)

type requestOptions struct {
	idempotencyKey string
}

func WithIdempotencyKey(key string) RequestOption {
	return func(o *requestOptions) { o.idempotencyKey = key }
}

func buildRequestOptions(opts []RequestOption) *requestOptions {
	ro := &requestOptions{}
	for _, opt := range opts {
		opt(ro)
	}
	return ro
}

func (ro *requestOptions) toEditors() []plaidlyapi.RequestEditorFn {
	if ro.idempotencyKey == "" {
		return nil
	}
	key := ro.idempotencyKey
	return []plaidlyapi.RequestEditorFn{
		func(_ context.Context, req *http.Request) error {
			req.Header.Set("Idempotency-Key", key)
			return nil
		},
	}
}
