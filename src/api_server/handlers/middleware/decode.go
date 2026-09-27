package middleware

import (
	"encoding/json"
	"github.com/go-playground/form/v4"
	"net/http"
)

var queryDecoder = form.NewDecoder()

// DecodeJSON decodes the request body into dst, writing 400 and returning false
// on failure.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return false
	}
	return true
}

// DecodeQuery decodes the URL query parameters into dst, writing 400 and
// returning false on failure.
func DecodeQuery(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := queryDecoder.Decode(dst, r.URL.Query()); err != nil {
		http.Error(w, "invalid query params", http.StatusBadRequest)
		return false
	}
	return true
}
