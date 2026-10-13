package e2e

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/fdanielsin/inbackend/tests/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getStatus(t *testing.T, url string) int {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := callClient.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	return resp.StatusCode
}

// GIVEN the real binary started on its own PostgreSQL / WHEN that database stops, then comes back on the same port
// THEN /ready answers 503 while it is down and 200 again after, /health stays 200
func TestReady_RealBinaryFollowsTheDatabase(t *testing.T) {
	pg.Require(t)
	db := testsupport.StartPostgresOnHostPort(freePort(t))
	t.Cleanup(db.Terminate)
	db.Require(t)
	server := startServer(t, "DATABASE_URL="+db.DSN) // the last DATABASE_URL wins
	base := "http://" + server.addr
	require.Equal(t, http.StatusOK, getStatus(t, base+"/ready"))

	require.NoError(t, db.Stop(context.Background()))

	assert.Equal(t, http.StatusServiceUnavailable, getStatus(t, base+"/ready"))
	assert.Equal(t, http.StatusOK, getStatus(t, base+"/health"))

	require.NoError(t, db.Restart(context.Background()))
	assert.Eventually(t, func() bool { return getStatus(t, base+"/ready") == http.StatusOK },
		20*time.Second, 200*time.Millisecond, "the server must report ready again once the database is back:\n%s", server.output.String())
}
