package shared

// APIResponse est la structure de réponse standard de l'API.
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SuccessResponse crée une réponse de succès.
func SuccessResponse(data interface{}, message string) APIResponse {
	return APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

// SuccessResponseEmpty crée une réponse de succès sans données.
func SuccessResponseEmpty(message string) APIResponse {
	return APIResponse{
		Success: true,
		Message: message,
	}
}

// ErrorResponse crée une réponse d'erreur.
func ErrorResponse(errorMsg string) APIResponse {
	return APIResponse{
		Success: false,
		Error:   errorMsg,
	}
}
