package apiclient

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"chat/internal/shared/api"
	userapi "chat/internal/shared/api/user"
)

// asAPIError converts a structured api.Error that survived do's own envelope
// check (i.e. a 2xx body that still carries an "error" object) into the
// *APIError shape the rest of the WebUI expects.
func asAPIError(status int, err *api.Error) *APIError {
	return &APIError{Status: status, Field: err.Field, Message: err.Message}
}

// Register creates a new account on the target api_server and returns its
// session (uid + token). Maps to POST /user/register.
func (this *Client) Register(ctx context.Context, baseURL string, name string, password string) (*userapi.UserSession, error) {
	request := userapi.UserRegistrationRequest{Name: name, Password: password}
	var response userapi.UserRegistrationResponse
	if err := this.do(ctx, http.MethodPost, baseURL, userRegisterPath, nil, "", request, &response); err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, asAPIError(http.StatusOK, response.Error)
	}
	return response.UserSession, nil
}

// LogIn signs in on the target api_server and returns its session (uid + token).
// Maps to POST /user/login.
func (this *Client) LogIn(ctx context.Context, baseURL string, name string, password string) (*userapi.UserSession, error) {
	request := userapi.UserLogInRequest{Name: name, Password: password}
	var response userapi.UserLogInResponse
	if err := this.do(ctx, http.MethodPost, baseURL, userLoginPath, nil, "", request, &response); err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, asAPIError(http.StatusOK, response.Error)
	}
	return response.UserSession, nil
}

// LogOut invalidates the token on the target api_server.
// Maps to POST /user/logout.
func (this *Client) LogOut(ctx context.Context, baseURL string, token string) error {
	var response userapi.UserLogOutResponse
	if err := this.do(ctx, http.MethodPost, baseURL, userLogoutPath, nil, token, nil, &response); err != nil {
		return err
	}
	if response.Error != nil {
		return asAPIError(http.StatusOK, response.Error)
	}
	return nil
}

// DeleteUser deletes the account behind the token, optionally removing its
// messages. Maps to POST /user/delete.
func (this *Client) DeleteUser(ctx context.Context, baseURL string, token string, deleteMessages bool) error {
	request := userapi.UserDeleteRequest{DeleteMessages: deleteMessages}
	var response userapi.UserDeleteResponse
	if err := this.do(ctx, http.MethodPost, baseURL, userDeletePath, nil, token, request, &response); err != nil {
		return err
	}
	if response.Error != nil {
		return asAPIError(http.StatusOK, response.Error)
	}
	return nil
}

// GetUsers resolves uids to display names. Maps to GET /user/get?uids=...
func (this *Client) GetUsers(ctx context.Context, baseURL string, token string, uids []uint) ([]userapi.User, error) {
	values := url.Values{}
	for _, uid := range uids {
		values.Add("uids", strconv.FormatUint(uint64(uid), 10))
	}
	var response userapi.UsersGetResponse
	if err := this.do(ctx, http.MethodGet, baseURL, userGetPath, values, token, nil, &response); err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, asAPIError(http.StatusOK, response.Error)
	}
	return response.Users, nil
}
