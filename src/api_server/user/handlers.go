package user

import (
	"chat/src/shared_api"
	"encoding/json"
	"net/http"
)

func (this *UsersManager) UserRegistrationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request shared_api.UserRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var response shared_api.UserRegistrationResponse

	if name_error := shared_api.ValidateUserName(request.Name); name_error.Code != 0 {
		response.Error = name_error
	} else if password_error := shared_api.ValidateUserName(request.Name); password_error.Code != 0 {
		response.Error = password_error
	} else {
		user, err := this.RegisterUser(r.Context(), request.Name, request.Password)
		if err != nil {
			response.Error = shared_api.Error{Code: http.StatusInternalServerError, Name: "internal_error", Message: err.Error()}
		}
		if user != nil {
			response.User.UID = user.UID
			response.User.Name = user.Name
			response.User.Token = user.Token
		}
	}

	json.NewEncoder(w).Encode(response)
}
