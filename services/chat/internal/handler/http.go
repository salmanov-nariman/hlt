package handler

import (
	"chat/internal/service"
	"encoding/json"
	"net/http"
)

type ChatHandler struct {
	chatService *service.ChatService
}

func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

func (h *ChatHandler) PostMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	senderID := r.Header.Get("X-User-Id")
	if senderID == "" {
		http.Error(w, "X-User-Id header is missing", http.StatusBadRequest)
		return
	}

	var req struct {
		RoomID string `json:"room_id"`
		Text   string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.chatService.ProcessMessage(r.Context(), senderID, req.RoomID, req.Text); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		http.Error(w, "Invalid write response", http.StatusBadRequest)
	}
}

func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	senderID := r.Header.Get("X-User-Id")
	if senderID == "" {
		http.Error(w, "X-User-Id header is missing", http.StatusBadRequest)
		return
	}

	var req struct {
		MemberIDs []string `json:"member_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(req.MemberIDs) == 0 {
		http.Error(w, "At least one chat member is required", http.StatusBadRequest)
		return
	}

	chatID, err := h.chatService.CreateChat(r.Context(), senderID, req.MemberIDs)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(map[string]string{
		"chat_id": chatID,
	})
}
