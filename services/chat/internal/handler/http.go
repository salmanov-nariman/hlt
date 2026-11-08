package handler

import (
	"chat/internal/dto"
	"chat/internal/repository"
	"chat/internal/service"
	"chat/middleware"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
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

	senderID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var req dto.PostMessageRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	msg, err := h.chatService.ProcessMessage(r.Context(), senderID, req.ChatID, req.ClientMessageID, req.Text)

	w.Header().Set("Content-Type", "application/json")

	var messageDTO *dto.MessageResponse
	if msg != nil {
		messageDTO = &dto.MessageResponse{
			ID:              msg.ID,
			ChatID:          msg.ChatID,
			SenderID:        msg.SenderID,
			Content:         msg.Content,
			CreatedAt:       msg.CreatedAt,
			ClientMessageID: msg.ClientMessageID,
			Sequence:        msg.Sequence,
		}
	}

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
	json.NewEncoder(w).Encode(messageDTO)
}

func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	senderID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var req dto.CreateChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"code": "VALIDATION_ERROR", "message": "partner_id is required"}`))
		return
	}

	chat, isNew, err := h.chatService.CreateOrGetDirectChat(r.Context(), senderID, req.PartnerID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")

		if errors.Is(err, service.ErrInvalidPartner) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"code": "INVALID_PARTNER", "message": "Cannot create chat with yourself"}`))
			return
		}

		if errors.Is(err, service.ErrUserNotFound) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"code": "USER_NOT_FOUND", "message": "Partner user does not exist"}`))
			return
		}

		log.Printf("[ERROR] CreateOrGetDirectChat failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code": "INTERNAL_ERROR", "message": "Internal server error"}`))
		return
	}

	response := dto.ChatSummaryResponse{
		ID:          chat.ID,
		UnreadCount: chat.UnreadCount,
		UpdatedAt:   chat.UpdatedAt,
		Revision:    chat.Revision,
	}
	if chat.LastMessage != nil {
		response.LastMessage = &dto.MessageResponse{
			ID:              chat.LastMessage.ID,
			ChatID:          chat.LastMessage.ChatID,
			SenderID:        chat.LastMessage.SenderID,
			Content:         chat.LastMessage.Content,
			CreatedAt:       chat.LastMessage.CreatedAt,
			ClientMessageID: chat.LastMessage.ClientMessageID,
			Sequence:        chat.LastMessage.Sequence,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if isNew {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[ERROR] Failed to write CreateChat response: %v", err)
	}
}

func (h *ChatHandler) SyncMessages(w http.ResponseWriter, r *http.Request) {
	senderID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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

	response := make([]dto.MessageResponse, 0, len(messages))
	for _, m := range messages {
		response = append(response, dto.MessageResponse{
			ID:              m.ID,
			ChatID:          m.ChatID,
			SenderID:        m.SenderID,
			Content:         m.Content,
			CreatedAt:       m.CreatedAt,
			ClientMessageID: m.ClientMessageID,
			Sequence:        m.Sequence,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (h *ChatHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var req dto.MarkReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ChatID == "" || req.LastReadSequence <= 0 {
		http.Error(w, "chat_id and valid last_read_sequence are required", http.StatusBadRequest)
		return
	}

	err := h.chatService.UpdateReadStatus(r.Context(), req.ChatID, userID, req.LastReadSequence)
	if err != nil {
		if strings.Contains(err.Error(), "is not member of chat") {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ChatHandler) GetUserChats(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > 50 {
				l = 50
			}
			limit = l
		} else {
			http.Error(w, `{"code": "INVALID_PAGINATION"}`, http.StatusBadRequest)
			return
		}

	}

	cursor := r.URL.Query().Get("cursor")

	chats, nextCursor, err := h.chatService.GetUserChats(r.Context(), userID, limit, cursor)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCursor) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"code": "INVALID_CURSOR"}`))
			return
		}
		log.Printf("chatService.GetUserChats error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := dto.PaginatedChatsResponse{
		Items:      make([]dto.ChatSummaryResponse, 0, len(chats)),
		NextCursor: nextCursor,
	}

	for _, c := range chats {
		summaryDTO := dto.ChatSummaryResponse{
			ID:          c.ID,
			UnreadCount: c.UnreadCount,
			UpdatedAt:   c.UpdatedAt,
		}
		if c.LastMessage != nil {
			summaryDTO.LastMessage = &dto.MessageResponse{
				ID:              c.LastMessage.ID,
				ChatID:          c.LastMessage.ChatID,
				SenderID:        c.LastMessage.SenderID,
				Content:         c.LastMessage.Content,
				CreatedAt:       c.LastMessage.CreatedAt,
				ClientMessageID: c.LastMessage.ClientMessageID,
				Sequence:        c.LastMessage.Sequence,
			}
		}

		response.Items = append(response.Items, summaryDTO)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("failed to encode GetUserChats response: %v", err)
	}
}

func (h *ChatHandler) GetChat(w http.ResponseWriter, r *http.Request) {
    userID, ok := middleware.GetUserID(r.Context())
    if !ok {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusUnauthorized)
        w.Write([]byte(`{"code": "UNAUTHORIZED", "message": "Требуется авторизация"}`))
        return
    }

    chatID := chi.URLParam(r, "chat_id")

    chat, err := h.chatService.GetChat(r.Context(), chatID, userID)
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
        
        if errors.Is(err, repository.ErrChatNotFound) || errors.Is(err, service.ErrUserNotInChat) {
            w.WriteHeader(http.StatusNotFound)
            w.Write([]byte(`{"code": "NOT_FOUND", "message": "Чат не найден"}`))
            return
        }

        w.WriteHeader(http.StatusInternalServerError)
        w.Write([]byte(`{"code": "INTERNAL_ERROR", "message": "Внутренняя ошибка сервера"}`))
        return
    }

    resp := dto.ChatSummaryResponse{
        ID:          chat.ID,
        UnreadCount: chat.UnreadCount,
        UpdatedAt:   chat.UpdatedAt,
        Revision:    chat.Revision,
    }
    if chat.LastMessage != nil {
        resp.LastMessage = &dto.MessageResponse{
            ID:              chat.LastMessage.ID,
            ChatID:          chat.LastMessage.ChatID,
            SenderID:        chat.LastMessage.SenderID,
            Content:         chat.LastMessage.Content,
            CreatedAt:       chat.LastMessage.CreatedAt,
            ClientMessageID: chat.LastMessage.ClientMessageID,
            Sequence:        chat.LastMessage.Sequence,
        }
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    if err := json.NewEncoder(w).Encode(resp); err != nil {
        log.Printf("[ERROR] failed to encode GetUserChats response: %v", err)
    }
}

func (h *ChatHandler) GetChatMessages(w http.ResponseWriter, r *http.Request) {
    userID, ok := middleware.GetUserID(r.Context())
    if !ok {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusUnauthorized)
        w.Write([]byte(`{"code": "UNAUTHORIZED", "message": "Требуется авторизация"}`))
        return
    }

    chatID := chi.URLParam(r, "chat_id")
    if chatID == "" {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    
    limit := 50
    if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
        if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
            if parsedLimit > 100 {
                limit = 100
            } else {
                limit = parsedLimit
            }
        }
    }

    var cursor *int64
    if cursorStr := r.URL.Query().Get("cursor"); cursorStr != "" {
        parsedCursor, err := strconv.ParseInt(cursorStr, 10, 64)
        if err != nil {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusBadRequest)
            w.Write([]byte(`{"code": "INVALID_CURSOR", "message": "Некорректный формат курсора"}`))
            return
        }
        cursor = &parsedCursor
    }

    messages, nextCursor, err := h.chatService.GetChatMessages(r.Context(), chatID, userID, cursor, limit)
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
        if errors.Is(err, service.ErrUserNotFound) {
            w.WriteHeader(http.StatusForbidden)
            w.Write([]byte(`{"code": "FORBIDDEN", "message": "Нет доступа к чату"}`))
            return 
        }

        w.WriteHeader(http.StatusInternalServerError)
        w.Write([]byte(`{"code": "INTERNAL_ERROR", "message": "Внутренняя ошибка сервера"}`))
        return
    }

    response := dto.PaginatedChatMessagesResponse{
        Messages: make([]dto.MessageResponse, 0, len(messages)),
        NextCursor: nextCursor,
    }

    for _, msg := range messages {
        response.Messages = append(response.Messages, dto.MessageResponse{
            ID:              msg.ID,
            ChatID:          msg.ChatID,
            SenderID:        msg.SenderID,
            Content:         msg.Content,
            CreatedAt:       msg.CreatedAt,
            ClientMessageID: msg.ClientMessageID,
            Sequence:        msg.Sequence,
        })
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("[ERROR] failed to encode GetChatMessages response: %v", err)
    }
}
