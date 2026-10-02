package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v5"
)

// bodySchema est ce que le contrat dit d'un corps JSON de requête : ses
// propriétés (toute autre est refusée).
type bodySchema struct {
	properties map[string]propertyRule
}

// propertyRule reprend d'une propriété ce que le validateur contrôle : si elle
// est obligatoire, sa longueur maximale en caractères (0 : aucune) et son
// maximum numérique.
type propertyRule struct {
	required  bool
	maxLength uint64
	maximum   *float64
}

// RequestValidator applique au corps JSON des requêtes le contrat OpenAPI :
// propriétés inconnues et longueurs maximales. Les autres règles (champ
// obligatoire, vide, type, format d'email) restent à Bind et au domaine, qui
// gardent leurs codes et leurs messages ; un corps absent ou un Content-Type
// autre que JSON leur est laissé tel quel.
//
// Le schéma est lu dans le contrat plutôt que passé par un validateur
// générique : celui de kin-openapi pour OpenAPI 3.1 ne rend que du texte, qu'on
// ne peut pas associer à nos codes sans l'analyser.
type RequestValidator struct {
	schemas map[string]bodySchema
}

var pathParam = regexp.MustCompile(`\{([^}/]+)\}`)

// NewRequestValidator lit dans le contrat le schéma du corps de chaque
// opération demandée, au format « POST /api/v1/users ». Une opération absente
// du contrat, sans corps JSON ou sans refus des propriétés inconnues est une
// erreur : le serveur ne doit pas démarrer sans la validation attendue.
func NewRequestValidator(contract []byte, operations ...string) (*RequestValidator, error) {
	doc, err := openapi3.NewLoader().LoadFromData(contract)
	if err != nil {
		return nil, fmt.Errorf("load contract: %w", err)
	}
	validator := &RequestValidator{schemas: map[string]bodySchema{}}
	for _, operation := range operations {
		method, path, _ := strings.Cut(operation, " ")
		schema, err := requestBodySchema(doc, method, path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", operation, err)
		}
		// Echo désigne les paramètres de chemin par « :id », le contrat par « {id} ».
		validator.schemas[method+" "+pathParam.ReplaceAllString(path, ":$1")] = schema
	}
	return validator, nil
}

func requestBodySchema(doc *openapi3.T, method, path string) (bodySchema, error) {
	item := doc.Paths.Find(path)
	if item == nil || item.GetOperation(method) == nil {
		return bodySchema{}, errors.New("operation not in contract")
	}
	body := item.GetOperation(method).RequestBody
	if body == nil || body.Value == nil || body.Value.Content.Get("application/json") == nil {
		return bodySchema{}, errors.New("no JSON request body in contract")
	}
	schema := body.Value.Content.Get("application/json").Schema
	if schema == nil || schema.Value == nil {
		return bodySchema{}, errors.New("no request body schema in contract")
	}
	if extra := schema.Value.AdditionalProperties; extra.Has == nil || *extra.Has {
		return bodySchema{}, errors.New("request body schema must set additionalProperties: false")
	}
	if err := checkKeywords(schema.Value); err != nil {
		return bodySchema{}, err
	}
	result := bodySchema{properties: map[string]propertyRule{}}
	for name, property := range schema.Value.Properties {
		rule := propertyRule{required: slices.Contains(schema.Value.Required, name)}
		if property.Value != nil {
			rule.maximum = property.Value.Max
			if property.Value.MaxLength != nil {
				rule.maxLength = *property.Value.MaxLength
			}
		}
		result.properties[name] = rule
	}
	return result, nil
}

// tooLongMessages donne, par propriété, le message d'une valeur trop longue ;
// aboveMaximumMessages, celui d'un nombre au-dessus du maximum du contrat.
var (
	tooLongMessages = map[string]shared.MessageKey{
		"name":        shared.MsgUserNameTooLong,
		"email":       shared.MsgUserEmailTooLong,
		"title":       shared.MsgTaskTitleTooLong,
		"description": shared.MsgTaskDescriptionTooLong,
	}
	aboveMaximumMessages = map[string]shared.MessageKey{
		"user_id": shared.MsgTaskUserIDOutOfRange,
	}
)

// Middleware refuse en 400 un corps JSON qui contient une propriété inconnue
// (`INVALID_BODY`) ou une valeur plus longue que le contrat ne le permet
// (`VALIDATION_FAILED`). Il se place après BodyLimit, pour que la limite de
// taille réponde 413 avant lui.
func (v *RequestValidator) Middleware(base *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()
			schema, ok := v.schemas[req.Method+" "+c.Path()]
			// Même règle qu'Echo (BindBody) : sans corps, rien n'est lié ; sinon le
			// JSON est ce qui précède le premier « ; », espaces retirés. Un autre
			// type de contenu (XML ou formulaire, que Bind lierait aussi) n'est pas
			// décrit par le contrat : refusé, pour qu'aucun corps lié n'échappe aux
			// vérifications.
			if !ok || req.ContentLength == 0 {
				return next(c)
			}
			log := requestLogger(c, base)
			if !isJSON(req.Header.Get(echo.HeaderContentType)) {
				return respondBadRequest(c, log, shared.CodeInvalidBody, shared.MsgInvalidBody)
			}

			raw, err := io.ReadAll(req.Body)
			if err != nil {
				return respondInvalidBody(c, log, err)
			}
			req.Body = io.NopCloser(bytes.NewReader(raw))

			var fields map[string]json.RawMessage
			// Un corps illisible est laissé à Bind, qui le refuse de la même façon.
			if json.Unmarshal(raw, &fields) != nil {
				return next(c)
			}
			names := make([]string, 0, len(fields))
			for name := range fields {
				names = append(names, name)
			}
			sort.Strings(names)

			for _, name := range names {
				if _, known := schema.properties[name]; !known {
					return respondBadRequest(c, log, shared.CodeInvalidBody, shared.MsgUnknownField)
				}
			}
			for _, name := range names {
				rule := schema.properties[name]
				// null sur une propriété obligatoire est dit « vide » par le domaine ;
				// sur une propriété facultative, le contrat ne l'admet pas.
				if isNull(fields[name]) && !rule.required {
					return respondBadRequest(c, log, shared.CodeInvalidBody, shared.MsgInvalidBody)
				}
			}
			for _, name := range names {
				if key, violated := exceeds(schema.properties[name], name, fields[name]); violated {
					return respondBadRequest(c, log, shared.CodeValidationFailed, key)
				}
			}
			return next(c)
		}
	}
}

