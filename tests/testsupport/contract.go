package testsupport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// Contract validates HTTP exchanges against docs/specs/api/openapi.yaml and
// records which (operation, status) pairs the tests produced.
type Contract struct {
	doc    *openapi3.T
	router routers.Router

	mu       sync.Mutex
	produced map[string]bool
}

// LoadContract loads and validates the OpenAPI contract of the repository.
func LoadContract() (*Contract, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("cannot locate the contract")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "docs", "specs", "api", "openapi.yaml")

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", path, err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("invalid contract: %w", err)
	}
	// Without servers the router matches any host, such as httptest's example.com.
	doc.Servers = nil
	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("build contract router: %w", err)
	}
	return &Contract{doc: doc, router: router, produced: map[string]bool{}}, nil
}

// ValidateResponse fails the test when the request is not an operation of the
// contract, or when the response status, headers or body do not match it.
// The request itself is not validated: tests send invalid input on purpose.
func (c *Contract) ValidateResponse(t testing.TB, req *http.Request, status int, header http.Header, body []byte) {
	t.Helper()
	route, pathParams, err := c.router.FindRoute(req)
	if err != nil {
		t.Fatalf("contract: %s %s is not an operation of openapi.yaml: %v", req.Method, req.URL.Path, err)
	}

	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		},
		Status:  status,
		Header:  header,
		Body:    io.NopCloser(bytes.NewReader(body)),
		Options: &openapi3filter.Options{IncludeResponseStatus: true},
	}
	if err := openapi3filter.ValidateResponse(context.Background(), input); err != nil {
		t.Fatalf("contract: %s %s -> %d does not match openapi.yaml: %v\nbody: %s", req.Method, req.URL.Path, status, err, body)
	}

	c.mu.Lock()
	c.produced[pairKey(route.Operation.OperationID, status)] = true
	c.mu.Unlock()
}

// MissingPairs lists the (operationId, status) pairs declared by the contract
// that no validated response produced.
func (c *Contract) MissingPairs() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	var missing []string
	for _, item := range c.doc.Paths.Map() {
		for _, op := range item.Operations() {
			for status := range op.Responses.Map() {
				code, err := strconv.Atoi(status)
				if err != nil {
					missing = append(missing, fmt.Sprintf("%s %s (only numeric statuses are supported)", op.OperationID, status))
					continue
				}
				if !c.produced[pairKey(op.OperationID, code)] {
					missing = append(missing, pairKey(op.OperationID, code))
				}
			}
		}
	}
	sort.Strings(missing)
	return missing
}

func pairKey(operationID string, status int) string {
	return fmt.Sprintf("%s %d", operationID, status)
}
