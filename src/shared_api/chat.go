package shared_api

type Message struct {
	UserUID    uint   `json:"user_uid"`
	ChatUID    uint   `json:"chat_uid"`
	MessageUID uint   `json:"message_uid"`
	CreatedAt  string `json:"created_at"`
	Text       string `json:"text"`
}

type ChatSendMessageRequest struct {
	Token   string `json:"token"`
	ChatUID uint   `json:"chat_uid"`
	Text    string `json:"text"`
}

type ChatSendMessageResponse struct {
	Error   *Error   `json:"error,omitempty"`
	Message *Message `json"message,omitempty"`
}
