package auth

import (
	"buildServer/config"
	"buildServer/utils"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

type OauthPayload struct {
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

type GhError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorURI         string `json:"error_uri"`
}

func GetAccessToken(userId int) (string, error) {
	isAccessValid, isRefreshValid := areTokensValid(userId)

	var accessToken string
	var fetchErr error

	if isAccessValid && isRefreshValid {
		accessToken, fetchErr = fetchTokensFromDB(userId)
		if fetchErr != nil {
			return "", fetchErr
		}
	} else if isRefreshValid {
		var TokenPayload OauthPayload
		var refreshToken string

		query := `SELECT refresh FROM "deploy-io".users WHERE id = $1`
		err := config.DataBase.QueryRow(query, userId).Scan(&refreshToken)
		if err != nil {
			errMsg := "[USER] Error while fetching refresh token"
			return "", fmt.Errorf(errMsg)
		}

		TokenPayload.RefreshToken = refreshToken
		cId, cSecret := getClientIdnSecret()

		response, err := getOauthResponse(cId, cSecret, TokenPayload)
		if err != nil {
			errMsg := "[USER] Refresh id invalid"
			return "", fmt.Errorf(errMsg)
		}

		updateUserTokens(response, int64(userId))

		accessToken = response.AccessToken
	}

	return accessToken, nil
}

func updateUserTokens(response GH_UAT_API_Response, id int64) bool {
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

func getOauthResponse(cId string, cSecret string, user OauthPayload) (GH_UAT_API_Response, error) {

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

func areTokensValid(userId int) (bool, bool) {
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

func fetchTokensFromDB(userId int) (string, error) {
	query := `SELECT u.access FROM "deploy-io".users u WHERE u.id = $1`

	var accessToken string

	err := config.DataBase.QueryRow(query, userId).Scan(&accessToken)
	if err != nil {
		return "", err
	}

	return accessToken, nil
}

func getClientIdnSecret() (string, string) {
	clientId := os.Getenv("GH_CLIENT_ID")
	clientSecret := os.Getenv("GH_CLIENT_SECRET")

	if len(strings.TrimSpace(clientId)) == 0 || len(strings.TrimSpace(clientSecret)) == 0 {
		log.Fatalln("[AUTH] Github Client ID or Secret was not recognized.")
	}

	return clientId, clientSecret
}
