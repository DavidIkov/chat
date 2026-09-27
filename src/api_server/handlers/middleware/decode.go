package middleware

import (
	"encoding/json"
	"github.com/go-playground/form/v4"
	"net/http"
	"sync"
)

// form.Decoder is not safe for concurrent use, so requests must not share one.
// Building a fresh decoder per call would be correct but would also discard the
// decoder's struct-tag cache on every request, so decoders are pooled instead:
// concurrent requests each borrow their own instance while the parsed struct
// definitions survive and are reused across requests.
var queryDecoders = sync.Pool{
	New: func() any { return form.NewDecoder() },
}

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
	decoder := queryDecoders.Get().(*form.Decoder)
	defer queryDecoders.Put(decoder)

	if err := decoder.Decode(dst, r.URL.Query()); err != nil {
		http.Error(w, "invalid query params", http.StatusBadRequest)
		return false
	}
	return true
}
