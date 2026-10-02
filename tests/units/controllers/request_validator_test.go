package controllers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/app"
	"github.com/fdanielsin/inbackend/internal/controllers/httpapi"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func committedContract(t *testing.T) string {
	t.Helper()
	contract, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "specs", "api", "openapi.yaml"))
	require.NoError(t, err)
	return string(contract)
}

// changeFirst replaces the first occurrence of anchor, which must exist, so that a contract change
// that no longer applies fails loudly instead of testing the committed contract.
func changeFirst(t *testing.T, contract, anchor, replacement string) string {
	t.Helper()
	require.Contains(t, contract, anchor)
	return strings.Replace(contract, anchor, replacement, 1)
}

const (
	createUserSchema = "    CreateUserRequest:\n      type: object\n      additionalProperties: false\n"
	userName         = "        name:\n          type: string\n          minLength: 1\n          maxLength: 100\n"
	userEmail        = "        email:\n          type: string\n          minLength: 1\n          maxLength: 254\n"
	taskUserID       = "          minimum: 1\n          maximum: 4294967295\n"
	taskRequired     = "      required: [user_id, title]\n"
	taskDescription  = "        description:\n          type: string\n          maxLength: 2000\n"
	taskCompleted    = "        completed:\n          type: boolean\n          default: false\n"
)

// GIVEN the committed contract / WHEN the validator of the three operations is built / THEN no error
func TestRequestValidator_LoadsTheCommittedContract(t *testing.T) {
	_, err := httpapi.NewRequestValidator([]byte(committedContract(t)), app.ValidatedOperations...)

	require.NoError(t, err)
}

// GIVEN a contract the validator could not enforce in full / WHEN it is built / THEN an error, so that
// the server does not start with a weaker validation than the contract promises (fail closed)
func TestRequestValidator_RefusesWhatItCannotEnforce(t *testing.T) {
	cases := map[string]struct {
		change     func(t *testing.T, contract string) string
		operations []string
	}{
		"operation absent from the contract": {operations: append([]string{"PATCH /api/v1/users"}, app.ValidatedOperations...)},
		"operation without a JSON body":      {operations: []string{"GET /health"}},
		"unreadable contract": {change: func(_ *testing.T, _ string) string {
			return "openapi: [3.1"
		}},
		"additionalProperties missing": {change: func(t *testing.T, c string) string {
			return changeFirst(t, c, createUserSchema, "    CreateUserRequest:\n      type: object\n")
		}},
		"additionalProperties true": {change: func(t *testing.T, c string) string {
			return changeFirst(t, c, createUserSchema, "    CreateUserRequest:\n      type: object\n      additionalProperties: true\n")
		}},
		"minProperties on a body": {change: func(t *testing.T, c string) string {
			return changeFirst(t, c, createUserSchema, createUserSchema+"      minProperties: 1\n")
		}},
		"pattern on name": {change: func(t *testing.T, c string) string {
			return changeFirst(t, c, userName, userName+"          pattern: '^[a-z]+$'\n")
		}},
		"enum on name": {change: func(t *testing.T, c string) string {
			return changeFirst(t, c, userName, userName+"          enum: [Ada]\n")
		}},
		"minLength 2 on name": {change: func(t *testing.T, c string) string {
			return changeFirst(t, c, userName, strings.Replace(userName, "minLength: 1", "minLength: 2", 1))
		}},
		"format date on email": {change: func(t *testing.T, c string) string {
			return changeFirst(t, c, userEmail, userEmail+"          format: date\n")
		}},
		"minimum 5 on user_id": {change: func(t *testing.T, c string) string {
			return changeFirst(t, c, taskUserID, strings.Replace(taskUserID, "minimum: 1", "minimum: 5", 1))
		}},
		"minLength 1 on description": {change: func(t *testing.T, c string) string {
			return changeFirst(t, c, taskDescription, taskDescription+"          minLength: 1\n")
		}},
		"minimum on completed": {change: func(t *testing.T, c string) string {
			return changeFirst(t, c, taskCompleted, taskCompleted+"          minimum: 1\n")
		}},
		"description required": {change: func(t *testing.T, c string) string {
			return changeFirst(t, c, taskRequired, "      required: [user_id, title, description]\n")
		}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			contract := committedContract(t)
			if tc.change != nil {
				contract = tc.change(t, contract)
			}
			operations := tc.operations
			if operations == nil {
				operations = app.ValidatedOperations
			}

			validator, err := httpapi.NewRequestValidator([]byte(contract), operations...)

			require.Error(t, err)
			assert.Nil(t, validator)
		})
	}
}

// GIVEN annotations that change no rule (example, deprecated, x-*) / WHEN the validator is built / THEN no error
func TestRequestValidator_AcceptsAnnotations(t *testing.T) {
	contract := changeFirst(t, committedContract(t), userName, userName+"          example: Ada\n          deprecated: false\n          x-note: kept\n")

	_, err := httpapi.NewRequestValidator([]byte(contract), app.ValidatedOperations...)

	require.NoError(t, err)
}

// GIVEN the contract / THEN app.ValidatedOperations lists exactly its operations with a request body:
// a new operation with a body cannot be added without its validation
func TestRequestValidator_CoversEveryOperationWithABody(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromData([]byte(committedContract(t)))
	require.NoError(t, err)

	var withBody []string
	for path, item := range doc.Paths.Map() {
		for method, operation := range item.Operations() {
			if operation.RequestBody != nil {
				withBody = append(withBody, method+" "+path)
			}
		}
	}

	assert.ElementsMatch(t, withBody, app.ValidatedOperations)
}
