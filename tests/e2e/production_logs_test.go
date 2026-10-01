package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GIVEN the real binary with ENV=production / WHEN it serves a probe, a 400, a user creation, then stops
// THEN every output line is JSON with time, level, source (the real caller) and msg; successful probes (Debug) are hidden;
// the 400 is a WARN access line; no personal data is written
func TestServerOutput_ProductionIsJSONWithoutDebug(t *testing.T) {
	s := startServer(t, "ENV=production")
	base := "http://" + s.addr
	body := fmt.Sprintf(`{"name":%q,"email":%q}`, personName, personEmail)

	require.Equal(t, http.StatusOK, getStatus(t, base+"/ready"))
	require.Equal(t, http.StatusBadRequest, getStatus(t, base+"/api/v1/users/abc"))
	require.Equal(t, http.StatusCreated, postJSON(t, base+"/api/v1/users", body))
	require.NoError(t, s.cmd.Process.Signal(syscall.SIGTERM))
	require.Equal(t, 0, s.waitExit(t, 15*time.Second))

	var routes []string
	output := strings.TrimSpace(s.output.String())
	for _, line := range strings.Split(output, "\n") {
		var entry map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &entry), "not a JSON line: %s", line)
		for _, key := range []string{"time", "level", "source", "msg"} {
			assert.Contains(t, entry, key, line)
		}
		assert.NotEqual(t, "DEBUG", entry["level"], line)
		if source, ok := entry["source"].(map[string]any); assert.True(t, ok, line) {
			assert.NotContains(t, source["file"], "infrastructure/logger", "source is the real caller: %s", line)
		}
		if entry["msg"] == "Request" {
			routes = append(routes, fmt.Sprintf("%v %v %v", entry["route"], entry["status"], entry["level"]))
		}
	}
	assert.Equal(t, []string{"/api/v1/users/:id 400 WARN", "/api/v1/users 201 INFO"}, routes,
		"the probes polled by startServer and the /ready call are Debug, hence hidden")
	assert.Contains(t, output, `"msg":"Starting server"`)
	assert.Contains(t, output, `"msg":"Server stopped"`)
	for _, personal := range forbiddenInLogs(body) {
		assert.NotContains(t, output, personal)
	}
}
