package httpapi

import (
	"log/slog"
	"net/http"
	"regexp"

	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const (
	requestIDKey = "request_id"
	loggerKey    = "logger"
	languageKey  = "language"

	headerAcceptLanguage      = "Accept-Language"
	headerContentLanguage     = "Content-Language"
	headerContentTypeOptions  = "X-Content-Type-Options"
	contentTypeOptionsNoSniff = "nosniff"
)

// probeRoutes sont les sondes appelées en continu : réussies, elles sont
// journalisées en Debug pour ne pas noyer le journal ; leurs échecs restent visibles.
var probeRoutes = map[string]bool{"/health": true, "/ready": true}

// validRequestID borne l'identifiant repris du client : longueur et alphabet
// limités, pour qu'il ne puisse ni injecter de lignes de log ni les gonfler.
var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// RequestID attribue un identifiant à chaque requête : celui du client s'il est
// acceptable, sinon un UUID. Il est renvoyé dans X-Request-Id et ajouté aux logs
// des controllers par requestLogger.
func RequestID(base *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			id := c.Request().Header.Get(echo.HeaderXRequestID)
			if !validRequestID.MatchString(id) {
				id = uuid.NewString()
			}
			c.Response().Header().Set(echo.HeaderXRequestID, id)
			c.Set(requestIDKey, id)
			c.Set(loggerKey, base.With(slog.String("request_id", id)))
			return next(c)
		}
	}
}

// SecurityHeaders pose sur toute réponse l'en-tête qui interdit au navigateur de
// deviner le type du contenu (X-Content-Type-Options: nosniff). Il se pose avant
// d'appeler la suite et doit être le premier middleware : ainsi il couvre aussi
// les réponses qui court-circuitent la suite (préappel CORS, 404, 413, panique
// récupérée) et celles de la suite qui échouent.
func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Response().Header().Set(headerContentTypeOptions, contentTypeOptionsNoSniff)
			return next(c)
		}
	}
}

// Language choisit la langue des messages d'après Accept-Language (voir
// shared.NegotiateLanguage), la met dans le contexte et signale sur toute
// réponse, y compris celles qui court-circuitent la suite (préflight CORS, 404),
// la langue choisie et la dépendance à l'en-tête. Vary est ajouté, pas remplacé :
// le middleware CORS y met Origin.
func Language() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			lang := shared.NegotiateLanguage(c.Request().Header.Values(headerAcceptLanguage))
			c.Set(languageKey, lang)
			header := c.Response().Header()
			header.Set(headerContentLanguage, string(lang))
			header.Add(echo.HeaderVary, headerAcceptLanguage)
			return next(c)
		}
	}
}

// requestLanguage retourne la langue négociée pour la requête (anglais hors du
// middleware Language).
func requestLanguage(c *echo.Context) shared.Lang {
	if lang, ok := c.Get(languageKey).(shared.Lang); ok {
		return lang
	}
	return shared.LangEnglish
}

// requestLogger retourne le logger de la requête (avec son request_id), ou
// fallback hors du middleware RequestID.
func requestLogger(c *echo.Context, fallback *slog.Logger) *slog.Logger {
	if log, ok := c.Get(loggerKey).(*slog.Logger); ok {
		return log
	}
	return fallback
}

// AccessLog journalise une ligne par requête : méthode, chemin (sans query
// string), route, statut, durée. Jamais de corps ni de texte d'erreur.
func AccessLog(base *slog.Logger) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:    true,
		LogURIPath:   true,
		LogRoutePath: true,
		LogStatus:    true,
		LogLatency:   true,
		LogRemoteIP:  true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			level := logLevelForStatus(v.Status)
			if probeRoutes[v.RoutePath] && v.Status < http.StatusBadRequest {
				level = slog.LevelDebug
			}
			requestLogger(c, base).LogAttrs(c.Request().Context(), level, "Request",
				slog.String("method", v.Method),
				slog.String("path", v.URIPath),
				slog.String("route", v.RoutePath),
				slog.Int("status", v.Status),
				slog.Duration("latency", v.Latency),
				slog.String("remote_ip", v.RemoteIP),
			)
			return nil
		},
	})
}

// BodyLimit refuse (413) tout corps de plus de maxBytes octets, exactement, que
// la taille soit annoncée par Content-Length ou découverte en lisant : le
// middleware BodyLimit d'Echo laisse passer, sans Content-Length, un JSON
// complet qui dépasse de quelques octets. La lecture interrompue produit un
// *http.MaxBytesError, reconnu par respondInvalidBody.
func BodyLimit(maxBytes int64) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()
			if req.ContentLength > maxBytes {
				return echo.ErrStatusRequestEntityTooLarge
			}
			req.Body = http.MaxBytesReader(c.Response(), req.Body, maxBytes)
			return next(c)
		}
	}
}
