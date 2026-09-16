package shared_api

type Error struct {
	Code    int   `json:"code"`
	Name    string `json:"name"`
	Message string `json:"message"`
}
