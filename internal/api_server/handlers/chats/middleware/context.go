package middleware

import (
	"chat/internal/shared"
	"context"
)

func withChatUID(ctx context.Context, chatUID shared.UID) context.Context {
	return context.WithValue(ctx, contextKey{}, chatUID)
}

func ChatUIDFromContext(ctx context.Context) (shared.UID, bool) {
	chatUID, ok := ctx.Value(contextKey{}).(shared.UID)
	return chatUID, ok
}
