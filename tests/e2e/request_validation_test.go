package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/pgstore"
	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// familyEmoji is one grapheme made of 5 runes (three people joined by two zero-width joiners):
// lengths are counted in runes, as JSON Schema maxLength does, not in graphemes nor in bytes.
const familyEmoji = "👨\u200d👩\u200d👧"

func english(key shared.MessageKey) string { return shared.Message(shared.LangEnglish, key) }

// sendRaw sends body through the router with the given Content-Type, none when empty.
func (s *testServer) sendRaw(method, path, contentType, body string) response {
	s.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set(echo.HeaderContentType, contentType)
	}
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	return response{code: rec.Code, body: rec.Body.Bytes(), header: rec.Header()}
}

func jsonBody(t *testing.T, fields map[string]any) string {
	t.Helper()
	body, err := json.Marshal(fields)
	require.NoError(t, err)
	return string(body)
}

// GIVEN the answers of 2f41faf to bodies the strict validation must not change (decided: D2, D5)
// WHEN the same requests are sent / THEN the same status, code and message, and nothing stored or changed
func TestRequestValidation_KeepsTheCurrentAnswers(t *testing.T) {
	cases := map[string]struct {
		method, contentType, body string
		status                    int
		code, message             string
	}{
		"users without Content-Type":   {http.MethodPost, "", `{"name":"A","email":"a@b.co"}`, http.StatusBadRequest, "INVALID_BODY", english(shared.MsgInvalidBody)},
		"users as text/plain":          {http.MethodPost, "text/plain", `{"name":"A","email":"a@b.co"}`, http.StatusBadRequest, "INVALID_BODY", english(shared.MsgInvalidBody)},
		"users empty body":             {http.MethodPost, echo.MIMEApplicationJSON, ``, http.StatusBadRequest, "VALIDATION_FAILED", english(shared.MsgUserNameEmpty)},
		"users empty, no Content-Type": {http.MethodPost, "", ``, http.StatusBadRequest, "VALIDATION_FAILED", english(shared.MsgUserNameEmpty)},
		"users null body":              {http.MethodPost, echo.MIMEApplicationJSON, `null`, http.StatusBadRequest, "VALIDATION_FAILED", english(shared.MsgUserNameEmpty)},
		"users name null":              {http.MethodPost, echo.MIMEApplicationJSON, `{"name":null,"email":"a@b.co"}`, http.StatusBadRequest, "VALIDATION_FAILED", english(shared.MsgUserNameEmpty)},
		"users email null":             {http.MethodPost, echo.MIMEApplicationJSON, `{"name":"A","email":null}`, http.StatusBadRequest, "VALIDATION_FAILED", english(shared.MsgUserEmailEmpty)},
		"users array body":             {http.MethodPost, echo.MIMEApplicationJSON, `[]`, http.StatusBadRequest, "INVALID_BODY", english(shared.MsgInvalidBody)},
		"users string body":            {http.MethodPost, echo.MIMEApplicationJSON, `"x"`, http.StatusBadRequest, "INVALID_BODY", english(shared.MsgInvalidBody)},
		"tasks empty body":             {http.MethodPost, echo.MIMEApplicationJSON, ``, http.StatusBadRequest, "VALIDATION_FAILED", english(shared.MsgTaskTitleEmpty)},
		"tasks null body":              {http.MethodPost, echo.MIMEApplicationJSON, `null`, http.StatusBadRequest, "VALIDATION_FAILED", english(shared.MsgTaskTitleEmpty)},
		"tasks title null":             {http.MethodPost, echo.MIMEApplicationJSON, `{"user_id":1,"title":null}`, http.StatusBadRequest, "VALIDATION_FAILED", english(shared.MsgTaskTitleEmpty)},
		"tasks user_id null":           {http.MethodPost, echo.MIMEApplicationJSON, `{"user_id":null,"title":"T"}`, http.StatusBadRequest, "VALIDATION_FAILED", english(shared.MsgTaskUserIDInvalid)},
		"update without Content-Type":  {http.MethodPut, "", `{"title":"x"}`, http.StatusBadRequest, "INVALID_BODY", english(shared.MsgInvalidBody)},
		"update array body":            {http.MethodPut, echo.MIMEApplicationJSON, `[]`, http.StatusBadRequest, "INVALID_BODY", english(shared.MsgInvalidBody)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := newTestServer(t)
			owner := s.createUser("Owner", "owner@example.com")
			task := s.createTask(owner.ID, "Buy milk", "Semi-skimmed")
			path := "/api/v1/users"
			if strings.HasPrefix(name, "tasks") {
				path = "/api/v1/tasks"
			} else if strings.HasPrefix(name, "update") {
				path = fmt.Sprintf("/api/v1/tasks/%d", task.ID)
			}

			r := s.sendRaw(tc.method, path, tc.contentType, tc.body)

			requireError(t, r, tc.status, tc.code, tc.message)
			assert.Equal(t, int64(1), s.countRows(&pgstore.UserModel{}))
			assert.Equal(t, []taskJSON{task}, s.listTasks(owner.ID))
		})
	}
}

