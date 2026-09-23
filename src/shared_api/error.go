package shared_api

type Error struct {
	Field    string `json:"field,omitempty"`
	Message string `json:"message"`
}
