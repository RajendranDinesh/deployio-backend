package utils

import (
	"bytes"
	"net/http"
	u "net/url"
)

func Request(method, url string, headers, params *map[string]string, body *[]byte) (*http.Response, error) {
	client := &http.Client{}

	if params != nil && len(*params) > 0 {
		parameters := u.Values{}

		for key, value := range *params {
			parameters.Add(key, value)
		}

		url = url + "?" + parameters.Encode()
	}

	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(*body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("Accept", "application/json")

	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	req.Header.Set("User-Agent", "deployio-app")

	if headers != nil && len(*headers) > 0 {
		for key, value := range *headers {
			req.Header.Set(key, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
