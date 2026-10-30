package handler

import (
	"chat/internal/repository"
	"chat/internal/service"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
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
		ChatID string `json:"chat_id"`
		ClientMessageID string `json:"client_message_id"`
		Text   string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	msg, err := h.chatService.ProcessMessage(r.Context(), senderID, req.ChatID, req.ClientMessageID, req.Text)
	
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		if errors.Is(err, repository.ErrMessageConflict) {
			w.WriteHeader(http.StatusConflict) // 409
			w.Write([]byte(`{"code":"CLIENT_MESSAGE_ID_CONFLICT"}`))
			return
		} else if errors.Is(err, repository.ErrMessageAlreadyExists) {
			w.WriteHeader(http.StatusOK) // 200 OK
			json.NewEncoder(w).Encode(msg)
			return
		}

		log.Printf("ProcessMessage error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(msg)
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
		log.Printf("[ERROR] Не удалось создать чат: %v", err)

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(map[string]string{
		"chat_id": chatID,
	})
}


func (h *ChatHandler) SyncMessages(w http.ResponseWriter, r *http.Request) {
	senderID := r.Header.Get("X-User-Id")
	if senderID == "" {
		http.Error(w, "X-User-Id header is missing", http.StatusBadRequest)
		return
	}

	chatID := r.URL.Query().Get("chat_id")
	if chatID == "" {
		http.Error(w, "missing chat_id query parameter", http.StatusBadRequest)
		return
	}

	var afterSequence int64
	afterSeqStr := r.URL.Query().Get("after_sequence")
	if afterSeqStr != "" {
		parsedSeq, err := strconv.ParseInt(afterSeqStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid after_sequence parameter", http.StatusBadRequest)
			return
		}
		afterSequence = parsedSeq
	}

	limit := 50
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	messages, err := h.chatService.SyncMessages(r.Context(), senderID, chatID, afterSequence, limit)
	if err != nil {
		if strings.Contains(err.Error(), "is not member of chat") {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return 
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
