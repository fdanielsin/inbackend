package githooks

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const zeroSHA = "0000000000000000000000000000000000000000"

type repo struct {
	t   *testing.T
	dir string
}

func newRepo(t *testing.T) *repo {
	t.Helper()
	r := &repo{t: t, dir: t.TempDir()}
	r.git("init", "-q")
	return r
}

func (r *repo) git(args ...string) string {
	r.t.Helper()
	cmd := exec.CommandContext(r.t.Context(), "git", args...) //nolint:gosec // G204: arguments written by this test file
	cmd.Dir = r.dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")
	out, err := cmd.CombinedOutput()
	require.NoError(r.t, err, string(out))
	return strings.TrimSpace(string(out))
}

func (r *repo) commit(message string) string {
	r.t.Helper()
	r.git("commit", "-q", "--allow-empty", "-m", message)
	return r.git("rev-parse", "HEAD")
}

// push lance la garde avec les lignes que git enverrait à un hook pre-push.
func (r *repo) push(releaseTag string, lines ...string) (string, error) {
	r.t.Helper()
	script, err := filepath.Abs("../../../scripts/check-push-main.sh")
	require.NoError(r.t, err)
	cmd := exec.CommandContext(r.t.Context(), "sh", script) //nolint:gosec // G204: runs this repository's own guard script
	cmd.Dir = r.dir
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n") + "\n")
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if releaseTag != "" {
		cmd.Env = append(cmd.Env, "RELEASE_TAG="+releaseTag)
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func line(localRef, localSHA, remoteRef string) string {
	return localRef + " " + localSHA + " " + remoteRef + " " + zeroSHA
}

func TestPushToMain_RefusedWithoutReleaseTag(t *testing.T) {
	r := newRepo(t)
	sha := r.commit("one")

	out, err := r.push("", line("refs/heads/develop", sha, "refs/heads/main"))

	require.Error(t, err)
	assert.Contains(t, out, "push vers main refusé")
}

func TestPushToMain_RefusedWhenTagIsMissingOrElsewhere(t *testing.T) {
	r := newRepo(t)
	first := r.commit("one")
	r.git("tag", "v1.1.0")
	second := r.commit("two")

	_, err := r.push("v9.9.9", line("refs/heads/develop", second, "refs/heads/main"))
	require.Error(t, err, "unknown tag")

	_, err = r.push("v1.1.0", line("refs/heads/develop", second, "refs/heads/main"))
	require.Error(t, err, "tag on another commit than the pushed one")

	_, err = r.push("v1.1.0", line("refs/heads/develop", first, "refs/heads/main"))
	require.NoError(t, err, "control: the tag is on the pushed commit")
}

func TestPushToMain_AllowedWithTagOnPushedCommit(t *testing.T) {
	r := newRepo(t)
	sha := r.commit("one")
	r.git("tag", "v1.1.0")

	_, err := r.push("v1.1.0", line("refs/heads/develop", sha, "refs/heads/main"))

	require.NoError(t, err)
}

func TestPushToMain_AllowedWithAnnotatedTag(t *testing.T) {
	r := newRepo(t)
	sha := r.commit("one")
	r.git("tag", "-a", "v1.1.0", "-m", "release")

	_, err := r.push("v1.1.0", line("refs/heads/develop", sha, "refs/heads/main"))

	require.NoError(t, err)
}

func TestPushToMain_DeletionRefusedEvenWithReleaseTag(t *testing.T) {
	r := newRepo(t)
	r.commit("one")
	r.git("tag", "v1.1.0")

	_, err := r.push("v1.1.0", line("(delete)", zeroSHA, "refs/heads/main"))

	require.Error(t, err)
}

func TestPushToMain_OtherRefsAreNotConcerned(t *testing.T) {
	r := newRepo(t)
	sha := r.commit("one")

	for _, remoteRef := range []string{
		"refs/heads/develop", "refs/heads/feature/x", "refs/heads/main-backup", "refs/heads/release/v1.1.0", "refs/tags/v1.1.0",
	} {
		_, err := r.push("", line("refs/heads/x", sha, remoteRef))
		assert.NoError(t, err, remoteRef)
	}
}

func TestPushToMain_OneRefAmongSeveralStillRefused(t *testing.T) {
	r := newRepo(t)
	sha := r.commit("one")

	_, err := r.push("",
		line("refs/heads/develop", sha, "refs/heads/develop"),
		line("refs/heads/develop", sha, "refs/heads/main"))

	require.Error(t, err)
}

func TestPushToMain_MainFirstThenOtherRefStillRefused(t *testing.T) {
	r := newRepo(t)
	sha := r.commit("one")

	_, err := r.push("",
		line("refs/heads/develop", sha, "refs/heads/main"),
		line("refs/heads/develop", sha, "refs/heads/develop"))

	require.Error(t, err)
}

func TestPushToMain_NothingToPushIsAllowed(t *testing.T) {
	r := newRepo(t)

	_, err := r.push("")

	require.NoError(t, err)
}
