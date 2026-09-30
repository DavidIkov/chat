package apiclient

// APIError is a failure returned by an api_server. It is decoded from the
// backend's {"error":{"field":..,"message":..}} body, or synthesised from a
// non-2xx status that has no structured error.
type APIError struct {
	// Status is the HTTP status code of the response.
	Status int
	// Field names the offending form field when the backend reported one.
	Field string
	// Message is the human-readable message shown in the UI.
	Message string
}

func (this *APIError) Error() string {
	return this.Message
}
