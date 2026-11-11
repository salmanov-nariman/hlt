package handler

import (
	"chat/internal/domain"
	"chat/internal/dto"
	"chat/internal/httputil"
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
	senderID, ok := middleware.GetUserID(r.Context())
	if !ok {
		httputil.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется авторизация")
		return
	}

	chatID := chi.URLParam(r, "chat_id")

	var req dto.PostMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "INVALID_JSON", "Неверный формат запроса")
		return
	}

	msg, err := h.chatService.ProcessMessage(r.Context(), senderID, chatID, req.ClientMessageID, req.Text)

	if err != nil {
		if errors.Is(err, domain.ErrMessageConflict) {
			httputil.JSONError(w, http.StatusConflict, "CLIENT_MESSAGE_ID_CONFLICT", "")
			return
		}

		if errors.Is(err, domain.ErrUserNotInChat) {
			httputil.JSONError(w, http.StatusForbidden, "FORBIDDEN", "Нет доступа к чату")
			return
		}

		if errors.Is(err, domain.ErrMessageAlreadyExists) {
			httputil.JSON(w, http.StatusOK, mapDomainToDTO(msg))
			return
		}

		log.Printf("ProcessMessage error: %v", err)
		httputil.JSONError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Внутренняя ошибка сервера")
		return
	}
	
	httputil.JSON(w, http.StatusCreated, mapDomainToDTO(msg))
}

func mapDomainToDTO(msg *domain.Message) *dto.MessageResponse {
	if msg == nil {
		return nil
	} else {
		return &dto.MessageResponse{
			ID:              msg.ID,
			ChatID:          msg.ChatID,
			SenderID:        msg.SenderID,
			Content:         msg.Content,
			CreatedAt:       msg.CreatedAt,
			ClientMessageID: msg.ClientMessageID,
			Sequence:        msg.Sequence,
		}
	}
}

func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	senderID, ok := middleware.GetUserID(r.Context())
	if !ok {
		httputil.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется авторизация")		
		return
	}

	var req dto.CreateChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "INVALID_JSON", "Неверный формат запроса")
		return
	}

	chat, isNew, err := h.chatService.CreateOrGetDirectChat(r.Context(), senderID, req.PartnerID)
	if err != nil {

		if errors.Is(err, service.ErrInvalidPartner) {
			httputil.JSONError(w, http.StatusBadRequest, "INVALID_PARTNER", "Невозможно создать чат самим с собой")
			return
		}

		if errors.Is(err, service.ErrUserNotFound) {
			httputil.JSONError(w, http.StatusNotFound, "USER_NOT_FOUND", "")
			return
		}

		httputil.JSONError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Внутренняя ошибка сервера")
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

	statusCode := http.StatusOK
	if isNew {
		statusCode = http.StatusCreated
	}

	httputil.JSON(w, statusCode, response)
}

func (h *ChatHandler) SyncMessages(w http.ResponseWriter, r *http.Request) {
	senderID, ok := middleware.GetUserID(r.Context())
	if !ok {
		httputil.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется авторизация")
		return
	}

	chatID := r.URL.Query().Get("chat_id")
	if chatID == "" {
		httputil.JSONError(w, http.StatusBadRequest, "MISSING_QUERY_PARAM", "Отсутствует query parameter: chat_id")
		return
	}

	var afterSequence int64
	afterSeqStr := r.URL.Query().Get("after_sequence")
	if afterSeqStr != "" {
		parsedSeq, err := strconv.ParseInt(afterSeqStr, 10, 64)
		if err != nil {
			httputil.JSONError(w, http.StatusBadRequest, "INVALID_QUERY_PARAM", "")
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
			httputil.JSONError(w, http.StatusForbidden, "FORBIDDEN", "")
			return
		}

		httputil.JSONError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Внутренняя ошибка сервера")
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

	httputil.JSON(w, http.StatusOK, response)
}

func (h *ChatHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		httputil.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется авторизация")
		return
	}

	chatID := chi.URLParam(r, "chat_id")

	var req dto.MarkReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "INVALID_JSON", "Неверный формат запроса")
		return
	}

	if req.LastReadSequence <= 0 {
		httputil.JSONError(w, http.StatusBadRequest, "INVALID_JSON", "")
		return
	}

	err := h.chatService.UpdateReadStatus(r.Context(), chatID, userID, req.LastReadSequence)
	if err != nil {
		if strings.Contains(err.Error(), "is not member of chat") {
			httputil.JSONError(w, http.StatusForbidden, "FORBIDDEN", "")
			return
		}

		httputil.JSONError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Внутренняя ошибка сервера")
		return
	}

	httputil.NoContent(w)
}

func (h *ChatHandler) GetUserChats(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		httputil.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется авторизация")
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
			httputil.JSONError(w, http.StatusBadRequest, "INVALID_PAGINATION", "")
			return
		}

	}

	cursor := r.URL.Query().Get("cursor")

	chats, nextCursor, err := h.chatService.GetUserChats(r.Context(), userID, limit, cursor)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCursor) {
			httputil.JSONError(w, http.StatusBadRequest, "INVALID_CURSOR", "")
			return
		}
		log.Printf("chatService.GetUserChats error: %v", err)
		httputil.JSONError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Внутренняя ошибка сервера")
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

	httputil.JSON(w, http.StatusOK, response)
}

func (h *ChatHandler) GetChat(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		httputil.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется авторизация")
		return
	}

	chatID := chi.URLParam(r, "chat_id")

	chat, err := h.chatService.GetChat(r.Context(), chatID, userID)
	if err != nil {

		if errors.Is(err, domain.ErrChatNotFound) || errors.Is(err, domain.ErrUserNotInChat) {
			httputil.JSONError(w, http.StatusNotFound, "NOT_FOUND", "Чат не найден")
			return
		}

		httputil.JSONError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Внутренняя ошибка сервера")
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

	httputil.JSON(w, http.StatusOK, resp)
}

func (h *ChatHandler) GetChatMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		httputil.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется авторизация")
		return
	}

	chatID := chi.URLParam(r, "chat_id")
	if chatID == "" {
		httputil.JSONError(w, http.StatusBadRequest, "INVALID_URL", "")
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
			httputil.JSONError(w, http.StatusBadRequest, "INVALID_CURSOR", "Некорректный формат курсора")
			return
		}
		cursor = &parsedCursor
	}

	messages, nextCursor, err := h.chatService.GetChatMessages(r.Context(), chatID, userID, cursor, limit)
	if err != nil {

		if errors.Is(err, service.ErrUserNotFound) {
			httputil.JSONError(w, http.StatusForbidden, "FORBIDDEN", "Нет доступа к чату")
			return
		}

		httputil.JSONError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Внутренняя ошибка сервера")
		return
	}

	response := dto.PaginatedChatMessagesResponse{
		Messages:   make([]dto.MessageResponse, 0, len(messages)),
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

	httputil.JSON(w, http.StatusOK, response)
}
