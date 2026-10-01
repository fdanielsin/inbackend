// Package controllers tests the readiness probe with a stubbed dependency check.
package controllers

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fdanielsin/inbackend/internal/controllers/httpapi"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// probeServer mounts /ready on the production error handler and middlewares, with the given check.
func probeServer(t *testing.T, check httpapi.ReadinessCheck) (*echo.Echo, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	log := slog.Default()
	e := echo.New()
	e.HTTPErrorHandler = httpapi.NewErrorHandler(log)
	e.Use(httpapi.RequestID(log))
	e.Use(middleware.Recover())
	e.GET("/ready", httpapi.NewHealthController(log, check).Ready)
	return e, &logs
}

func probe(e *echo.Echo) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))
	return rec
}

// GIVEN a check that succeeds / WHEN /ready / THEN 200
func TestReady_CheckPasses(t *testing.T) {
	e, _ := probeServer(t, func(context.Context) error { return nil })

	rec := probe(e)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"success":true,"message":"Service is ready","data":{"status":"ready"}}`, rec.Body.String())
}

// GIVEN a check that never returns by itself / WHEN /ready / THEN the 2 s deadline stops it and the probe answers 503
func TestReady_CheckIsBoundedByItsDeadline(t *testing.T) {
	e, _ := probeServer(t, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	start := time.Now()
	rec := probe(e)
	elapsed := time.Since(start)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.GreaterOrEqual(t, elapsed, 1900*time.Millisecond, "the check gets its full 2 s")
	assert.Less(t, elapsed, 3*time.Second, "the probe answers before a 3 s probe timeout")
}

// GIVEN a check whose error carries connection details / WHEN /ready / THEN a generic body, and logs without the text
func TestReady_FailureDetailsNeverLeak(t *testing.T) {
	e, logs := probeServer(t, func(context.Context) error {
		return errors.New("dial postgres://admin:s3cr3t@db.internal:5432 refused")
	})

	rec := probe(e)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.JSONEq(t, `{"success":false,"error":"Database unavailable","error_code":"DATABASE_UNAVAILABLE"}`, rec.Body.String())
	assert.Equal(t, "no-store", rec.Header().Get(echo.HeaderCacheControl))
	assert.Contains(t, logs.String(), "error_type=*errors.errorString")
	for _, secret := range []string{"s3cr3t", "admin", "db.internal"} {
		assert.NotContains(t, rec.Body.String(), secret)
		assert.NotContains(t, logs.String(), secret)
	}
}

// GIVEN a dependency that fails then recovers / WHEN /ready is probed again / THEN 503 then 200
func TestReady_RecoversWhenTheDependencyComesBack(t *testing.T) {
	down := true
	e, _ := probeServer(t, func(context.Context) error {
		if down {
			return errors.New("connection refused")
		}
		return nil
	})

	first := probe(e)
	down = false
	second := probe(e)

	assert.Equal(t, http.StatusServiceUnavailable, first.Code)
	assert.Equal(t, http.StatusOK, second.Code)
}

// GIVEN a check that panics / WHEN /ready / THEN 500 INTERNAL_ERROR without detail
func TestReady_PanickingCheckIsAnInternalError(t *testing.T) {
	e, _ := probeServer(t, func(context.Context) error { panic("secret panic detail") })

	rec := probe(e)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	require.JSONEq(t, `{"success":false,"error":"Internal server error","error_code":"INTERNAL_ERROR"}`, rec.Body.String())
}

// GIVEN /ready / WHEN the check runs / THEN it receives a deadline 2 s ahead, the request's context being its parent
func TestReady_CheckReceivesATwoSecondDeadline(t *testing.T) {
	var remaining time.Duration
	var hasDeadline bool
	e, _ := probeServer(t, func(ctx context.Context) error {
		var deadline time.Time
		deadline, hasDeadline = ctx.Deadline()
		remaining = time.Until(deadline)
		return nil
	})

	probe(e)

	require.True(t, hasDeadline, "the check must be bounded")
	assert.Greater(t, remaining, 1900*time.Millisecond)
	assert.LessOrEqual(t, remaining, 2*time.Second)
}

// GIVEN a slow check that still finishes under the deadline (1.5 s) / WHEN /ready / THEN 200
func TestReady_SlowCheckUnderTheDeadlinePasses(t *testing.T) {
	e, _ := probeServer(t, func(ctx context.Context) error {
		select {
		case <-time.After(1500 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	assert.Equal(t, http.StatusOK, probe(e).Code)
}

// GIVEN a client that already gave up / WHEN /ready / THEN the check sees a cancelled context and the probe answers 503
func TestReady_CancelledRequestStopsTheCheck(t *testing.T) {
	var checkErr error
	e, _ := probeServer(t, func(ctx context.Context) error {
		checkErr = ctx.Err()
		return checkErr
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, http.MethodGet, "/ready", nil))

	require.ErrorIs(t, checkErr, context.Canceled)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
