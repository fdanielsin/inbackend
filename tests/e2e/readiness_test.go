package e2e

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/fdanielsin/inbackend/internal/app"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type readyJSON struct {
	Status string `json:"status"`
}

// GIVEN a reachable database / WHEN GET /ready (with or without query) / THEN 200 « Service is ready », never cached
func TestReady_DatabaseReachable(t *testing.T) {
	for _, path := range []string{"/ready", "/ready?verbose=1&x=%00"} {
		t.Run(path, func(t *testing.T) {
			s := newTestServer(t)

			r := s.do(http.MethodGet, path, "")

			require.Equal(t, http.StatusOK, r.code, "body: %s", r.body)
			env := decode[readyJSON](t, r)
			assert.True(t, env.Success)
			assert.Equal(t, "Service is ready", env.Message)
			assert.Equal(t, readyJSON{Status: "ready"}, env.Data)
			assert.Equal(t, "no-store", r.header.Get(echo.HeaderCacheControl))
			assert.NotEmpty(t, r.header.Get(echo.HeaderXRequestID))
		})
	}
}

// GIVEN the probes / THEN /health is never cached either
func TestHealth_IsNeverCached(t *testing.T) {
	s := newTestServer(t)

	r := s.do(http.MethodGet, "/health", "")

	assert.Equal(t, "no-store", r.header.Get(echo.HeaderCacheControl))
	assert.NotEmpty(t, r.header.Get(echo.HeaderXRequestID))
}

// GIVEN the database is unavailable / WHEN the probes are called
// THEN /ready answers 503 DATABASE_UNAVAILABLE without any connection detail, /health stays 200
func TestReady_DatabaseDown(t *testing.T) {
	logs := captureLogs(t)
	s := newTestServer(t)
	s.closeDatabase()

	ready := s.do(http.MethodGet, "/ready", "")
	health := s.do(http.MethodGet, "/health", "")

	requireError(t, ready, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Database unavailable")
	assert.Equal(t, "no-store", ready.header.Get(echo.HeaderCacheControl))
	assert.Equal(t, http.StatusOK, health.code, "liveness does not depend on the database")
	failures := logLines(t, logs, "Readiness check failed")
	require.Len(t, failures, 1)
	assert.Equal(t, "ERROR", failures[0]["level"])
	assert.Equal(t, "DATABASE_UNAVAILABLE", failures[0]["error_code"])
	assert.Contains(t, failures[0], "error_type")
	for _, secret := range []string{"testpass", "testuser", "database is closed"} {
		assert.NotContains(t, string(ready.body), secret)
		assert.NotContains(t, logs.String(), secret)
	}
}

// GIVEN /ready / WHEN called with another method / THEN 405 METHOD_NOT_ALLOWED
func TestReady_OtherMethodIsNotAllowed(t *testing.T) {
	s := newTestServer(t)

	requireError(t, s.doOutsideOperations(http.MethodPost, "/ready"), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
}

// GIVEN 50 probes at once / THEN all answer 200 and no database connection stays in use
func TestReady_ParallelProbesReleaseTheirConnections(t *testing.T) {
	s := newTestServer(t)
	codes := make([]int, 50)
	var wg sync.WaitGroup
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i] = s.doRaw(http.MethodGet, "/ready", "").code
		}(i)
	}
	wg.Wait()

	for i, code := range codes {
		assert.Equal(t, http.StatusOK, code, fmt.Sprintf("probe %d", i))
	}
	sqlDB, err := s.db.DB()
	require.NoError(t, err)
	assert.Zero(t, sqlDB.Stats().InUse, "every probe must release its connection")
}

// unresponsiveDatabase accepts TCP connections and never answers, like a hung database server.
func unresponsiveDatabase(t *testing.T) string {
	t.Helper()
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	var mu sync.Mutex
	var conns []net.Conn
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()
		}
	}()
	t.Cleanup(func() {
		_ = l.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, conn := range conns {
			_ = conn.Close()
		}
	})
	return fmt.Sprintf("postgres://testuser:testpass@%s/testdb?sslmode=disable", l.Addr())
}

// GIVEN a database that accepts connections but never answers / WHEN GET /ready through the real driver
// THEN the check gives up at its 2 s deadline: 503 DATABASE_UNAVAILABLE before a 3 s probe timeout
func TestReady_UnresponsiveDatabaseHitsTheDeadline(t *testing.T) {
	db, err := gorm.Open(postgres.Open(unresponsiveDatabase(t)), &gorm.Config{DisableAutomaticPing: true, Logger: gormlogger.Discard})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	s := &testServer{t: t, e: app.NewRouter(db), db: db}

	type result struct {
		req *http.Request
		r   response
	}
	done := make(chan result, 1)
	start := time.Now()
	go func() {
		req, r := s.send(http.MethodGet, "/ready", "")
		done <- result{req, r}
	}()
	var got result
	select {
	case got = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("/ready never answered: the readiness check ignores its deadline")
	}
	elapsed := time.Since(start)

	contract.ValidateResponse(t, got.req, got.r.code, got.r.header, got.r.body)
	requireError(t, got.r, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Database unavailable")
	assert.Equal(t, "no-store", got.r.header.Get(echo.HeaderCacheControl))
	assert.GreaterOrEqual(t, elapsed, 1900*time.Millisecond, "the check gets its full 2 s")
	assert.Less(t, elapsed, 3*time.Second, "the probe answers before a 3 s probe timeout")
}
