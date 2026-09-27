package users

import (
	"chat/src/api_server/handlers/auth"
	"chat/src/api_server/handlers/middleware"
	userservice "chat/src/api_server/services/users"
	"chat/src/shared/api"
	userapi "chat/src/shared/api/user"
	"errors"
	"net/http"
)

func (this *UsersHandler) UserRegistrationHandler(w http.ResponseWriter, r *http.Request) {
	var request userapi.UserRegistrationRequest
	if !middleware.DecodeJSON(w, r, &request) {
		return
	}

	if name_error := api.ValidateUserName(request.Name); name_error != nil {
		middleware.WriteJSON(w, http.StatusUnprocessableEntity, userapi.UserRegistrationResponse{
			Error: &api.Error{Field: "name", Message: name_error.Error()},
		})
		return
	}

	if password_error := api.ValidateUserPassword(request.Password); password_error != nil {
		middleware.WriteJSON(w, http.StatusUnprocessableEntity, userapi.UserRegistrationResponse{
			Error: &api.Error{Field: "password", Message: password_error.Error()},
		})
		return
	}

	user, err := this.Services.Users.RegisterUser(r.Context(), request.Name, request.Password)
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, userapi.UserRegistrationResponse{
			Error: &api.Error{Message: err.Error()},
		})
		return
	}

	var response userapi.UserRegistrationResponse
	if user != nil {
		response.UserSession = &userapi.UserSession{UID: user.UID, Token: user.Token}
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}

func (this *UsersHandler) UserLogInHandler(w http.ResponseWriter, r *http.Request) {
	var request userapi.UserLogInRequest
	if !middleware.DecodeJSON(w, r, &request) {
		return
	}

	if name_error := api.ValidateUserName(request.Name); name_error != nil {
		middleware.WriteJSON(w, http.StatusUnprocessableEntity, userapi.UserLogInResponse{
			Error: &api.Error{Field: "name", Message: name_error.Error()},
		})
		return
	}

	if password_error := api.ValidateUserPassword(request.Password); password_error != nil {
		middleware.WriteJSON(w, http.StatusUnprocessableEntity, userapi.UserLogInResponse{
			Error: &api.Error{Field: "password", Message: password_error.Error()},
		})
		return
	}

	user, err := this.Services.Users.LogInUser(r.Context(), request.Name, request.Password)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, userservice.InvalidCredentialsError) {
			status = http.StatusUnauthorized
		}
		middleware.WriteJSON(w, status, userapi.UserLogInResponse{
			Error: &api.Error{Message: err.Error()},
		})
		return
	}

	var response userapi.UserLogInResponse
	if user != nil {
		response.UserSession = &userapi.UserSession{UID: user.UID, Token: user.Token}
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}

func (this *UsersHandler) UserLogOutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())

	if err := this.Services.Users.LogOutUser(r.Context(), session.Token); err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, userapi.UserLogOutResponse{
			Error: &api.Error{Message: err.Error()},
		})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, userapi.UserLogOutResponse{})
}

func (this *UsersHandler) UsersGetHandler(w http.ResponseWriter, r *http.Request) {
	var request userapi.UsersGetRequest
	if !middleware.DecodeQuery(w, r, &request) {
		return
	}

	users, err := this.Services.Users.GetUsers(r.Context(), request.UIDs)
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, userapi.UsersGetResponse{
			Error: &api.Error{Message: err.Error()},
		})
		return
	}

	response := userapi.UsersGetResponse{Users: make([]userapi.User, 0, len(users))}
	for i := range users {
		user := &users[i]
		response.Users = append(response.Users, userapi.User{UID: user.UID, Name: user.Name})
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}
