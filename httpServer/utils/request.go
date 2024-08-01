package utils

import (
	"bytes"
	"fmt"
	"net/http"
	u "net/url"
	"strconv"

	"github.com/go-chi/jwtauth/v5"
)

func Request(method, url string, headers, params *map[string]string, body *[]byte) (*http.Response, error) {
	client := &http.Client{}

	url = addParamsToURL(url, params)

	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(*body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		fmt.Println("[REQUEST] Error building request")
		return nil, err
	}

	addHeadersToRequest(req, headers)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func addParamsToURL(url string, params *map[string]string) string {
	if params != nil && len(*params) > 0 {
		parameters := u.Values{}

		for key, value := range *params {
			parameters.Add(key, value)
		}

		url = url + "?" + parameters.Encode()
	}

	return url
}

func addHeadersToRequest(req *http.Request, headers *map[string]string) {
	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("Accept", "application/json")

	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	req.Header.Set("User-Agent", "deployio-app")

	if headers != nil && len(*headers) > 0 {
		for key, value := range *headers {
			req.Header.Set(key, value)
		}
	}
}

func GetUserIdFromContext(w http.ResponseWriter, r *http.Request) *int {
	_, claims, _ := jwtauth.FromContext(r.Context())

	userId, err := strconv.Atoi(fmt.Sprintf("%v", claims["uId"]))

	if err != nil {
		errMsg := "[REQUEST] Error Converting string to integer"
		HandleError(ErrInvalid, err, w, &errMsg)
		return nil
	}

	return &userId
}
