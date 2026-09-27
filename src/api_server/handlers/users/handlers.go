package users

import (
	"chat/src/api_server/handlers/auth"
	"chat/src/api_server/handlers/middleware"
	userservice "chat/src/api_server/services/users"
	"chat/src/shared_api"
	"errors"
	"net/http"
)

func (this *UsersHandler) UserRegistrationHandler(w http.ResponseWriter, r *http.Request) {
	var request shared_api.UserRegistrationRequest
	if !middleware.DecodeJSON(w, r, &request) {
		return
	}

	if name_error := shared_api.ValidateUserName(request.Name); name_error != nil {
		middleware.WriteJSON(w, http.StatusUnprocessableEntity, shared_api.UserRegistrationResponse{
			Error: &shared_api.Error{Field: "name", Message: name_error.Error()},
		})
		return
	}

	if password_error := shared_api.ValidateUserPassword(request.Password); password_error != nil {
		middleware.WriteJSON(w, http.StatusUnprocessableEntity, shared_api.UserRegistrationResponse{
			Error: &shared_api.Error{Field: "password", Message: password_error.Error()},
		})
		return
	}

	user, err := this.Services.Users.RegisterUser(r.Context(), request.Name, request.Password)
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, shared_api.UserRegistrationResponse{
			Error: &shared_api.Error{Message: err.Error()},
		})
		return
	}

	var response shared_api.UserRegistrationResponse
	if user != nil {
		response.UserSession = &shared_api.UserSession{UID: user.UID, Token: user.Token}
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}

func (this *UsersHandler) UserLogInHandler(w http.ResponseWriter, r *http.Request) {
	var request shared_api.UserLogInRequest
	if !middleware.DecodeJSON(w, r, &request) {
		return
	}

	if name_error := shared_api.ValidateUserName(request.Name); name_error != nil {
		middleware.WriteJSON(w, http.StatusUnprocessableEntity, shared_api.UserLogInResponse{
			Error: &shared_api.Error{Field: "name", Message: name_error.Error()},
		})
		return
	}

	if password_error := shared_api.ValidateUserPassword(request.Password); password_error != nil {
		middleware.WriteJSON(w, http.StatusUnprocessableEntity, shared_api.UserLogInResponse{
			Error: &shared_api.Error{Field: "password", Message: password_error.Error()},
		})
		return
	}

	user, err := this.Services.Users.LogInUser(r.Context(), request.Name, request.Password)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, userservice.InvalidCredentialsError) {
			status = http.StatusUnauthorized
		}
		middleware.WriteJSON(w, status, shared_api.UserLogInResponse{
			Error: &shared_api.Error{Message: err.Error()},
		})
		return
	}

	var response shared_api.UserLogInResponse
	if user != nil {
		response.UserSession = &shared_api.UserSession{UID: user.UID, Token: user.Token}
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}

func (this *UsersHandler) UserLogOutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())

	if err := this.Services.Users.LogOutUser(r.Context(), session.Token); err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, shared_api.UserLogOutResponse{
			Error: &shared_api.Error{Message: err.Error()},
		})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, shared_api.UserLogOutResponse{})
}

func (this *UsersHandler) UsersGetHandler(w http.ResponseWriter, r *http.Request) {
	var request shared_api.UsersGetRequest
	if !middleware.DecodeQuery(w, r, &request) {
		return
	}

	users, err := this.Services.Users.GetUsers(r.Context(), request.UIDs)
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, shared_api.UsersGetResponse{
			Error: &shared_api.Error{Message: err.Error()},
		})
		return
	}

	response := shared_api.UsersGetResponse{Users: make([]shared_api.User, 0, len(users))}
	for i := range users {
		user := &users[i]
		response.Users = append(response.Users, shared_api.User{UID: user.UID, Name: user.Name})
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}
