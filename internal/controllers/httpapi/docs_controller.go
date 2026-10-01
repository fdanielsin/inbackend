package httpapi

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/labstack/echo/v5"
)

// Swagger UI est chargé depuis un CDN à une version exacte, avec l'intégrité
// de chaque fichier : un fichier modifié par le CDN n'est pas exécuté. Changer
// de version impose de recalculer les deux empreintes sha384.
const (
	swaggerUIVersion = "5.33.1"
	swaggerUIBase    = "https://cdn.jsdelivr.net/npm/swagger-ui-dist@" + swaggerUIVersion
	swaggerUICSSHash = "sha384-Ov4/wv3j2bmct8cDc5X4ngJZohVPzEmc6uDPH8WeljUxO5vtoykvMEfbu9Vh6RaW"
	swaggerUIJSHash  = "sha384-ZPehFMQommnnuaZ4rpxgkgTT2DKFVp4hZC/7pLit+9Lek9T1YGSo23eHFbvNkXkw"

	swaggerInitScript = `window.onload = function () {
  window.ui = SwaggerUIBundle({
    url: "/openapi.yaml",
    dom_id: "#swagger-ui",
    presets: [SwaggerUIBundle.presets.apis],
    layout: "BaseLayout",
    validatorUrl: null,
  });
};`

	swaggerPage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>InDrive Backend API</title>
<link rel="stylesheet" href="%[1]s/swagger-ui.css" integrity="%[2]s" crossorigin="anonymous">
</head>
<body>
<div id="swagger-ui"></div>
<script src="%[1]s/swagger-ui-bundle.js" integrity="%[3]s" crossorigin="anonymous"></script>
<script>%[4]s</script>
</body>
</html>
`
)

// DocsController sert le contrat OpenAPI et la page Swagger UI.
type DocsController struct {
	spec []byte
	page string
	csp  string
}

// NewDocsController crée le controller pour le contrat donné. Le hash du script
// d'initialisation en ligne est calculé ici pour la politique CSP, afin de ne
// pas avoir à le maintenir à la main.
func NewDocsController(spec []byte) *DocsController {
	scriptHash := sha256.Sum256([]byte(swaggerInitScript))
	return &DocsController{
		spec: spec,
		page: fmt.Sprintf(swaggerPage, swaggerUIBase, swaggerUICSSHash, swaggerUIJSHash, swaggerInitScript),
		// style-src autorise 'unsafe-inline' : Swagger UI pose des styles en ligne.
		csp: "default-src 'none'; script-src https://cdn.jsdelivr.net 'sha256-" +
			base64.StdEncoding.EncodeToString(scriptHash[:]) + "'; " +
			"style-src https://cdn.jsdelivr.net 'unsafe-inline'; img-src 'self' data:; " +
			"connect-src 'self'; frame-ancestors 'none'; base-uri 'none'",
	}
}

// OpenAPI gère GET /openapi.yaml.
func (dc *DocsController) OpenAPI(c *echo.Context) error {
	setDocsHeaders(c, shared.LangFrench)
	return c.Blob(http.StatusOK, "application/yaml", dc.spec)
}

// SwaggerUI gère GET /docs.
func (dc *DocsController) SwaggerUI(c *echo.Context) error {
	setDocsHeaders(c, shared.LangEnglish)
	c.Response().Header().Set("Content-Security-Policy", dc.csp)
	return c.HTML(http.StatusOK, dc.page)
}

// setDocsHeaders pose Content-Language à la langue réelle du contenu (le contrat
// est écrit en français, la page Swagger UI est en anglais), sans tenir compte
// d'Accept-Language : ces textes ne sont pas dans le catalogue.
func setDocsHeaders(c *echo.Context, contentLanguage shared.Lang) {
	header := c.Response().Header()
	header.Set(headerContentLanguage, string(contentLanguage))
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set(echo.HeaderCacheControl, "no-cache")
}
