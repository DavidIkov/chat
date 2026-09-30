package sessions

// CreateSessions builds an empty in-memory session store.
func CreateSessions() (*SessionsService, error) {
	var service SessionsService
	service.sessions = make(map[SessionID]*WebUISession)
	return &service, nil
}