// GIVEN a task / WHEN PUT with an empty or a null body / THEN 200 and nothing changes, like {} (decided: D5)
func TestRequestValidation_UpdateWithoutFieldsChangesNothing(t *testing.T) {
	for name, body := range map[string]string{"empty body": ``, "null body": `null`} {
		t.Run(name, func(t *testing.T) {
			s := newTestServer(t)
			owner := s.createUser("Owner", "owner@example.com")
			task := s.createTask(owner.ID, "Buy milk", "Semi-skimmed")

			r := s.sendRaw(http.MethodPut, fmt.Sprintf("/api/v1/tasks/%d", task.ID), echo.MIMEApplicationJSON, body)

			require.Equal(t, http.StatusOK, r.code, "body: %s", r.body)
			assert.Equal(t, []taskJSON{task}, s.listTasks(owner.ID))
		})
	}
}

// GIVEN a valid JSON body under every Content-Type that Echo binds as JSON, parameters included,
// even malformed ones (Echo only reads what comes before the first ";") / THEN accepted, as before the validation
func TestRequestValidation_AcceptsTheCharsetParameter(t *testing.T) {
	for _, contentType := range []string{"application/json; charset=utf-8", "application/json; charset", "application/json ;foo", " application/json"} {
		t.Run(contentType, func(t *testing.T) {
			s := newTestServer(t)

			r := s.sendRaw(http.MethodPost, "/api/v1/users", contentType, `{"name":"A","email":"a@b.co"}`)

			require.Equal(t, http.StatusCreated, r.code, "body: %s", r.body)
		})
	}
}

// GIVEN a field the contract does not declare, also one differing only by case (encoding/json binds
// keys case-insensitively) / WHEN POST or PUT / THEN 400 INVALID_BODY, the value is not echoed,
// and nothing is stored or changed
func TestRequestValidation_RefusesUnknownFields(t *testing.T) {
	cases := map[string]struct{ method, path, body string }{
		"user with a role":             {http.MethodPost, "/api/v1/users", `{"name":"A","email":"a@b.co","role":"Secretadmin"}`},
		"user name in capitals":        {http.MethodPost, "/api/v1/users", `{"NAME":"Secretadmin","email":"a@b.co"}`},
		"user email differing by case": {http.MethodPost, "/api/v1/users", `{"name":"A","Email":"secretadmin@b.co"}`},
		"task with an id":              {http.MethodPost, "/api/v1/tasks", `{"user_id":1,"title":"T","id":99}`},
		"task title differing by case": {http.MethodPost, "/api/v1/tasks", `{"user_id":1,"title":"T","Title":"Secretadmin"}`},
		"update with a user_id":        {http.MethodPut, "/api/v1/tasks/1", `{"user_id":2}`},
		"update with an id":            {http.MethodPut, "/api/v1/tasks/1", `{"id":2,"completed":true}`},
		"update completed by case":     {http.MethodPut, "/api/v1/tasks/1", `{"Completed":true}`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := newTestServer(t)
			owner := s.createUser("Owner", "owner@example.com")
			task := s.createTask(owner.ID, "Buy milk", "Semi-skimmed")
			require.Equal(t, uint(1), task.ID)

			r := s.do(tc.method, tc.path, tc.body)

			requireError(t, r, http.StatusBadRequest, "INVALID_BODY", english(shared.MsgUnknownField))
			assert.NotContains(t, string(r.body), "ecretadmin", "the value sent is not echoed")
			assert.Equal(t, int64(1), s.countRows(&pgstore.UserModel{}))
			assert.Equal(t, []taskJSON{task}, s.listTasks(owner.ID))
		})
	}
}

