package github

import (
	"encoding/json"
	"fmt"
	auth "httpServer/src/routes/Auth"
	"httpServer/utils"
	"io"
	"net/http"
)

func (gHandler GithubHandler) GetUserRepositories(w http.ResponseWriter, r *http.Request) {
	userId := utils.GetUserIdFromContext(w, r)

	if userId == nil {
		return
	}

	repositories, err := getUserRepositories(*userId)
	utils.HandleError(utils.ErrInternal, err, w)

	body, err := json.Marshal(repositories)
	utils.HandleError(utils.ErrInternal, err, w)

	w.Write([]byte(body))
}

func getUserRepositories(userId int) (*RepoAPIResponse, error) {

	accessToken, err := auth.GetAccessToken(userId)
	if err != nil {
		fmt.Println("[GITHUB] Error while reading access token")
		return nil, err
	}

	header := map[string]string{
		"Authorization": `Bearer ` + *accessToken,
	}

	resp, err := utils.Request("GET", "https://api.github.com/user/repos", &header, nil, nil)
	if err != nil {
		fmt.Println("[GITHUB] Error while requesting for user repositories")
		return nil, err
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("[GITHUB] Error while reading response from user repositories endpoint")
		return nil, err
	}

	var RepoAPIResponse RepoAPIResponse

	err = json.Unmarshal(respBody, &RepoAPIResponse)
	if err != nil {
		fmt.Println("[GITHUB] Error while reading response from user repositories response")
		return nil, err
	}

	return &RepoAPIResponse, nil
}