func respondBadRequest(c *echo.Context, log *slog.Logger, code string, key shared.MessageKey) error {
	log.Warn("Request body rejected", slog.String("error_code", code))
	return c.JSON(http.StatusBadRequest, shared.ErrorResponse(code, shared.Message(requestLanguage(c), key)))
}

func isJSON(contentType string) bool {
	base, _, _ := strings.Cut(contentType, ";")
	return strings.TrimSpace(base) == echo.MIMEApplicationJSON
}

func isNull(raw json.RawMessage) bool {
	return string(bytes.TrimSpace(raw)) == "null"
}

// exceeds dit si la valeur dépasse une limite du contrat et rend le message
// correspondant. Une valeur d'un autre type est laissée à Bind.
func exceeds(rule propertyRule, name string, raw json.RawMessage) (shared.MessageKey, bool) {
	if rule.maxLength > 0 {
		var text string
		//nolint:gosec // un nombre de caractères n'est jamais négatif
		if json.Unmarshal(raw, &text) == nil && uint64(utf8.RuneCountInString(text)) > rule.maxLength {
			return tooLongMessages[name], true
		}
	}
	if rule.maximum != nil && len(raw) > 0 && (raw[0] == '-' || (raw[0] >= '0' && raw[0] <= '9')) {
		if number, err := strconv.ParseFloat(string(raw), 64); err == nil && number > *rule.maximum {
			return aboveMaximumMessages[name], true
		}
	}
	return "", false
}

// Mots-clés du schéma d'un corps que le validateur ne contrôle pas lui-même :
// un schéma qui en utilise un autre que ceux-ci est refusé au démarrage, pour
// qu'un ajout au contrat ne soit jamais ignoré en silence.
var (
	// requiredGuaranteedByDomain : propriétés dont l'absence ou la vacuité est refusée par Bind et le domaine.
	requiredGuaranteedByDomain = []string{"name", "email", "title", "user_id"}
	// minLengthOneGuaranteedByDomain : propriétés dont le domaine refuse la chaîne vide.
	minLengthOneGuaranteedByDomain = []string{"name", "email", "title"}
	// minimumOneGuaranteedByDomain : propriétés dont le domaine refuse 0.
	minimumOneGuaranteedByDomain = []string{"user_id"}
	// formatsGuaranteed : formats garantis par Bind (uint32) ou par le domaine (email).
	formatsGuaranteed = []string{"email", "uint32"}
	// annotationKeywords ne contraignent aucune valeur.
	annotationKeywords = []string{"description", "title", "default", "example", "examples", "deprecated"}
)

func checkKeywords(body *openapi3.Schema) error {
	keywords, err := schemaKeywords(body)
	if err != nil {
		return err
	}
	for keyword := range keywords {
		switch {
		case keyword == "type", keyword == "properties", keyword == "additionalProperties", isAnnotation(keyword):
		case keyword == "required":
			for _, name := range body.Required {
				if !slices.Contains(requiredGuaranteedByDomain, name) {
					return fmt.Errorf("required property %q is not guaranteed by the domain", name)
				}
			}
		default:
			return fmt.Errorf("request body schema keyword %q is not applied by the validator", keyword)
		}
	}
	for name, property := range body.Properties {
		if property.Value == nil {
			return fmt.Errorf("property %q has no schema", name)
		}
		if err := checkPropertyKeywords(name, property.Value); err != nil {
			return err
		}
	}
	return nil
}

func checkPropertyKeywords(name string, property *openapi3.Schema) error {
	keywords, err := schemaKeywords(property)
	if err != nil {
		return err
	}
	for keyword, value := range keywords {
		var allowed bool
		switch keyword {
		case "type", "maxLength", "maximum":
			allowed = true
		case "format":
			allowed = slices.Contains(formatsGuaranteed, fmt.Sprint(value))
		case "minLength":
			allowed = value == float64(1) && slices.Contains(minLengthOneGuaranteedByDomain, name)
		case "minimum":
			allowed = value == float64(1) && slices.Contains(minimumOneGuaranteedByDomain, name)
		default:
			allowed = isAnnotation(keyword)
		}
		if !allowed {
			return fmt.Errorf("property %q: keyword %q with this value is not applied by the validator", name, keyword)
		}
	}
	return nil
}

func isAnnotation(keyword string) bool {
	return slices.Contains(annotationKeywords, keyword) || strings.HasPrefix(keyword, "x-")
}

// schemaKeywords liste les mots-clés présents dans le schéma, avec leur valeur.
func schemaKeywords(schema *openapi3.Schema) (map[string]any, error) {
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("read schema: %w", err)
	}
	var keywords map[string]any
	if err := json.Unmarshal(raw, &keywords); err != nil {
		return nil, fmt.Errorf("read schema: %w", err)
	}
	return keywords, nil
}
