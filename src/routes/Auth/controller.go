package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func GetAuth(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("auth root"))
}

func (u UserHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	var user UserSignInPayload

	// gets Code from body and stores it into user
	err := json.NewDecoder(r.Body).Decode(&user)

	if err != nil {
		ErrInternalServer(err, w)
	}

	w.Write([]byte(getUserEmail("")))

	// cId, cSecret := getClientIdnSecret()

	// response, err := getAccessTokens(cId, cSecret, user)

	// if err != nil {
	// 	ErrInvalid(err, w)
	// 	return
	// }

	// println(response.AccessToken)

	// w.Write([]byte(response.AccessToken))
}

func getAccessTokens(cId string, cSecret string, user UserSignInPayload) (GH_UAT_API_Response, error) {

	var GHAPIResponse GH_UAT_API_Response
	var GhAPIError GhError

	// creating params to call the github api to get personal access token
	params := url.Values{}

	params.Add("client_id", cId)
	params.Add("client_secret", cSecret)
	params.Add("code", user.Code)

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token"+"?"+params.Encode(), nil)

	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("[AUTH] Error while constructing GH's UAT API")
		return GHAPIResponse, err
	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("Accept", "application/json")

	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	req.Header.Set("User-Agent", "deployio-app")

	client := &http.Client{}

	// request is sent from here to github
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println(err.Error())
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

func getUserEmail(accessToken string) string {
	var GhUserInfoResponse GhUserInfoResponse

	req, err := http.NewRequest("POST", "https://api.github.com/user", nil)

	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("[AUTH] Error while constructing GH's user API")
	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("Accept", "application/json")

	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	req.Header.Set("User-Agent", "deployio-app")

	req.Header.Set("Authorization", accessToken)

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("[AUTH] Error while calling GH's user API")
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)

	if err != nil {
		fmt.Println("[AUTH] Error while reading GH's user API response.")
	}

	err = json.Unmarshal(respBody, &GhUserInfoResponse)

	if err != nil {
		// return GH
	}

	return GhUserInfoResponse.Email

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
