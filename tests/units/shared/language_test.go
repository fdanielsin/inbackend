package shared

import (
	"fmt"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GIVEN Accept-Language values (RFC 9110 §12.5.4, RFC 4647 basic filtering)
// THEN the served language with the highest weight wins (the first one on a tie), English otherwise
func TestNegotiateLanguage_FollowsTheWeights(t *testing.T) {
	cases := []struct {
		lines []string
		want  shared.Lang
	}{
		{nil, shared.LangEnglish},
		{[]string{""}, shared.LangEnglish},
		{[]string{"fr"}, shared.LangFrench},
		{[]string{"fr-FR"}, shared.LangFrench},
		{[]string{"fr-CA"}, shared.LangFrench},
		{[]string{"FR"}, shared.LangFrench},
		{[]string{"Fr-ch"}, shared.LangFrench},
		{[]string{"en"}, shared.LangEnglish},
		{[]string{"en-US"}, shared.LangEnglish},
		{[]string{"de"}, shared.LangEnglish},
		{[]string{"de;q=1, fr;q=0.5"}, shared.LangFrench},
		{[]string{"fr;q=0.5, en;q=0.9"}, shared.LangEnglish},
		{[]string{"en;q=0.5, fr;q=0.9"}, shared.LangFrench},
		{[]string{"fr, en"}, shared.LangFrench},
		{[]string{"en, fr"}, shared.LangEnglish},
		{[]string{"fr;q=0.8, en;q=0.8"}, shared.LangFrench},
		{[]string{"en;q=0.8, fr;q=0.8"}, shared.LangEnglish},
		{[]string{"fr;q=0"}, shared.LangEnglish},
		{[]string{"fr;q=0, en;q=0"}, shared.LangEnglish},
		{[]string{"fr;q=0.001"}, shared.LangFrench},
		{[]string{"fr;q=1"}, shared.LangFrench},
		{[]string{"fr;q=1.000"}, shared.LangFrench},
		{[]string{"*"}, shared.LangEnglish},
		{[]string{"*;q=1, fr;q=0.1"}, shared.LangFrench},
		{[]string{"!!!"}, shared.LangEnglish},
		{[]string{"fr;q=abc"}, shared.LangEnglish},
		{[]string{"fr;q=1.5"}, shared.LangEnglish},
		{[]string{"fr_FR"}, shared.LangEnglish},
		{[]string{",,,"}, shared.LangEnglish},
		{[]string{" "}, shared.LangEnglish},
		{[]string{"de", "fr;q=0.5"}, shared.LangFrench},
		{[]string{"en", "fr"}, shared.LangEnglish},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%q", c.lines), func(t *testing.T) {
			assert.Equal(t, c.want, shared.NegotiateLanguage(c.lines))
		})
	}
}

// GIVEN a header that ends with "fr" / WHEN its total length is exactly 1024 bytes, then 1025
// THEN it is negotiated, then ignored (English): the limit bounds the work on hostile headers
func TestNegotiateLanguage_LengthLimit(t *testing.T) {
	// valid ranges ("de;q=0.1, ", 10 bytes each), spaces after the last comma (optional whitespace), then "fr"
	withLength := func(n int) string {
		const item = "de;q=0.1, "
		rest := n - len("fr")
		header := strings.Repeat(item, rest/len(item)) + strings.Repeat(" ", rest%len(item)) + "fr"
		require.Len(t, header, n)
		return header
	}

	assert.Equal(t, shared.LangFrench, shared.NegotiateLanguage([]string{withLength(1024)}))
	assert.Equal(t, shared.LangEnglish, shared.NegotiateLanguage([]string{withLength(1025)}))
	assert.Equal(t, shared.LangEnglish, shared.NegotiateLanguage([]string{withLength(600), withLength(600)}),
		"the limit applies to all the lines together")
}

