package chats

type MessageCursorDirection int

const (
	MessagesBefore MessageCursorDirection = iota
	MessagesAfter
)
