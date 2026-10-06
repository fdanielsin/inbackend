// Package applog tests the construction of the application's slog logger.
package applog

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fdanielsin/inbackend/internal/infrastructure/applog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lines splits the output into its lines, failing if the last one is not terminated.
func lines(t *testing.T, out *bytes.Buffer) []string {
	t.Helper()
	text := out.String()
	require.True(t, strings.HasSuffix(text, "\n"), "every entry ends with a newline: %q", text)
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

// jsonEntry decodes a single JSON log line.
func jsonEntry(t *testing.T, line string) map[string]any {
	t.Helper()
	var entry map[string]any
	require.NoError(t, json.Unmarshal([]byte(line), &entry), "not a JSON line: %s", line)
	return entry
}

// GIVEN ENV=production / WHEN an entry is logged / THEN one JSON line with time, level, source, msg and typed attributes
func TestNew_ProductionWritesJSONLines(t *testing.T) {
	var out bytes.Buffer
	log := applog.New(&out, "production")

	log.Info("entry", slog.String("s", "text"), slog.Int("n", 3), slog.Bool("b", true))

	got := lines(t, &out)
	require.Len(t, got, 1)
	entry := jsonEntry(t, got[0])
	_, err := time.Parse(time.RFC3339Nano, entry["time"].(string))
	require.NoError(t, err, "time is RFC 3339")
	assert.Equal(t, "INFO", entry["level"])
	assert.Equal(t, "entry", entry["msg"])
	assert.Equal(t, "text", entry["s"])
	assert.InDelta(t, 3, entry["n"], 0)
	assert.Equal(t, true, entry["b"])
	source, ok := entry["source"].(map[string]any)
	require.True(t, ok, "source is an object: %v", entry["source"])
	assert.True(t, strings.HasSuffix(source["file"].(string), "logger_test.go"), "source points to the caller: %v", source)
	assert.Positive(t, source["line"])
}

// GIVEN ENV=development / WHEN an entry is logged / THEN one key=value text line, not JSON, with its source
func TestNew_DevelopmentWritesText(t *testing.T) {
	var out bytes.Buffer
	log := applog.New(&out, "development")

	log.Info("entry", slog.String("s", "text"))

	got := lines(t, &out)
	require.Len(t, got, 1)
	assert.True(t, strings.HasPrefix(got[0], "time="), got[0])
	assert.Contains(t, got[0], " level=INFO ")
	assert.Contains(t, got[0], "logger_test.go:")
	assert.Contains(t, got[0], " msg=entry ")
	assert.Contains(t, got[0], " s=text")
}

// GIVEN each ENV value / WHEN Debug and Info entries are logged
// THEN only exactly "production" gives JSON without Debug; any other value (empty, other case, prefix) is development
func TestNew_FormatAndLevelFollowEnv(t *testing.T) {
	cases := []struct {
		env       string
		json      bool
		showDebug bool
	}{
		{"production", true, false},
		{"development", false, true},
		{"", false, true},
		{"PRODUCTION", false, true},
		{"Production", false, true},
		{"prod", false, true},
		{" production", false, true},
	}
	for _, c := range cases {
		t.Run("env="+c.env, func(t *testing.T) {
			var out bytes.Buffer
			log := applog.New(&out, c.env)

			log.Debug("debug-entry")
			log.Info("info-entry")

			got := lines(t, &out)
			assert.Equal(t, c.showDebug, strings.Contains(out.String(), "debug-entry"), "Debug shown")
			assert.Contains(t, got[len(got)-1], "info-entry", "Info is always shown")
			assert.Equal(t, c.json, strings.HasPrefix(got[0], "{"), "JSON format")
		})
	}
}

// GIVEN attribute values with newlines, quotes and JSON / WHEN logged in either format
// THEN each entry stays on one line and the value comes back intact (no log line injection)
func TestNew_SpecialCharactersStayOnOneLine(t *testing.T) {
	value := "line1\nline2 \"quoted\" {\"level\":\"ERROR\"}\r\ttab"
	for _, env := range []string{"production", "development"} {
		t.Run(env, func(t *testing.T) {
			var out bytes.Buffer
			log := applog.New(&out, env)

			log.Warn("special", slog.String("v", value))

			got := lines(t, &out)
			require.Len(t, got, 1, "one entry, one line")
			if env == "production" {
				assert.Equal(t, value, jsonEntry(t, got[0])["v"])
			} else {
				assert.Contains(t, got[0], `v="line1\nline2 \"quoted\"`)
			}
		})
	}
}

// GIVEN a logger with attributes and a group / WHEN logged in production / THEN the attributes are kept and the group nests the rest
func TestNew_WithAttrsAndGroup(t *testing.T) {
	var out bytes.Buffer
	log := applog.New(&out, "production")

	log.With(slog.String("request_id", "r-1")).WithGroup("http").Info("grouped", slog.Int("status", 200))

	entry := jsonEntry(t, lines(t, &out)[0])
	assert.Equal(t, "r-1", entry["request_id"])
	assert.Equal(t, map[string]any{"status": float64(200)}, entry["http"])
}

// GIVEN an Error entry / WHEN logged in either format / THEN a single line, without stack trace (accepted with slog, E1b)
func TestNew_ErrorEntryHasNoStackTrace(t *testing.T) {
	for _, env := range []string{"production", "development"} {
		t.Run(env, func(t *testing.T) {
			var out bytes.Buffer
			log := applog.New(&out, env)

			log.Error("failure", slog.String("error_code", "INTERNAL_ERROR"))

			assert.Len(t, lines(t, &out), 1)
			assert.Contains(t, out.String(), "ERROR")
			assert.NotContains(t, out.String(), "goroutine")
		})
	}
}

// captureStderr runs fn while os.Stderr is a pipe, and returns what was written.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	original := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = original }()

	fn()

	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

// GIVEN ENV=production / WHEN Init / THEN the logger writes JSON to stderr and becomes slog's default
func TestInit_WritesToStderrAndBecomesTheDefault(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })
	t.Setenv("ENV", "production")

	stderr := captureStderr(t, func() {
		log := applog.Init()
		assert.Same(t, log, slog.Default())
		slog.Info("default-entry")
	})

	assert.True(t, strings.HasPrefix(stderr, "{"), "JSON in production:\n%s", stderr)
	assert.Contains(t, stderr, `"msg":"default-entry"`)
}
