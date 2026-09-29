package apiclient

import "net/http"

// CreateClient builds the api_server client.
//
// The client is stateless with respect to WebUI sessions: every method receives
// the target baseURL and token explicitly. It uses http.DefaultClient for now; a
// caller that wants timeouts can construct Client with a custom *http.Client.
func CreateClient() *Client {
	return &Client{httpClient: http.DefaultClient}
}
