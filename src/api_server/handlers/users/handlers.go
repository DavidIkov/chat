package users

import (
	userservice "chat/src/api_server/services/users"
	"chat/src/shared_api"
	"encoding/json"
	"errors"
	"net/http"
)

func (this *UsersHandler) UserRegistrationHandler(w http.ResponseWriter, r *http.Request) {
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
		user, err := this.Services.Users.RegisterUser(r.Context(), request.Name, request.Password)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response.Error = &shared_api.Error{Message: err.Error()}
		}
		if user != nil {
			response.UserSession = &shared_api.UserSession{UID: user.UID, Token: user.Token}
		}
	}

	json.NewEncoder(w).Encode(response)
}

func (this *UsersHandler) UserLogInHandler(w http.ResponseWriter, r *http.Request) {
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
		user, err := this.Services.Users.LogInUser(r.Context(), request.Name, request.Password)
		if err != nil {
			switch {
			case errors.Is(err, userservice.InvalidCredentialsError):
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}
			response.Error = &shared_api.Error{Message: err.Error()}
		}
		if user != nil {
			response.UserSession = &shared_api.UserSession{UID: user.UID, Token: user.Token}
		}
	}

	json.NewEncoder(w).Encode(response)
}

func (this *UsersHandler) UserLogOutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request shared_api.UserLogOutRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var response shared_api.UserLogOutResponse

	err := this.Services.Users.LogOutUser(r.Context(), request.Token)
	if err != nil {
		response.Error = &shared_api.Error{Message: err.Error()}
		switch {
		case errors.Is(err, userservice.TokenNotFoundError):
			w.WriteHeader(http.StatusUnauthorized)
			response.Error.Field = "token"
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	json.NewEncoder(w).Encode(response)
}

func (this *UsersHandler) UsersGetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request shared_api.UsersGetRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var response shared_api.UsersGetResponse

	users, err := this.Services.Users.GetUsers(r.Context(), request.Token, request.UIDs)
	if err != nil {
		response.Error = &shared_api.Error{Message: err.Error()}
		w.WriteHeader(http.StatusInternalServerError)
	}

	response.Users = make([]shared_api.User, 0, len(users))
	for i := range users {
		user := &users[i]
		response.Users = append(response.Users, shared_api.User{UID: user.UID, Name: user.Name})
	}

	json.NewEncoder(w).Encode(response)
}
