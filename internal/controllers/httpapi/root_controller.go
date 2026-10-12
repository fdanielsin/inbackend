package httpapi

import (
	"net/http"

	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/labstack/echo/v5"
)

// RootController gère la racine de l'API.
type RootController struct {
	docsEnabled bool
}

// NewRootController crée le controller ; docsEnabled dit si /docs et /openapi.yaml existent.
func NewRootController(docsEnabled bool) *RootController {
	return &RootController{docsEnabled: docsEnabled}
}

// apiSummary est le sommaire de la racine : rien d'autre que ce qui est déjà public.
type apiSummary struct {
	Name    string          `json:"name"`
	Version string          `json:"version"`
	Links   apiSummaryLinks `json:"links"`
}

// apiSummaryLinks sont des chemins depuis la racine, jamais construits à partir de la requête.
type apiSummaryLinks struct {
	Docs    string `json:"docs,omitempty"`
	OpenAPI string `json:"openapi,omitempty"`
	Health  string `json:"health"`
	Ready   string `json:"ready"`
}

// Summary gère GET / : accueille et oriente, sans lire la base.
func (rc *RootController) Summary(c *echo.Context) error {
	forbidCaching(c)

	links := apiSummaryLinks{Health: "/health", Ready: "/ready"}
	if rc.docsEnabled {
		links.Docs = "/docs"
		links.OpenAPI = "/openapi.yaml"
	}

	return c.JSON(http.StatusOK, shared.SuccessResponse(apiSummary{
		Name:    shared.APIName,
		Version: shared.APIVersion,
		Links:   links,
	}, shared.Message(requestLanguage(c), shared.MsgWelcome)))
}
