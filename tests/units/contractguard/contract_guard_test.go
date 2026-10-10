package contractguard

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/tests/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingT is a testing.TB whose Fatalf records the failure and stops the goroutine, like the real one.
type recordingT struct {
	testing.TB
	failure string
}

func (r *recordingT) Helper() {}

func (r *recordingT) Fatalf(format string, args ...any) {
	r.failure = fmt.Sprintf(format, args...)
	runtime.Goexit()
}

// validate runs ValidateResponse for an error answer of the server and returns the failure it raised ("" when none).
func validate(t *testing.T, method, path, requestBody string, status int, body string) string {
	t.Helper()
	contract, err := testsupport.LoadContract()
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, strings.NewReader(requestBody))
	header := http.Header{"Content-Type": {"application/json"}, "Content-Language": {"en"}, "Vary": {"Accept-Language"}}
	recorder := &recordingT{TB: t}

	done := make(chan struct{})
	go func() {
		defer close(done)
		contract.ValidateResponse(recorder, req, status, header, []byte(body))
	}()
	<-done
	return recorder.failure
}

const (
	invalidIDBody   = `{"success":false,"error":"Invalid task ID","error_code":"INVALID_ID"}`
	invalidBodyBody = `{"success":false,"error":"Invalid request body","error_code":"INVALID_BODY"}`
)

// RULE: TASK-R09 (sad path)
// GIVEN updateTask answering 400 INVALID_ID for a non numeric id / WHEN the answer is checked against the contract
// THEN it is accepted: the description of the declared 400 names INVALID_ID
func TestContractGuard_UpdateTaskInvalidIDIsDescribed(t *testing.T) {
	failure := validate(t, http.MethodPut, "/api/v1/tasks/abc", `{"completed":true}`, http.StatusBadRequest, invalidIDBody)

	assert.Empty(t, failure)
}

// RULE: TASK-R09 (sad path)
// GIVEN the other codes of the updateTask 400 / THEN they stay described as well
func TestContractGuard_UpdateTaskBodyErrorsAreDescribed(t *testing.T) {
	for _, code := range []string{"INVALID_BODY", "VALIDATION_FAILED"} {
		t.Run(code, func(t *testing.T) {
			body := fmt.Sprintf(`{"success":false,"error":"x","error_code":%q}`, code)

			assert.Empty(t, validate(t, http.MethodPut, "/api/v1/tasks/1", `{"completed":true}`, http.StatusBadRequest, body))
		})
	}
}

// RULE: TASK-R09 (sad path)
// GIVEN an error code that the contract never names for this response / THEN the check fails and says which code
func TestContractGuard_UnnamedErrorCodeFails(t *testing.T) {
	body := `{"success":false,"error":"Email already in use","error_code":"EMAIL_TAKEN"}`

	failure := validate(t, http.MethodPut, "/api/v1/tasks/1", `{"completed":true}`, http.StatusBadRequest, body)

	assert.Contains(t, failure, "EMAIL_TAKEN")
	assert.Contains(t, failure, "does not name")
}

// RULE: TASK-R09 (sad path)
// GIVEN a code that is described for another status only / THEN it is not accepted on this one
func TestContractGuard_CodeIsCheckedAgainstTheDeclaredStatus(t *testing.T) {
	failure := validate(t, http.MethodPut, "/api/v1/tasks/1", `{"completed":true}`, http.StatusNotFound,
		`{"success":false,"error":"x","error_code":"INVALID_ID"}`)

	assert.Contains(t, failure, "INVALID_ID")
	assert.Contains(t, failure, "does not name")
}

// RULE: TASK-R09 (sad path)
// GIVEN a success answer or an answer without body / THEN the guard does not apply
func TestContractGuard_IgnoresAnswersWithoutErrorCode(t *testing.T) {
	assert.Empty(t, validate(t, http.MethodGet, "/health", "", http.StatusOK,
		`{"success":true,"message":"Service is healthy","data":{"id":"b9db8dbf-af96-40ed-84de-cc1268d1a2a1","status":"ok","version":"1.2.0"}}`))
}
