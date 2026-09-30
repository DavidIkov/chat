package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// do sends one JSON request to an api_server and decodes the JSON response.
//
// Intended implementation:
//   - resolve the target URL from baseURL + path (+ query) with net/url;
//   - when requestBody != nil, JSON-marshal it and set
//     "Content-Type: application/json";
//   - when token != "", set "Authorization: " + bearerPrefix + token;
//   - send with this.httpClient.Do(req.WithContext(ctx));
//   - when responseBody != nil, JSON-decode the body into it;
//   - on a non-2xx status or an {"error":{...}} payload, return *APIError
//     {Status, Field, Message}.
//
// query may be nil. path is the already-formatted endpoint path.
func (this *Client) do(ctx context.Context, method string, baseURL string, path string, query url.Values, token string, requestBody any, responseBody any) error {
	target := strings.TrimRight(baseURL, "/") + path
	if query != nil {
		if encoded := query.Encode(); encoded != "" {
			target += "?" + encoded
		}
	}

	var body io.Reader
	if requestBody != nil {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return err
	}
	if requestBody != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", bearerPrefix+token)
	}

	response, err := this.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	// Best effort: the api_server returns {"error":{...}} for structured failures.
	// Non-JSON bodies (e.g. http.Error plain text) are handled by the status check.
	var envelope struct {
		Error *struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &envelope)
	}
	if envelope.Error != nil {
		return &APIError{Status: response.StatusCode, Field: envelope.Error.Field, Message: envelope.Error.Message}
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message := http.StatusText(response.StatusCode)
		if trimmed := strings.TrimSpace(string(raw)); trimmed != "" {
			message = trimmed
		}
		return &APIError{Status: response.StatusCode, Message: message}
	}

	if responseBody != nil && len(raw) > 0 {
		return json.Unmarshal(raw, responseBody)
	}
	return nil
}