// GIVEN null on an optional field / WHEN POST or PUT / THEN 400 INVALID_BODY and nothing stored or changed (decided: D3, D4, D6)
func TestRequestValidation_RefusesNullOnOptionalFields(t *testing.T) {
	cases := map[string]struct{ method, path, body string }{
		"task description null": {http.MethodPost, "/api/v1/tasks", `{"user_id":1,"title":"T","description":null}`},
		"task completed null":   {http.MethodPost, "/api/v1/tasks", `{"user_id":1,"title":"T","completed":null}`},
		"update title null":     {http.MethodPut, "/api/v1/tasks/1", `{"title":null}`},
		"update description":    {http.MethodPut, "/api/v1/tasks/1", `{"description":null}`},
		"update completed null": {http.MethodPut, "/api/v1/tasks/1", `{"completed":null}`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := newTestServer(t)
			owner := s.createUser("Owner", "owner@example.com")
			task := s.createTask(owner.ID, "Buy milk", "Semi-skimmed")

			r := s.do(tc.method, tc.path, tc.body)

			requireError(t, r, http.StatusBadRequest, "INVALID_BODY", english(shared.MsgInvalidBody))
			assert.Equal(t, []taskJSON{task}, s.listTasks(owner.ID))
		})
	}
}

// lengthCase is a text field with its maximum length in runes, counted before trimming (decided).
type lengthCase struct {
	field   string
	max     int
	tooLong shared.MessageKey
	// send returns the response and the value stored for value.
	send func(t *testing.T, s *testServer, value string) (response, func() string)
}

func lengthCases() []lengthCase {
	createUser := func(field string) func(*testing.T, *testServer, string) (response, func() string) {
		return func(t *testing.T, s *testServer, value string) (response, func() string) {
			fields := map[string]any{"name": "A", "email": "a@b.co", field: value}
			r := s.do(http.MethodPost, "/api/v1/users", jsonBody(t, fields))
			return r, func() string {
				created := decode[userJSON](t, r).Data
				if field == "name" {
					return created.Name
				}
				return created.Email
			}
		}
	}
	createTask := func(field string) func(*testing.T, *testServer, string) (response, func() string) {
		return func(t *testing.T, s *testServer, value string) (response, func() string) {
			owner := s.createUser("Owner", "owner@example.com")
			fields := map[string]any{"user_id": owner.ID, "title": "T", field: value}
			r := s.do(http.MethodPost, "/api/v1/tasks", jsonBody(t, fields))
			return r, func() string { return taskField(s.listTasks(owner.ID)[0], field) }
		}
	}
	updateTask := func(field string) func(*testing.T, *testServer, string) (response, func() string) {
		return func(t *testing.T, s *testServer, value string) (response, func() string) {
			owner := s.createUser("Owner", "owner@example.com")
			task := s.createTask(owner.ID, "Buy milk", "Semi-skimmed")
			r := s.do(http.MethodPut, fmt.Sprintf("/api/v1/tasks/%d", task.ID), jsonBody(t, map[string]any{field: value}))
			return r, func() string { return taskField(s.listTasks(owner.ID)[0], field) }
		}
	}
	return []lengthCase{
		{"user name", 100, shared.MsgUserNameTooLong, createUser("name")},
		{"user email", 254, shared.MsgUserEmailTooLong, createUser("email")},
		{"task title", 200, shared.MsgTaskTitleTooLong, createTask("title")},
		{"task description", 2000, shared.MsgTaskDescriptionTooLong, createTask("description")},
		{"updated title", 200, shared.MsgTaskTitleTooLong, updateTask("title")},
		{"updated description", 2000, shared.MsgTaskDescriptionTooLong, updateTask("description")},
	}
}

func taskField(task taskJSON, field string) string {
	if field == "title" {
		return task.Title
	}
	return task.Description
}

// textOfRunes builds a value of exactly n runes from unit; an email keeps a valid shape.
func textOfRunes(field string, unit string, n int) string {
	if field == "user email" {
		const domain = "@example.com"
		local := strings.Repeat(unit, (n-len(domain))/len([]rune(unit)))
		return local + strings.Repeat("a", n-len(domain)-len([]rune(local))) + domain
	}
	whole := strings.Repeat(unit, n/len([]rune(unit)))
	return whole + strings.Repeat("a", n-len([]rune(whole)))
}

