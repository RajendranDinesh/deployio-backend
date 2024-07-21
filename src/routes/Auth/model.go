package auth

type UserHandler struct{}

type UserSignInPayload struct {
	Code string `json:"code"`
}

type GH_UAT_API_Response struct {
	AccessToken           string `json:"access_token"`
	AccessTokenExpiresIn  int    `json:"expires_in"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
	TokenType             string `json:"token_type"`
}

type GhUserInfoResponse struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type GhError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorURI         string `json:"error_uri"`
}
