package httputil

import (
	"chat/internal/dto"
	"encoding/json"
	"log"
	"net/http"
)

func JSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := dto.ErrorResponse{
		Code:    code,
		Message: message,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[ERROR] failed to encode error response: %v", err)
	}
}

func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("[ERROR] failed to encode response: %v", err)
	}
}