// GIVEN each text field at its maximum length in runes, ASCII, emoji (4 bytes each) and ZWJ sequences
// (5 runes per grapheme) / THEN accepted and stored as sent; one rune more / THEN 400 VALIDATION_FAILED
// with the field's message, nothing stored or changed
func TestRequestValidation_LengthBoundaries(t *testing.T) {
	for _, lc := range lengthCases() {
		for unitName, unit := range map[string]string{"ascii": "a", "emoji": "😀", "zwj": familyEmoji} {
			t.Run(lc.field+" "+unitName, func(t *testing.T) {
				atMax := textOfRunes(lc.field, unit, lc.max)
				require.Equal(t, lc.max, len([]rune(atMax)))

				s := newTestServer(t)
				r, stored := lc.send(t, s, atMax)
				require.Contains(t, []int{http.StatusCreated, http.StatusOK}, r.code, "body: %s", r.body)
				assert.Equal(t, strings.ToLower(atMax), strings.ToLower(stored()))

				over := newTestServer(t)
				r, _ = lc.send(t, over, atMax+"a")
				requireError(t, r, http.StatusBadRequest, "VALIDATION_FAILED", english(lc.tooLong))
			})
		}
	}
}

// GIVEN user_id at the uint32 maximum of the contract and one above / THEN 404 USER_NOT_FOUND,
// then 400 VALIDATION_FAILED (decided: D1)
func TestRequestValidation_UserIDMaximum(t *testing.T) {
	s := newTestServer(t)

	requireError(t, s.do(http.MethodPost, "/api/v1/tasks", `{"user_id":4294967295,"title":"T"}`), http.StatusNotFound, "USER_NOT_FOUND", english(shared.MsgUserNotFound))
	requireError(t, s.do(http.MethodPost, "/api/v1/tasks", `{"user_id":4294967296,"title":"T"}`), http.StatusBadRequest, "VALIDATION_FAILED", english(shared.MsgTaskUserIDOutOfRange))
}

// GIVEN the database is down / WHEN an invalid body is sent / THEN 400 before any database call;
// a valid body gets the 500 of the outage
func TestRequestValidation_ComesBeforeTheDatabase(t *testing.T) {
	s := newTestServer(t)
	s.closeDatabase()

	requireError(t, s.do(http.MethodPost, "/api/v1/users", `{"name":"A","email":"a@b.co","role":1}`), http.StatusBadRequest, "INVALID_BODY", english(shared.MsgUnknownField))
	requireError(t, s.do(http.MethodPost, "/api/v1/users", jsonBody(t, map[string]any{"name": strings.Repeat("a", 101), "email": "a@b.co"})), http.StatusBadRequest, "VALIDATION_FAILED", english(shared.MsgUserNameTooLong))
	requireError(t, s.do(http.MethodPost, "/api/v1/users", `{"name":"A","email":"a@b.co"}`), http.StatusInternalServerError, "INTERNAL_ERROR", english(shared.MsgInternalError))
}

// GIVEN a real server reached under another host name than the contract's servers / THEN the body is still validated
func TestRequestValidation_DoesNotDependOnTheHost(t *testing.T) {
	base := liveServer(t, newTestServer(t))
	for _, host := range []string{"api.example.com", "localhost:8080", "127.0.0.1"} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, base+"/api/v1/users", strings.NewReader(`{"name":"A","email":"a@b.co","role":1}`))
		require.NoError(t, err)
		req.Host = host
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		requireError(t, response{code: resp.StatusCode, body: body, header: resp.Header}, http.StatusBadRequest, "INVALID_BODY", english(shared.MsgUnknownField))
	}
}

// GIVEN a body over 1 MiB that would also fail the validation, with and without Content-Length
// THEN 413 PAYLOAD_TOO_LARGE: the limit is applied before the validation reads the body
func TestRequestValidation_BodyLimitComesFirst(t *testing.T) {
	s := newTestServer(t)
	body := `{"name":"` + strings.Repeat("a", 1<<20) + `","email":"a@b.co","role":1}`
	for name, chunked := range map[string]bool{"with Content-Length": false, "chunked": true} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			if chunked {
				req.ContentLength = -1
				req.TransferEncoding = []string{"chunked"}
			}
			rec := httptest.NewRecorder()

			s.e.ServeHTTP(rec, req)

			requireError(t, response{code: rec.Code, body: rec.Body.Bytes(), header: rec.Header()}, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", english(shared.MsgPayloadTooLarge))
		})
	}
}

// GIVEN the real binary in production, so without /docs / WHEN an unknown field is sent
// THEN 400: the validation does not depend on API_DOCS_ENABLED
func TestRequestValidation_RealBinaryWithoutTheDocs(t *testing.T) {
	server := startServer(t, "ENV=production")
	base := "http://" + server.addr

	assert.Equal(t, http.StatusNotFound, getStatus(t, base+"/docs"))
	r := call(t, http.MethodPost, base+"/api/v1/users", `{"name":"A","email":"a@b.co","role":1}`, nil)
	requireError(t, r.asResponse(), http.StatusBadRequest, "INVALID_BODY", english(shared.MsgUnknownField))
}

