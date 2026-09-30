package apiclient

import "net/http"

// Client talks to one api_server per call. It holds only an HTTP client and is
// stateless with respect to WebUI sessions: methods receive the baseURL and token
// explicitly.
type Client struct {
	httpClient *http.Client
}
