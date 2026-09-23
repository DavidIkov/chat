package user

import (
	"chat/src/shared_api"
	"encoding/json"
	"errors"
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

	if name_error := shared_api.ValidateUserName(request.Name); name_error != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		response.Error = &shared_api.Error{Field: "name", Message: name_error.Error()}
	} else if password_error := shared_api.ValidateUserPassword(request.Password); password_error != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		response.Error = &shared_api.Error{Field: "password", Message: password_error.Error()}
	} else {
		user, err := this.RegisterUser(r.Context(), request.Name, request.Password)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response.Error = &shared_api.Error{Message: err.Error()}
		}
		if user != nil {
			response.User = &shared_api.User{UID: user.UID, Name: user.Name, Token: user.Token}
		}
	}

	json.NewEncoder(w).Encode(response)
}

func (this *UsersManager) UserLogInHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request shared_api.UserLogInRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var response shared_api.UserLogInResponse

	if name_error := shared_api.ValidateUserName(request.Name); name_error != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		response.Error = &shared_api.Error{Field: "name", Message: name_error.Error()}
	} else if password_error := shared_api.ValidateUserPassword(request.Password); password_error != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		response.Error = &shared_api.Error{Field: "password", Message: password_error.Error()}
	} else {
		user, err := this.LogInUser(r.Context(), request.Name, request.Password)
		if err != nil {
			switch {
			case errors.Is(err, InvalidCredentialsError):
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}
			response.Error = &shared_api.Error{Message: err.Error()}
		}
		if user != nil {
			response.User = &shared_api.User{UID: user.UID, Name: user.Name, Token: user.Token}
		}
	}

	json.NewEncoder(w).Encode(response)
}
