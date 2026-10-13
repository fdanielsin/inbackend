package e2e

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Canary values with characters that logging and JSON escape (quotes, angle brackets, accents, capitals).
const (
	canaryName    = `Zanzibar "Canari" <Nôm>`
	canaryRenamed = `Renamé "Canari2" <Nôm2>`
	canaryEmail   = `Canari.Mail+Tag@Example.ORG`
)

func escapedForms(t *testing.T, value string) []string {
	t.Helper()
	asJSON, err := json.Marshal(value)
	require.NoError(t, err)
	return []string{value, strings.ToLower(value), strings.ToUpper(value), strconv.Quote(value), string(asJSON), strings.Trim(string(asJSON), `"`)}
}

func sendJSON(t *testing.T, method, url, body string) int {
	t.Helper()
	r := call(t, method, url, body, map[string]string{"Content-Type": "application/json"})
	return r.status
}

// GIVEN the real binary, logging at Debug (no ENV), and canary values for a name, a new name and an e-mail
// WHEN a user is created, read, renamed (200, 400 with the e-mail in the body, 404) and deleted (200, 404, and 500 made by a database trigger)
// THEN no line of stdout or stderr holds a canary, raw, in capitals or lower case, quoted or escaped as JSON (USER-R15)
func TestServerOutput_UserLifecycleNeverContainsPersonalData(t *testing.T) {
	pg.Require(t)
	pg.NewDB(t) // migrated, tables empty
	admin := pg.Open(t)
	port := freePort(t)
	output := &syncBuffer{}
	cmd := exec.CommandContext(t.Context(), buildServer(t)) //nolint:gosec // G204: runs the binary built by buildServer
	cmd.Env = serverEnv("DATABASE_URL="+pg.DSN, "PORT="+port)
	cmd.Stdout = output
	cmd.Stderr = output
	cmd.WaitDelay = 5 * time.Second // the server is killed on cleanup: never wait on a pipe a child keeps open
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	base := "http://127.0.0.1:" + port
	require.Eventually(t, func() bool {
		resp, err := http.Get(base + "/health") //nolint:noctx // polling a local test server
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 20*time.Second, 100*time.Millisecond,
		"the server must start:\n%s", output.String())

	create, err := json.Marshal(map[string]string{"name": canaryName, "email": canaryEmail})
	require.NoError(t, err)
	var created userJSON
	r := call(t, http.MethodPost, base+"/api/v1/users", string(create), map[string]string{"Content-Type": "application/json"})
	require.Equal(t, http.StatusCreated, r.status)
	var env struct{ Data userJSON }
	require.NoError(t, json.Unmarshal(r.body, &env))
	created = env.Data
	user := base + "/api/v1/users/" + strconv.Itoa(int(created.ID))
	rename, err := json.Marshal(map[string]string{"name": canaryRenamed})
	require.NoError(t, err)
	withEmail, err := json.Marshal(map[string]string{"name": canaryRenamed, "email": canaryEmail})
	require.NoError(t, err)

	statuses := []int{
		call(t, http.MethodGet, user, "", nil).status,
		sendJSON(t, http.MethodPut, user, string(rename)),
		sendJSON(t, http.MethodPut, user, string(withEmail)),
		sendJSON(t, http.MethodPut, base+"/api/v1/users/999999", string(rename)),
		sendJSON(t, http.MethodPut, base+"/api/v1/users/abc", string(rename)),
		sendJSON(t, http.MethodPost, base+"/api/v1/users", string(create)), // 409 EMAIL_TAKEN
	}
	assert.Equal(t, []int{http.StatusOK, http.StatusOK, http.StatusBadRequest, http.StatusNotFound, http.StatusBadRequest, http.StatusConflict}, statuses)

	// a task whose deletion fails makes the deletion of its user a 500, and leaves the user in place
	require.NoError(t, admin.Exec("INSERT INTO task_models (title, description, user_id, completed) VALUES ('t', '', ?, false)", created.ID).Error)
	require.NoError(t, admin.Exec(`CREATE FUNCTION canary_fail() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'simulated failure for the log test'; END $$ LANGUAGE plpgsql`).Error)
	require.NoError(t, admin.Exec("CREATE TRIGGER canary_fail BEFORE DELETE ON task_models FOR EACH ROW EXECUTE FUNCTION canary_fail()").Error)
	failed := call(t, http.MethodDelete, user, "", nil).status
	require.NoError(t, admin.Exec("DROP TRIGGER canary_fail ON task_models").Error)
	require.NoError(t, admin.Exec("DROP FUNCTION canary_fail()").Error)
	deleted := call(t, http.MethodDelete, user, "", nil).status
	again := call(t, http.MethodDelete, user, "", nil).status
	assert.Equal(t, []int{http.StatusInternalServerError, http.StatusOK, http.StatusNotFound}, []int{failed, deleted, again})

	require.Eventually(t, func() bool { return strings.Count(output.String(), "/api/v1/users") >= 14 }, 5*time.Second, 50*time.Millisecond,
		"the request logs must be written before they are checked:\n%s", output.String())
	logs := output.String()

	t.Run("USER-R15: the harness sees the log lines of the operations, so the check can fail (happy path)", func(t *testing.T) {
		assert.Contains(t, logs, "Updating user")
		assert.Contains(t, logs, "Deleting user")
		assert.Contains(t, logs, "level=ERROR", "the 500 is logged, so the check sees the error path")
	})
	t.Run("USER-R15: no canary appears in any log line, at any level (happy path)", func(t *testing.T) {
		lower := strings.ToLower(logs)
		for _, canary := range []string{canaryName, canaryRenamed, canaryEmail, string(create), string(rename), string(withEmail)} {
			for _, form := range escapedForms(t, canary) {
				assert.NotContains(t, logs, form)
			}
		}
		for _, token := range []string{"zanzibar", "canari", "renam", "nôm", "example.org"} {
			assert.NotContains(t, lower, token)
		}
		assert.NotContains(t, logs, "simulated failure", "the text of the database error is not logged")
	})
}
