package services

import (
	"chat/internal/webui_server/services/apiclient"
	"chat/internal/webui_server/services/sessions"
)

// Services groups the webui domain services. It mirrors api_server's
// services.Services.
type Services struct {
	// Sessions is the in-memory store of WebUI sessions and their connected
	// api servers.
	Sessions *sessions.SessionsService

	// API is the typed HTTP client used to call api_server instances.
	API *apiclient.Client
}

// CreateServices builds the service graph. There is no database: webui state is
// in-memory only (see docs/requirements.md, D3).
func CreateServices() (*Services, error) {
	sessionsService, err := sessions.CreateSessions()
	if err != nil {
		return nil, err
	}

	apiClient := apiclient.CreateClient()

	return &Services{
		Sessions: sessionsService,
		API:      apiClient,
	}, nil
}
