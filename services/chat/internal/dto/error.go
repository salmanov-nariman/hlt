package dto

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"` // omitempty скроет поле, если оно пустое
}