// GIVEN refused bodies carrying personal data (unknown field, too long name, too long email)
// THEN neither the answer nor the logs contain the values sent
func TestRequestValidation_NeverEchoesTheValues(t *testing.T) {
	logs := captureLogs(t)
	s := newTestServer(t)
	bodies := []string{
		jsonBody(t, map[string]any{"name": personName, "email": personEmail, "role": personName}),
		jsonBody(t, map[string]any{"name": personName + strings.Repeat("x", 100), "email": personEmail}),
		jsonBody(t, map[string]any{"name": personName, "email": strings.Repeat("x", 254) + personEmail}),
	}
	for _, body := range bodies {
		r := s.do(http.MethodPost, "/api/v1/users", body)

		require.Equal(t, http.StatusBadRequest, r.code, "body: %s", r.body)
		for _, secret := range forbiddenInLogs(body) {
			assert.NotContains(t, string(r.body), secret)
			assert.NotContains(t, logs.String(), secret)
		}
	}
}

// GIVEN Accept-Language: fr / THEN the messages of the strict validation are in French
func TestRequestValidation_MessagesInFrench(t *testing.T) {
	s := newTestServer(t)
	french := func(key shared.MessageKey) string { return shared.Message(shared.LangFrench, key) }

	r := s.sendWithLanguages(http.MethodPost, "/api/v1/users", `{"name":"A","email":"a@b.co","role":1}`, true, "fr")
	requireError(t, r, http.StatusBadRequest, "INVALID_BODY", french(shared.MsgUnknownField))
	requireLanguageHeaders(t, r, "fr")

	r = s.sendWithLanguages(http.MethodPost, "/api/v1/users", jsonBody(t, map[string]any{"name": strings.Repeat("a", 101), "email": "a@b.co"}), true, "fr")
	requireError(t, r, http.StatusBadRequest, "VALIDATION_FAILED", french(shared.MsgUserNameTooLong))
	assert.NotEqual(t, english(shared.MsgUserNameTooLong), french(shared.MsgUserNameTooLong))
}

// GIVEN a Content-Type that Echo still binds as JSON (it only cuts at the first ";") but that
// mime.ParseMediaType refuses / WHEN the body breaks the contract / THEN refused all the same:
// the media type cannot be used to skip the validation
func TestRequestValidation_CannotBeSkippedByTheContentType(t *testing.T) {
	bodies := map[string]string{
		"unknown field":  `{"name":"A","email":"a@b.co","role":1}`,
		"name too long":  jsonBody(t, map[string]any{"name": strings.Repeat("a", 101), "email": "a@b.co"}),
		"null on option": `{"name":"A","email":"a@b.co","extra":null}`,
	}
	for _, contentType := range []string{"application/json; charset", "application/json ;foo", "application/json; =x", `application/json; a="open`} {
		for name, body := range bodies {
			t.Run(contentType+" "+name, func(t *testing.T) {
				s := newTestServer(t)

				r := s.sendRaw(http.MethodPost, "/api/v1/users", contentType, body)

				require.Equal(t, http.StatusBadRequest, r.code, "body: %s", r.body)
				assert.Zero(t, s.countRows(&pgstore.UserModel{}))
			})
		}
	}
}

// GIVEN a body in a media type the contract does not describe but that Echo would bind (XML, form)
// THEN 400 INVALID_BODY and nothing stored: no bound body escapes the validation
func TestRequestValidation_RefusesOtherMediaTypes(t *testing.T) {
	cases := map[string]struct{ contentType, body string }{
		"xml":  {echo.MIMEApplicationXML, `<CreateUserRequest><Name>A</Name><Email>a@b.co</Email><Role>admin</Role></CreateUserRequest>`},
		"form": {echo.MIMEApplicationForm, "name=A&email=a%40b.co&role=admin"},
		"text": {"text/plain", `{"name":"A","email":"a@b.co"}`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := newTestServer(t)

			r := s.sendRaw(http.MethodPost, "/api/v1/users", tc.contentType, tc.body)

			requireError(t, r, http.StatusBadRequest, "INVALID_BODY", english(shared.MsgInvalidBody))
			assert.Zero(t, s.countRows(&pgstore.UserModel{}))
		})
	}
}