// every message key, as declared by the constants
var allMessageKeys = []shared.MessageKey{
	shared.MsgInternalError, shared.MsgUserNotFound, shared.MsgUserNameEmpty, shared.MsgUserEmailEmpty,
	shared.MsgUserEmailInvalid, shared.MsgEmailTaken, shared.MsgTaskNotFound, shared.MsgTaskTitleEmpty,
	shared.MsgTaskUserIDInvalid, shared.MsgInvalidBody, shared.MsgPayloadTooLarge, shared.MsgInvalidUserID,
	shared.MsgInvalidTaskID, shared.MsgRouteNotFound, shared.MsgMethodNotAllowed, shared.MsgInvalidRequest,
	shared.MsgDatabaseDown, shared.MsgUserCreated, shared.MsgTaskCreated, shared.MsgTaskUpdated,
	shared.MsgTaskDeleted, shared.MsgServiceHealthy, shared.MsgServiceReady,
	shared.MsgUserNameTooLong, shared.MsgUserEmailTooLong, shared.MsgTaskTitleTooLong,
	shared.MsgTaskDescriptionTooLong, shared.MsgTaskUserIDOutOfRange, shared.MsgUnknownField,
	shared.MsgTaskCompletedFilter,
}

// GIVEN the catalogue / THEN it holds exactly the declared keys, each with a non-empty text per language,
// French different from English
func TestMessages_CatalogueIsComplete(t *testing.T) {
	require.ElementsMatch(t, allMessageKeys, shared.MessageKeys(), "no key without text, no orphan text")
	require.ElementsMatch(t, []shared.Lang{shared.LangEnglish, shared.LangFrench}, shared.Languages())

	for _, key := range allMessageKeys {
		english := shared.Message(shared.LangEnglish, key)
		french := shared.Message(shared.LangFrench, key)
		assert.NotEmpty(t, english, key)
		assert.NotEmpty(t, french, key)
		assert.NotEqual(t, string(key), english, "%s has no English text", key)
		assert.NotEqual(t, string(key), french, "%s has no French text", key)
		assert.NotEqual(t, english, french, "%s is not translated", key)
	}
}

// GIVEN the English texts / THEN they are exactly the messages of the API before translation (English stays the default)
func TestMessages_EnglishTextsAreUnchanged(t *testing.T) {
	want := map[shared.MessageKey]string{
		shared.MsgInternalError: "Internal server error", shared.MsgUserNotFound: "User not found",
		shared.MsgUserNameEmpty: "User name cannot be empty", shared.MsgUserEmailEmpty: "User email cannot be empty",
		shared.MsgUserEmailInvalid: "User email is invalid", shared.MsgEmailTaken: "Email already in use",
		shared.MsgTaskNotFound: "Task not found", shared.MsgTaskTitleEmpty: "Task title cannot be empty",
		shared.MsgTaskUserIDInvalid: "Task user_id cannot be empty", shared.MsgInvalidBody: "Invalid request body",
		shared.MsgPayloadTooLarge: "Request body too large", shared.MsgInvalidUserID: "Invalid user ID",
		shared.MsgInvalidTaskID: "Invalid task ID", shared.MsgRouteNotFound: "Route not found",
		shared.MsgMethodNotAllowed: "Method not allowed", shared.MsgInvalidRequest: "Invalid request",
		shared.MsgDatabaseDown: "Database unavailable", shared.MsgUserCreated: "User created",
		shared.MsgTaskCreated: "Task created", shared.MsgTaskUpdated: "Task updated", shared.MsgTaskDeleted: "Task deleted",
		shared.MsgServiceHealthy: "Service is healthy", shared.MsgServiceReady: "Service is ready",
		shared.MsgUserNameTooLong:        "User name cannot exceed 100 characters",
		shared.MsgUserEmailTooLong:       "User email cannot exceed 254 characters",
		shared.MsgTaskTitleTooLong:       "Task title cannot exceed 200 characters",
		shared.MsgTaskDescriptionTooLong: "Task description cannot exceed 2000 characters",
		shared.MsgTaskUserIDOutOfRange:   "Task user_id is out of range",
		shared.MsgUnknownField:           "Request body contains an unknown field",
		shared.MsgTaskCompletedFilter:    "Query parameter completed must be true or false",
	}
	require.Len(t, want, len(allMessageKeys))
	for key, text := range want {
		assert.Equal(t, text, shared.Message(shared.LangEnglish, key), key)
	}
}

