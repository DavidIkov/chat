// Package routes holds path-wildcard names shared by the handler packages.
//
// It exists as a leaf package so that handlers/servers and handlers/chats can
// both refer to the same keys without importing each other (which would create a
// cycle through handlers).
package routes

const (
	// ServerIDPathKey is the wildcard name for a connected server id in webui
	// URLs, e.g. GET /servers/{server_id}.
	ServerIDPathKey = "server_id"

	// ChatUIDPathKey is the wildcard name for an api_server chat uid in webui
	// URLs, e.g. GET /servers/{server_id}/chats/{chat_uid}.
	ChatUIDPathKey = "chat_uid"
)
