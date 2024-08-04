package auth

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"httpServer/config"
	"httpServer/utils"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/jwtauth/v5"
)

func (u AuthHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	var user UserSignInPayload

	// gets Code from body and stores it into user
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		utils.HandleError(utils.ErrInvalid, nil, w, nil)
		return
	}

	if len(strings.TrimSpace(user.Code)) <= 0 {
		utils.HandleError(utils.ErrInvalid, nil, w, nil)
		return
	}

	cId, cSecret := GetClientIdnSecret()

	response, err := GetOauthResponse(cId, cSecret, user)
	if err != nil {
		utils.HandleError(utils.ErrUnAuthorized, err, w, nil)
		return
	}

	email, err := getUserEmail(response.AccessToken)
	if err != nil {
		utils.HandleError(utils.ErrUnAuthorized, err, w, nil)
		return
	}

	var userId *int64

	userId, userExists := DoesUserExists(*email)
	if !userExists && userId == nil {
		userInfo, err := FetchUserInfoFromGitHub(response.AccessToken)

		if err != nil {
			utils.HandleError(utils.ErrUnAuthorized, err, w, nil)
			return
		}

		var user User

		user.Email = *email
		user.Name = userInfo.Name
		user.Access = response.AccessToken
		user.Refresh = response.RefreshToken
		user.Access_expires_by = response.AccessTokenExpiresIn
		user.Refresh_expires_by = response.RefreshTokenExpiresIn

		userId, err = InsertNewUser(user)

		if err != nil {
			utils.HandleError(utils.ErrUnAuthorized, err, w, nil)
			return
		}

	} else {
		UpdateUserTokens(response, *userId)
	}

	jwtToken := generateJWT(*userId)

	body := map[string]string{
		"token": jwtToken,
	}

	responseBody, err := json.Marshal(body)
	if err != nil {
		utils.HandleError(utils.ErrUnAuthorized, err, w, nil)
		return
	}

	w.Write([]byte(responseBody))
}

func GetOauthResponse(cId string, cSecret string, user UserSignInPayload) (GH_UAT_API_Response, error) {

	var GHAPIResponse GH_UAT_API_Response
	var GhAPIError GhError

	// creating params to call the github api to get personal access token

	params := map[string]string{
		"client_id":     cId,
		"client_secret": cSecret,
	}

	if len(strings.TrimSpace(user.Code)) > 0 {
		params["code"] = user.Code
	} else if len(strings.TrimSpace(user.RefreshToken)) > 0 {
		params["refresh_token"] = user.RefreshToken
		params["grant_type"] = "refresh_token"
	}

	resp, err := utils.Request("POST", "https://github.com/login/oauth/access_token", nil, &params, nil)
	if err != nil {
		fmt.Println("[AUTH] Error while calling GH's UAT API")
		return GHAPIResponse, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("[AUTH] Error while reading GH's UAT API response.")
		return GHAPIResponse, err
	}

	// response from github is stored inside GHAPIResponse
	err = json.Unmarshal(body, &GHAPIResponse)
	if err != nil {
		return GHAPIResponse, err
	}

	if len(GHAPIResponse.AccessToken) > 0 {
		// access token will be returned from here
		return GHAPIResponse, nil
	}

	err = json.Unmarshal(body, &GhAPIError)
	if err != nil {
		return GHAPIResponse, err
	}

	if len(GhAPIError.Error) > 0 {
		// if bad code is used the error from gh will be raised from here
		return GHAPIResponse, fmt.Errorf(GhAPIError.ErrorDescription)
	}

	return GHAPIResponse, err
}

func GetAccessToken(userId int) (*string, error) {
	query := `SELECT u.access FROM "deploy-io".users u WHERE u.id = $1`

	var accessToken string

	err := config.DataBase.QueryRow(query, userId).Scan(&accessToken)
	if err != nil {
		return nil, err
	}

	return &accessToken, nil
}

func AreTokensValid(userId int) (bool, bool) {
	isAccessValid := false
	isRefreshValid := false

	query := `
	SELECT
		CASE
			WHEN u.access_expires_by > current_timestamp at TIME ZONE 'Asia/Kolkata' THEN true
			ELSE false
		END AS is_access_valid,
		CASE
			WHEN u.refresh_expires_by > current_timestamp at TIME ZONE 'Asia/Kolkata' THEN true
			ELSE false
		END AS is_refresh_valid
	FROM "deploy-io".users u
	WHERE u.id = $1`

	err := config.DataBase.QueryRow(query, userId).Scan(&isAccessValid, &isRefreshValid)
	if err != nil {
		println("[AUTH] ", err.Error())
		return false, false
	}

	return isAccessValid, isRefreshValid
}

