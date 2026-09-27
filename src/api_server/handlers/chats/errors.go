package chats

import (
	"errors"
)

// errNotChatMember is returned by ensureChatMember when the acting user does not
// belong to the chat they are trying to read from or write to. authStatus maps
// it to HTTP 403 Forbidden.
var errNotChatMember = errors.New("user does not belong to the chat")
