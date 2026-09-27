package users

import (
	"chat/src/api_server/handlers/auth"
	"chat/src/api_server/handlers/middleware"
	userservice "chat/src/api_server/services/users"
	"chat/src/shared/api"
	userapi "chat/src/shared/api/user"
	"chat/src/shared/api/validator"
	"errors"
	"net/http"
)

func (this *UsersHandler) UserRegistrationHandler(w http.ResponseWriter, r *http.Request) {
	var request userapi.UserRegistrationRequest
	if !middleware.DecodeJSON(w, r, &request) {
		return
	}

	name, name_error := validator.ValidateUserName(request.Name)
	if name_error != nil {
		middleware.WriteJSON(w, http.StatusUnprocessableEntity, userapi.UserRegistrationResponse{
			Error: name_error,
		})
		return
	}

	password, password_error := validator.ValidateUserPassword(request.Password)
	if password_error != nil {
		middleware.WriteJSON(w, http.StatusUnprocessableEntity, userapi.UserRegistrationResponse{
			Error: password_error,
		})
		return
	}

	user, err := this.Services.Users.RegisterUser(r.Context(), name, password)
	if err != nil {
		if errors.Is(err, userservice.DuplicateUserNameError) {
			middleware.WriteJSON(w, http.StatusConflict, userapi.UserRegistrationResponse{
				Error: &api.Error{Field: "name", Message: err.Error()},
			})
			return
		}
		middleware.WriteJSON(w, http.StatusInternalServerError, userapi.UserRegistrationResponse{
			Error: &api.Error{Message: err.Error()},
		})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, userapi.UserRegistrationResponse{
		UserSession: &userapi.UserSession{UID: user.UID, Token: user.Token},
	})
}

func (this *UsersHandler) UserLogInHandler(w http.ResponseWriter, r *http.Request) {
	var request userapi.UserLogInRequest
	if !middleware.DecodeJSON(w, r, &request) {
		return
	}

	name, name_error := validator.ValidateUserName(request.Name)
	if name_error != nil {
		middleware.WriteJSON(w, http.StatusUnprocessableEntity, userapi.UserLogInResponse{
			Error: name_error,
		})
		return
	}

	password, password_error := validator.ValidateUserPassword(request.Password)
	if password_error != nil {
		middleware.WriteJSON(w, http.StatusUnprocessableEntity, userapi.UserLogInResponse{
			Error: password_error,
		})
		return
	}

	user, err := this.Services.Users.LogInUser(r.Context(), name, password)
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

	middleware.WriteJSON(w, http.StatusOK, userapi.UserLogInResponse{
		UserSession: &userapi.UserSession{UID: user.UID, Token: user.Token},
	})
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
