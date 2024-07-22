package auth

import (
	"database/sql"
	"deployio-backend/config"
	"deployio-backend/utils"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/jwtauth/v5"
)

func (u UserHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	var user UserSignInPayload

	// gets Code from body and stores it into user
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		ErrInternalServer(err, w)
		return
	}

	cId, cSecret := getClientIdnSecret()

	response, err := getOauthResponse(cId, cSecret, user)
	if err != nil {
		ErrInvalid(err, w)
		return
	}

	email, err := getUserEmail(response.AccessToken)
	if err != nil {
		ErrInternalServer(err, w)
		return
	}

	var userId *int64

	userId, userExists := DoesUserExists(*email)
	if !userExists && userId == nil {
		userInfo, err := FetchUserInfoFromGitHub(response.AccessToken)

		if err != nil {
			ErrInternalServer(err, w)
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
			ErrInternalServer(err, w)
			return
		}

	} else {
		UpdateUserTokens(response, *userId)
	}

	jwtToken := generateJWT(*userId)

	responseBody, err := json.Marshal(jwtToken)

	if err != nil {
		ErrInternalServer(err, w)
		return
	}

	w.Write([]byte(responseBody))
}

func getOauthResponse(cId string, cSecret string, user UserSignInPayload) (GH_UAT_API_Response, error) {

	var GHAPIResponse GH_UAT_API_Response
	var GhAPIError GhError

	// creating params to call the github api to get personal access token

	params := map[string]string{
		"client_id":     cId,
		"client_secret": cSecret,
		"code":          user.Code,
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

func generateJWT(userId int64) string {
	tokenAuth := getJWTAuthConfig()

	_, token, _ := tokenAuth.Encode(map[string]interface{}{"uId": userId})

	return token
}

func getJWTAuthConfig() *jwtauth.JWTAuth {
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
		fmt.Println("[AUTH] Error while reading GH's user API response.")
		return nil, err
	}

	err = json.Unmarshal(respBody, &GhUserInfoResponse)
	if err != nil {
		fmt.Println("[AUTH] Error while reading GH's user API response.")
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
	result, err := config.DataBase.Exec("INSERT INTO \"deploy-io\".users(email, name, access, refresh, access_expires_by, refresh_expires_by) VALUES($1, $2, $3, $4, $5, $6)", user.Email, user.Name, user.Access, user.Refresh, time.Now().Add(user.Access_expires_by*time.Second), time.Now().Add(user.Refresh_expires_by*time.Second))

	if err != nil {
		return nil, err
	}

	lastInsertId, err := result.RowsAffected()

	if err != nil || lastInsertId == 0 {
		return nil, err
	}

	return &lastInsertId, nil
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

func getClientIdnSecret() (string, string) {
	clientId := os.Getenv("GH_CLIENT_ID")
	clientSecret := os.Getenv("GH_CLIENT_SECRET")

	if len(strings.TrimSpace(clientId)) == 0 || len(strings.TrimSpace(clientSecret)) == 0 {
		log.Fatalln("[AUTH] Github Client ID or Secret was not recognized.")
	}

	return clientId, clientSecret
}

func ErrInternalServer(err error, w http.ResponseWriter) {
	println("[AUTH] ", err.Error())
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

func ErrInvalid(err error, w http.ResponseWriter) {
	http.Error(w, "Invalid Request", http.StatusBadRequest)
}
