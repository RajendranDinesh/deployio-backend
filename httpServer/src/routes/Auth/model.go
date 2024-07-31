package auth

import "time"

type AuthHandler struct{}

type UserSignInPayload struct {
	Code         string `json:"code"`
	RefreshToken string `json:"refresh_token"`
}

type GH_UAT_API_Response struct {
	AccessToken           string        `json:"access_token"`
	AccessTokenExpiresIn  time.Duration `json:"expires_in"`
	RefreshToken          string        `json:"refresh_token"`
	RefreshTokenExpiresIn time.Duration `json:"refresh_token_expires_in"`
	TokenType             string        `json:"token_type"`
}

type GhUserEmailResponse struct {
	Emails []EmailObject
}

type EmailObject struct {
	Email   string `json:"email"`
	Primary bool   `json:"primary"`
}

type GhUserNameResponse struct {
	Name string `json:"name"`
}

type GhError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorURI         string `json:"error_uri"`
}

type User struct {
	Id                 int64
	Email              string
	Access             string
	Refresh            string
	Access_expires_by  time.Duration
	Refresh_expires_by time.Duration
	Name               string
}