// GIVEN the French texts / THEN each key has its own meaning (two swapped texts would still be "translated")
func TestMessages_FrenchTexts(t *testing.T) {
	want := map[shared.MessageKey]string{
		shared.MsgInternalError: "Erreur interne du serveur", shared.MsgUserNotFound: "Utilisateur introuvable",
		shared.MsgUserNameEmpty:    "Le nom de l'utilisateur ne peut pas être vide",
		shared.MsgUserEmailEmpty:   "L'adresse e-mail de l'utilisateur ne peut pas être vide",
		shared.MsgUserEmailInvalid: "L'adresse e-mail de l'utilisateur est invalide",
		shared.MsgEmailTaken:       "Cette adresse e-mail est déjà utilisée",
		shared.MsgTaskNotFound:     "Tâche introuvable", shared.MsgTaskTitleEmpty: "Le titre de la tâche ne peut pas être vide",
		shared.MsgTaskUserIDInvalid: "Le user_id de la tâche ne peut pas être vide",
		shared.MsgInvalidBody:       "Corps de requête invalide", shared.MsgPayloadTooLarge: "Corps de requête trop volumineux",
		shared.MsgInvalidUserID: "Identifiant d'utilisateur invalide", shared.MsgInvalidTaskID: "Identifiant de tâche invalide",
		shared.MsgRouteNotFound: "Route introuvable", shared.MsgMethodNotAllowed: "Méthode non autorisée",
		shared.MsgInvalidRequest: "Requête invalide", shared.MsgDatabaseDown: "Base de données indisponible",
		shared.MsgUserCreated: "Utilisateur créé", shared.MsgTaskCreated: "Tâche créée",
		shared.MsgTaskUpdated: "Tâche mise à jour", shared.MsgTaskDeleted: "Tâche supprimée",
		shared.MsgServiceHealthy: "Le service est opérationnel", shared.MsgServiceReady: "Le service est prêt",
		shared.MsgUserNameTooLong:        "Le nom de l'utilisateur ne peut pas dépasser 100 caractères",
		shared.MsgUserEmailTooLong:       "L'adresse e-mail de l'utilisateur ne peut pas dépasser 254 caractères",
		shared.MsgTaskTitleTooLong:       "Le titre de la tâche ne peut pas dépasser 200 caractères",
		shared.MsgTaskDescriptionTooLong: "La description de la tâche ne peut pas dépasser 2000 caractères",
		shared.MsgTaskUserIDOutOfRange:   "Le user_id de la tâche est hors limites",
		shared.MsgUnknownField:           "Le corps de la requête contient un champ inconnu",
		shared.MsgTaskCompletedFilter:    "Le paramètre de requête completed doit valoir true ou false",
	}
	require.Len(t, want, len(allMessageKeys))
	for key, text := range want {
		assert.Equal(t, text, shared.Message(shared.LangFrench, key), key)
	}
}

// GIVEN an unknown language or key / THEN English, or the key itself when it has no text at all
func TestMessages_UnknownLanguageOrKey(t *testing.T) {
	assert.Equal(t, "User not found", shared.Message(shared.Lang("de"), shared.MsgUserNotFound))
	assert.Equal(t, "User not found", shared.Message(shared.Lang(""), shared.MsgUserNotFound))
	assert.Equal(t, "NoSuchKey", shared.Message(shared.LangFrench, shared.MessageKey("NoSuchKey")))
}
