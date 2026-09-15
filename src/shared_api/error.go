package shared_api

type Error struct {
	Code    uint   `json:"code"`
	Name    string `json:"name"`
	Message string `json:"message"`
}
