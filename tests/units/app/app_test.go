// Package app tests the composition helpers: CORS origins and the HTTP server lifecycle.
package app

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fdanielsin/inbackend/internal/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GIVEN CORS_ALLOWED_ORIGINS values / WHEN parsed / THEN trimmed origins, empty items dropped, "*" when nothing is left
func TestParseAllowedOrigins(t *testing.T) {
	cases := map[string][]string{
		"":                  {"*"},
		" , ,":              {"*"},
		"https://a.example": {"https://a.example"},
		" https://a.example , https://b.example,": {"https://a.example", "https://b.example"},
		"*": {"*"},
	}
	for value, want := range cases {
		assert.Equal(t, want, app.ParseAllowedOrigins(value), "%q", value)
	}
}

func listen(t *testing.T) net.Listener {
	t.Helper()
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	return l
}

func get(t *testing.T, url string) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	require.NoError(t, err)
	return http.DefaultClient.Do(req)
}

// GIVEN a running server / WHEN its context is cancelled / THEN Serve returns nil and stops listening
func TestServe_ReturnsNilAfterCancellation(t *testing.T) {
	l := listen(t)
	url := "http://" + l.Addr().String()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- app.Serve(ctx, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "ok")
		}), l)
	}()
	require.Eventually(t, func() bool {
		resp, err := get(t, url)
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 20*time.Millisecond)

	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(15 * time.Second):
		t.Fatal("Serve did not return after cancellation")
	}
	resp, err := get(t, url)
	if err == nil {
		_ = resp.Body.Close()
	}
	assert.Error(t, err, "the server must no longer accept connections")
}

// GIVEN a request in progress / WHEN the context is cancelled / THEN the request completes before Serve returns
func TestServe_LetsARequestInProgressFinish(t *testing.T) {
	l := listen(t)
	url := "http://" + l.Addr().String()
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- app.Serve(ctx, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			close(started)
			<-release
			_, _ = io.WriteString(w, "finished")
		}), l)
	}()
	type result struct {
		body string
		err  error
	}
	answer := make(chan result, 1)
	go func() {
		resp, err := get(t, url)
		if err != nil {
			answer <- result{err: err}
			return
		}
		defer func() { _ = resp.Body.Close() }()
		b, err := io.ReadAll(resp.Body)
		answer <- result{body: string(b), err: err}
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("the request never reached the handler")
	}

	cancel()
	select {
	case err := <-done:
		t.Fatalf("Serve returned before the request finished: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	close(release)

	select {
	case r := <-answer:
		require.NoError(t, r.err)
		assert.Equal(t, "finished", r.body)
	case <-time.After(5 * time.Second):
		t.Fatal("the request in progress never completed")
	}
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after the request finished")
	}
}

// GIVEN a listener that is already closed / WHEN Serve starts / THEN it returns the error at once
func TestServe_ClosedListenerReturnsAnError(t *testing.T) {
	l := listen(t)
	require.NoError(t, l.Close())

	done := make(chan error, 1)
	go func() { done <- app.Serve(context.Background(), http.NotFoundHandler(), l) }()

	select {
	case err := <-done:
		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "closed"), err.Error())
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return on a closed listener")
	}
}
