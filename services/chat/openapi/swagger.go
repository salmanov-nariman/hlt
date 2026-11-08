package openapi


// @title 		   				 HLT chat service API
// @version 	  				 1.0
// @description    				 API server for chat service
// @host           				 localhost:8080
// @BasePath       				 /api/v1
// @securityDefinitions.apikey   UserIdAuth
// @in 							 header
// @name 						 X-User-Id
func main() {}

// PostMessage godoc
// @Summary      Отправить сообщение
// @Description  Отправляет новое сообщение в указанный чат
// @Tags         messages
// @Accept       json
// @Produce      json
// @Security     UserIdAuth
// @Param        request  body      dto.PostMessageRequest  true  "Данные сообщения"
// @Success      201      {object}  dto.MessageResponse
// @Success      200      {object}  dto.MessageResponse "Сообщение уже существует"
// @Failure      400      {string}  string "Invalid JSON"
// @Failure      409      {string}  string `{"code":"CLIENT_MESSAGE_ID_CONFLICT"}`
// @Failure      500      {string}  string "Internal server error"
// @Router       /api/messages [post]
func PostMessage()

// CreateChat godoc
// @Summary      Создать чат
// @Description  Создает новый direct-чат с пользователем или возвращает существующий
// @Tags         chats
// @Accept       json
// @Produce      json
// @Security     UserIdAuth
// @Param        request  body      dto.CreateChatRequest true  "ID партнера"
// @Success      201      {object}  dto.ChatSummaryResponse "Чат создан"
// @Success      200      {object}  dto.ChatSummaryResponse "Чат уже существует"
// @Failure      400      {string}  string `{"code": "INVALID_PARTNER", "message": "Cannot create chat with yourself"}`
// @Failure      404      {string}  string `{"code": "USER_NOT_FOUND", "message": "Partner user does not exist"}`
// @Failure      500      {string}  string `{"code": "INTERNAL_ERROR", "message": "Internal server error"}`
// @Failure      502      {string}  string `{"code": "VALIDATION_ERROR", "message": "partner_id is required"}`
// @Router       /api/chats [post]
func CreateChat()

// SyncMessages godoc
// @Summary      Получить историю сообщений
// @Description  Возвращает список сообщений чата с фильтрацией по sequence
// @Tags         messages
// @Produce      json
// @Security     UserIdAuth
// @Param        chat_id         query  string  true   "ID чата"
// @Param        after_sequence  query  int     false  "Sequence сообщения, после которого нужно получить новые"
// @Param        limit           query  int     false  "Лимит сообщений (по умолчанию 50)"
// @Success      200             {array}   dto.MessageResponse
// @Failure      400             {string}  string "missing chat_id or invalid parameters"
// @Failure      403             {string}  string "is not member of chat"
// @Failure      500             {string}  string "internal server error"
// @Router       /api/chat/messages [get]
func SyncMessages() 

// MarkAsRead godoc
// @Summary      Прочитать сообщения
// @Description  Обновляет статус прочитанных сообщений (last_read_sequence) для текущего пользователя
// @Tags         chats
// @Accept       json
// @Produce      json
// @Security     UserIdAuth
// @Param        request  body      dto.MarkReadRequest  true  "Данные для отметки о прочтении"
// @Success      200      {string}  string "OK"
// @Failure      400      {string}  string "invalid request body or missing parameters"
// @Failure      403      {string}  string "is not member of chat"
// @Failure      500      {string}  string "internal server error"
// @Router       /api/chat/read [post]
func MarkAsRead()

// GetUserChats godoc
// @Summary      Получить список чатов
// @Description  Возвращает пагинированный список чатов текущего пользователя
// @Tags         chats
// @Produce      json
// @Security     UserIdAuth
// @Param        limit   query     int     false  "Лимит чатов (от 1 до 50, по умолчанию 20)"
// @Param        cursor  query     string  false  "Курсор для пагинации"
// @Success      200     {object}  dto.PaginatedChatsResponse
// @Failure      400     {string}  string `{"code": "INVALID_CURSOR"}`
// @Failure      500     {string}  string "Internal server error"
// @Router       /api/chats [get]
func  GetUserChats()