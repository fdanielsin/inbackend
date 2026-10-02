package e2e

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/pgstore"
	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GIVEN surrounding spaces / THEN the length is counted before trimming (decided): maximum + 1 rune
// with a leading space is refused, maximum - 1 runes plus a trailing space is accepted and stored trimmed
func TestRequestValidation_LengthIsCountedBeforeTrimming(t *testing.T) {
	for _, lc := range lengthCases() {
		t.Run(lc.field, func(t *testing.T) {
			s := newTestServer(t)
			r, _ := lc.send(t, s, " "+textOfRunes(lc.field, "a", lc.max))
			requireError(t, r, http.StatusBadRequest, "VALIDATION_FAILED", english(lc.tooLong))

			inside := textOfRunes(lc.field, "a", lc.max-1)
			r, stored := lc.send(t, newTestServer(t), inside+" ")
			require.Contains(t, []int{http.StatusCreated, http.StatusOK}, r.code, "body: %s", r.body)
			assert.Equal(t, inside, stored())
		})
	}
}

// GIVEN text with surrounding white space (space, tab, new line) / WHEN POST or PUT
// THEN stored and returned trimmed, inner spaces kept; a description of spaces only becomes ""
func TestRequestValidation_TrimsTextFields(t *testing.T) {
	s := newTestServer(t)

	user := s.createUser(" \tAl  ice\n ", "trim@example.com")
	assert.Equal(t, "Al  ice", user.Name)

	task := s.createTask(user.ID, "  Buy  milk ", "\n Semi skimmed \t")
	assert.Equal(t, taskJSON{ID: task.ID, Title: "Buy  milk", Description: "Semi skimmed", UserID: user.ID}, task)

	r := s.do(http.MethodPut, fmt.Sprintf("/api/v1/tasks/%d", task.ID), `{"title":" New title ","description":"   "}`)
	require.Equal(t, http.StatusOK, r.code, "body: %s", r.body)
	assert.Equal(t, []taskJSON{{ID: task.ID, Title: "New title", Description: "", UserID: user.ID}}, s.listTasks(user.ID))
}

// GIVEN a name or a title made of white space only (space, tab, new line, no-break space)
// THEN 400 VALIDATION_FAILED with the "cannot be empty" message, nothing stored or changed;
// a zero-width space is not white space for strings.TrimSpace and stays accepted (decided)
func TestRequestValidation_RefusesBlankNameAndTitle(t *testing.T) {
	for name, blank := range map[string]string{"spaces": "   ", "tab and new line": "\t\n", "no-break space": "\u00a0"} {
		t.Run(name, func(t *testing.T) {
			s := newTestServer(t)
			owner := s.createUser("Owner", "owner@example.com")
			task := s.createTask(owner.ID, "Buy milk", "")

			r := s.do(http.MethodPost, "/api/v1/users", jsonBody(t, map[string]any{"name": blank, "email": "blank@example.com"}))
			requireError(t, r, http.StatusBadRequest, "VALIDATION_FAILED", english(shared.MsgUserNameEmpty))

			r = s.do(http.MethodPost, "/api/v1/tasks", jsonBody(t, map[string]any{"user_id": owner.ID, "title": blank}))
			requireError(t, r, http.StatusBadRequest, "VALIDATION_FAILED", english(shared.MsgTaskTitleEmpty))

			r = s.do(http.MethodPut, fmt.Sprintf("/api/v1/tasks/%d", task.ID), jsonBody(t, map[string]any{"title": blank}))
			requireError(t, r, http.StatusBadRequest, "VALIDATION_FAILED", english(shared.MsgTaskTitleEmpty))

			assert.Equal(t, int64(1), s.countRows(&pgstore.UserModel{}))
			assert.Equal(t, []taskJSON{task}, s.listTasks(owner.ID))
		})
	}

	s := newTestServer(t)
	assert.Equal(t, "\u200b", s.createUser("\u200b", "zwsp@example.com").Name)
}
