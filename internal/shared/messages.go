package shared

// Lang est une langue des messages de l'API.
type Lang string

// Langues prises en charge ; l'anglais est la langue par défaut.
const (
	LangEnglish Lang = "en"
	LangFrench  Lang = "fr"
)

// MessageKey identifie un texte de réponse dans le catalogue.
type MessageKey string

// Clés du catalogue : textes d'erreur puis textes de succès.
const (
	MsgInternalError          MessageKey = "InternalError"
	MsgUserNotFound           MessageKey = "UserNotFound"
	MsgUserNameEmpty          MessageKey = "UserNameEmpty"
	MsgUserEmailEmpty         MessageKey = "UserEmailEmpty"
	MsgUserEmailInvalid       MessageKey = "UserEmailInvalid"
	MsgUserNameTooLong        MessageKey = "UserNameTooLong"
	MsgUserEmailTooLong       MessageKey = "UserEmailTooLong"
	MsgEmailTaken             MessageKey = "EmailTaken"
	MsgTaskNotFound           MessageKey = "TaskNotFound"
	MsgTaskTitleEmpty         MessageKey = "TaskTitleEmpty"
	MsgTaskTitleTooLong       MessageKey = "TaskTitleTooLong"
	MsgTaskDescriptionTooLong MessageKey = "TaskDescriptionTooLong"
	MsgTaskUserIDOutOfRange   MessageKey = "TaskUserIDOutOfRange"
	MsgTaskUserIDInvalid      MessageKey = "TaskUserIDInvalid"
	MsgInvalidBody            MessageKey = "InvalidBody"
	MsgUnknownField           MessageKey = "UnknownField"
	MsgPayloadTooLarge        MessageKey = "PayloadTooLarge"
	MsgInvalidUserID          MessageKey = "InvalidUserID"
	MsgInvalidTaskID          MessageKey = "InvalidTaskID"
	MsgTaskCompletedFilter    MessageKey = "TaskCompletedFilter"
	MsgRouteNotFound          MessageKey = "RouteNotFound"
	MsgMethodNotAllowed       MessageKey = "MethodNotAllowed"
	MsgInvalidRequest         MessageKey = "InvalidRequest"
	MsgDatabaseDown           MessageKey = "DatabaseUnavailable"
	MsgUserCreated            MessageKey = "UserCreated"
	MsgTaskCreated            MessageKey = "TaskCreated"
	MsgTaskUpdated            MessageKey = "TaskUpdated"
	MsgTaskDeleted            MessageKey = "TaskDeleted"
	MsgServiceHealthy         MessageKey = "ServiceHealthy"
	MsgServiceReady           MessageKey = "ServiceReady"
)

var catalog = map[MessageKey]map[Lang]string{
	MsgInternalError:          {LangEnglish: "Internal server error", LangFrench: "Erreur interne du serveur"},
	MsgUserNotFound:           {LangEnglish: "User not found", LangFrench: "Utilisateur introuvable"},
	MsgUserNameEmpty:          {LangEnglish: "User name cannot be empty", LangFrench: "Le nom de l'utilisateur ne peut pas être vide"},
	MsgUserEmailEmpty:         {LangEnglish: "User email cannot be empty", LangFrench: "L'adresse e-mail de l'utilisateur ne peut pas être vide"},
	MsgUserEmailInvalid:       {LangEnglish: "User email is invalid", LangFrench: "L'adresse e-mail de l'utilisateur est invalide"},
	MsgUserNameTooLong:        {LangEnglish: "User name cannot exceed 100 characters", LangFrench: "Le nom de l'utilisateur ne peut pas dépasser 100 caractères"},
	MsgUserEmailTooLong:       {LangEnglish: "User email cannot exceed 254 characters", LangFrench: "L'adresse e-mail de l'utilisateur ne peut pas dépasser 254 caractères"},
	MsgEmailTaken:             {LangEnglish: "Email already in use", LangFrench: "Cette adresse e-mail est déjà utilisée"},
	MsgTaskNotFound:           {LangEnglish: "Task not found", LangFrench: "Tâche introuvable"},
	MsgTaskTitleEmpty:         {LangEnglish: "Task title cannot be empty", LangFrench: "Le titre de la tâche ne peut pas être vide"},
	MsgTaskTitleTooLong:       {LangEnglish: "Task title cannot exceed 200 characters", LangFrench: "Le titre de la tâche ne peut pas dépasser 200 caractères"},
	MsgTaskDescriptionTooLong: {LangEnglish: "Task description cannot exceed 2000 characters", LangFrench: "La description de la tâche ne peut pas dépasser 2000 caractères"},
	MsgTaskUserIDOutOfRange:   {LangEnglish: "Task user_id is out of range", LangFrench: "Le user_id de la tâche est hors limites"},
	MsgTaskUserIDInvalid:      {LangEnglish: "Task user_id cannot be empty", LangFrench: "Le user_id de la tâche ne peut pas être vide"},
	MsgInvalidBody:            {LangEnglish: "Invalid request body", LangFrench: "Corps de requête invalide"},
	MsgUnknownField:           {LangEnglish: "Request body contains an unknown field", LangFrench: "Le corps de la requête contient un champ inconnu"},
	MsgPayloadTooLarge:        {LangEnglish: "Request body too large", LangFrench: "Corps de requête trop volumineux"},
	MsgInvalidUserID:          {LangEnglish: "Invalid user ID", LangFrench: "Identifiant d'utilisateur invalide"},
	MsgInvalidTaskID:          {LangEnglish: "Invalid task ID", LangFrench: "Identifiant de tâche invalide"},
	MsgTaskCompletedFilter:    {LangEnglish: "Query parameter completed must be true or false", LangFrench: "Le paramètre de requête completed doit valoir true ou false"},
	MsgRouteNotFound:          {LangEnglish: "Route not found", LangFrench: "Route introuvable"},
	MsgMethodNotAllowed:       {LangEnglish: "Method not allowed", LangFrench: "Méthode non autorisée"},
	MsgInvalidRequest:         {LangEnglish: "Invalid request", LangFrench: "Requête invalide"},
	MsgDatabaseDown:           {LangEnglish: "Database unavailable", LangFrench: "Base de données indisponible"},
	MsgUserCreated:            {LangEnglish: "User created", LangFrench: "Utilisateur créé"},
	MsgTaskCreated:            {LangEnglish: "Task created", LangFrench: "Tâche créée"},
	MsgTaskUpdated:            {LangEnglish: "Task updated", LangFrench: "Tâche mise à jour"},
	MsgTaskDeleted:            {LangEnglish: "Task deleted", LangFrench: "Tâche supprimée"},
	MsgServiceHealthy:         {LangEnglish: "Service is healthy", LangFrench: "Le service est opérationnel"},
	MsgServiceReady:           {LangEnglish: "Service is ready", LangFrench: "Le service est prêt"},
}

// Languages retourne les langues prises en charge.
func Languages() []Lang { return []Lang{LangEnglish, LangFrench} }

// MessageKeys retourne toutes les clés du catalogue.
func MessageKeys() []MessageKey {
	keys := make([]MessageKey, 0, len(catalog))
	for key := range catalog {
		keys = append(keys, key)
	}
	return keys
}

// Message retourne le texte d'une clé dans la langue donnée ; une langue ou une
// clé inconnue donne le texte anglais, ou la clé elle-même si elle est absente.
func Message(lang Lang, key MessageKey) string {
	texts, ok := catalog[key]
	if !ok {
		return string(key)
	}
	if text, ok := texts[lang]; ok {
		return text
	}
	return texts[LangEnglish]
}