func generateJWT(userId int64) string {
	tokenAuth := GetJWTAuthConfig()

	_, tokenStr, _ := tokenAuth.Encode(map[string]interface{}{"uId": userId}) //,"exp": time.Now().AddDate(0, 1, 0)})

	return tokenStr
}

func GetJWTAuthConfig() *jwtauth.JWTAuth {
	jwtSecret := os.Getenv("JWT_SECRET")

	if len(strings.TrimSpace(jwtSecret)) == 0 {
		log.Fatalln("[AUTH] JWT Secret was not recognized.")
	}

	return jwtauth.New("HS256", []byte(os.Getenv("JWT_SECRET")), nil)
}

func FetchUserInfoFromGitHub(accessToken string) (*GhUserNameResponse, error) {
	var GhUserInfoResponse GhUserNameResponse

	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}

	resp, err := utils.Request("GET", "https://api.github.com/user", &headers, nil, nil)
	if err != nil {
		fmt.Println("[AUTH] Error while calling GH's user API")
		return nil, err
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("[AUTH] Error while reading GH's user API response")
		return nil, err
	}

	err = json.Unmarshal(respBody, &GhUserInfoResponse)
	if err != nil {
		fmt.Println("[AUTH] Error while reading GH's user API response")
		return nil, err
	}

	return &GhUserInfoResponse, nil
}

func UpdateUserTokens(response GH_UAT_API_Response, id int64) bool {
	result, err := config.DataBase.Exec("UPDATE \"deploy-io\".users SET access = $1, refresh = $2, access_expires_by = $3, refresh_expires_by = $4 WHERE id = $5", response.AccessToken, response.RefreshToken, time.Now().Add(response.AccessTokenExpiresIn*time.Second), time.Now().Add(response.RefreshTokenExpiresIn*time.Second), id)

	if err != nil {
		return false
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil || rowsAffected == 0 {
		return false
	}

	return true
}

func InsertNewUser(user User) (*int64, error) {
	var userId int64
	err := config.DataBase.QueryRow("INSERT INTO \"deploy-io\".users(email, name, access, refresh, access_expires_by, refresh_expires_by) VALUES($1, $2, $3, $4, $5, $6) RETURNING id", user.Email, user.Name, user.Access, user.Refresh, time.Now().Add(user.Access_expires_by*time.Second), time.Now().Add(user.Refresh_expires_by*time.Second)).Scan(&userId)
	if err != nil {
		return nil, err
	}

	return &userId, nil
}

func DoesUserExists(emailId string) (*int64, bool) {
	var user User

	err := config.DataBase.QueryRow("SELECT id FROM \"deploy-io\".users WHERE email = $1 ", emailId).Scan(&user.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false
		}

		fmt.Println("[AUTH] Error while retrieving user data")
		return nil, false
	}

	return &user.Id, true
}

func getUserEmail(accessToken string) (*string, error) {
	var GhUserEmailResponse []EmailObject

	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}

	resp, err := utils.Request("GET", "https://api.github.com/user/emails", &headers, nil, nil)
	if err != nil {
		fmt.Println("[AUTH] Error while calling GH's email API")
		return nil, err
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("[AUTH] Error while reading GH's email API response.")
		return nil, err
	}

	err = json.Unmarshal(respBody, &GhUserEmailResponse)
	if err != nil {
		fmt.Println("[AUTH] Error while parsing GH's email API response.")
		return nil, err
	}

	var userEmail string

	for _, info := range GhUserEmailResponse {
		if info.Primary {
			userEmail = info.Email
		}
	}

	return &userEmail, nil
}

func GetClientIdnSecret() (string, string) {
	clientId := os.Getenv("GH_CLIENT_ID")
	clientSecret := os.Getenv("GH_CLIENT_SECRET")

	if len(strings.TrimSpace(clientId)) == 0 || len(strings.TrimSpace(clientSecret)) == 0 {
		log.Fatalln("[AUTH] Github Client ID or Secret was not recognized.")
	}

	return clientId, clientSecret
}
