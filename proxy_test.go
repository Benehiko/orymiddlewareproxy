package oryproxy_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Benehiko/oryproxy"
	"github.com/cenkalti/backoff/v4"
	"github.com/gorilla/mux"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestProxy(t *testing.T) {
	ctx, ctxCancel := context.WithTimeoutCause(context.Background(), time.Second*5, context.DeadlineExceeded)

	r := mux.NewRouter()

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	ts := httptest.NewServer(r)

	t.Cleanup(ts.Close)

	// usually the Ory URL, but here we are mocking Ory's service
	config := oryproxy.NewDefaultConfig(ts.URL)

	require.Equal(t, ts.URL, config.OryProjectURL(ctx))

	proxy := oryproxy.NewOryProxy(config)

	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return proxy.ListenAndServe(egCtx, port)
	})

	eg.Go(func() error {
		// kill the proxy server after this function exits
		defer ctxCancel()

		req, err := http.NewRequestWithContext(egCtx, "GET", fmt.Sprintf("http://127.0.0.1:%d/.ory/health", port), nil)
		if err != nil {
			return err
		}

		client := &http.Client{}
		var resp *http.Response

		b := backoff.NewConstantBackOff(time.Millisecond * 100)
		err = backoff.Retry(func() error {
			resp, err = client.Do(req)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status code 200 but got %d", resp.StatusCode)
			}
			return nil
		}, backoff.WithMaxRetries(b, 5))

		if err != nil {
			return err
		}

		return nil
	})

	require.NoError(t, eg.Wait())
}
