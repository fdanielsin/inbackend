package controllers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fdanielsin/inbackend/internal/controllers/httpapi"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const noSniff = "X-Content-Type-Options"

func serve(e *echo.Echo, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestSecurityHeaders(t *testing.T) {
	t.Run("SEC-R04: an answer of the next handler carries nosniff once (happy path)", func(t *testing.T) {
		e := echo.New()
		e.Use(httpapi.SecurityHeaders())
		e.GET("/ok", func(c *echo.Context) error { return c.String(http.StatusOK, "ok") })

		rec := serve(e, http.MethodGet, "/ok")

		assert.Equal(t, []string{"nosniff"}, rec.Header().Values(noSniff))
	})

	t.Run("SEC-R04: HEAD carries nosniff like GET (happy path)", func(t *testing.T) {
		e := echo.New()
		e.Use(httpapi.SecurityHeaders())
		e.Match([]string{http.MethodGet, http.MethodHead}, "/ok", func(c *echo.Context) error { return c.String(http.StatusOK, "ok") })

		rec := serve(e, http.MethodHead, "/ok")

		assert.Equal(t, []string{"nosniff"}, rec.Header().Values(noSniff))
	})

	t.Run("SEC-R04: a handler that fails still answers with nosniff (sad path)", func(t *testing.T) {
		e := echo.New()
		e.Use(httpapi.SecurityHeaders())
		e.GET("/fail", func(*echo.Context) error { return errors.New("boom") })

		rec := serve(e, http.MethodGet, "/fail")

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, []string{"nosniff"}, rec.Header().Values(noSniff))
	})

	t.Run("SEC-R04: an unknown route and a wrong method carry nosniff (sad path)", func(t *testing.T) {
		e := echo.New()
		e.Use(httpapi.SecurityHeaders())
		e.GET("/only-get", func(c *echo.Context) error { return c.String(http.StatusOK, "ok") })

		unknown := serve(e, http.MethodGet, "/nope")
		wrongMethod := serve(e, http.MethodPost, "/only-get")

		require.Equal(t, http.StatusNotFound, unknown.Code)
		require.Equal(t, http.StatusMethodNotAllowed, wrongMethod.Code)
		assert.Equal(t, []string{"nosniff"}, unknown.Header().Values(noSniff))
		assert.Equal(t, []string{"nosniff"}, wrongMethod.Header().Values(noSniff))
	})

	t.Run("SEC-R04: a later middleware that answers without calling the handler cannot lose it (limit)", func(t *testing.T) {
		e := echo.New()
		e.Use(httpapi.SecurityHeaders())
		e.Use(func(echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error { return c.NoContent(http.StatusNoContent) }
		})
		e.GET("/never", func(*echo.Context) error { panic("not reached") })

		rec := serve(e, http.MethodOptions, "/never")

		require.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, []string{"nosniff"}, rec.Header().Values(noSniff))
	})

	t.Run("SEC-R04: a recovered panic still carries nosniff (exception)", func(t *testing.T) {
		e := echo.New()
		e.Use(httpapi.SecurityHeaders())
		e.Use(middleware.Recover())
		e.GET("/panic", func(*echo.Context) error { panic("boom") })

		rec := serve(e, http.MethodGet, "/panic")

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, []string{"nosniff"}, rec.Header().Values(noSniff))
	})

	t.Run("SEC-R04: a value set earlier is replaced, not doubled (boundary)", func(t *testing.T) {
		e := echo.New()
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
				c.Response().Header().Set(noSniff, "sniff")
				c.Response().Header().Add(noSniff, "again")
				return next(c)
			}
		})
		e.Use(httpapi.SecurityHeaders())
		e.GET("/ok", func(c *echo.Context) error { return c.String(http.StatusOK, "ok") })

		rec := serve(e, http.MethodGet, "/ok")

		assert.Equal(t, []string{"nosniff"}, rec.Header().Values(noSniff))
	})

	t.Run("SEC-R04: the other headers and the body are left alone (limit)", func(t *testing.T) {
		e := echo.New()
		e.Use(httpapi.SecurityHeaders())
		e.GET("/json", func(c *echo.Context) error {
			c.Response().Header().Set("X-Other", "kept")
			return c.JSON(http.StatusOK, map[string]string{"a": "b"})
		})

		rec := serve(e, http.MethodGet, "/json")

		assert.Equal(t, "kept", rec.Header().Get("X-Other"))
		assert.Contains(t, rec.Header().Get(echo.HeaderContentType), "application/json")
		assert.JSONEq(t, `{"a":"b"}`, rec.Body.String())
	})
}
