// Package render owns template loading and HTML responses.
//
// It is a leaf package: handlers/servers and handlers/chats both import it, so it
// must not import anything from handlers.
package render

import (
	"bytes"
	"html/template"
	"io/fs"
	"net/http"
)

// errorTemplateName is the file name of the template rendered by RenderError.
const errorTemplateName = "error.html"

// Page holds the fields that every page template needs. View models in the
// handler packages embed it so that {{.Title}} works inside the shared "head"
// partial.
type Page struct {
	Title string
}

// ErrorPage is the view model for error.html, used when a request cannot be
// served (missing connection, upstream failure, bad input, ...).
type ErrorPage struct {
	Page
	Message string
}

// Templates wraps the parsed page templates. It is created once at startup and
// is safe for concurrent use (html/template execution is safe for concurrent
// reads after parsing).
type Templates struct {
	templates *template.Template
}

// LoadTemplates parses every "*.html" file in templatesFS.
//
// Intended implementation: template.New("").ParseFS(templatesFS, "*.html").
// Files are complete documents that {{template "head" .}} / {{template "foot" .}}
// the partials defined in layout.html; there are no {{define "content"}}
// overrides, so parsing everything together is safe. Pages are rendered by their
// file name (see Render).
func LoadTemplates(templatesFS fs.FS) (*Templates, error) {
	parsed, err := template.ParseFS(templatesFS, "*.html")
	if err != nil {
		return nil, err
	}
	return &Templates{templates: parsed}, nil
}

// Render executes the template named name (e.g. "index.html") into w with status
// and data.
//
// Intended implementation: buffer the execution first, set
// "Content-Type: text/html; charset=utf-8", write the status, then flush the
// buffer, so a template error does not produce a half-written 200 response.
func (this *Templates) Render(w http.ResponseWriter, status int, name string, data any) {
	var buffer bytes.Buffer
	if err := this.templates.ExecuteTemplate(&buffer, name, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buffer.WriteTo(w)
}

// RenderError renders error.html with the given status and message.
func (this *Templates) RenderError(w http.ResponseWriter, status int, message string) {
	this.Render(w, status, errorTemplateName, ErrorPage{
		Page:    Page{Title: "Error"},
		Message: message,
	})
}

// Redirect answers with 303 See Other to url. Used to implement
// Post/Redirect/Get for every mutating route.
func Redirect(w http.ResponseWriter, r *http.Request, url string) {
	http.Redirect(w, r, url, http.StatusSeeOther)
}
