package github

import (
	"encoding/json"
	"fmt"
	auth "httpServer/src/routes/Auth"
	"httpServer/utils"
	"io"
	"net/http"
)

func (gHandler GithubHandler) ListUserRepositories(w http.ResponseWriter, r *http.Request) {
	userId := utils.GetUserIdFromContext(w, r)

	if userId == nil {
		return
	}

	repositories, err := getUserRepositories(*userId)
	if err != nil {
		utils.HandleError(utils.ErrInternal, err, w, nil)
		return
	}

	body, err := json.Marshal(repositories)
	if err != nil {
		utils.HandleError(utils.ErrInternal, err, w, nil)
		return
	}

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

func GetGithubURL(githubId int, userId int) (string, error) {
	accessToken, accessTokenErr := auth.GetAccessToken(userId)
	if accessTokenErr != nil || accessToken == nil {
		return "", accessTokenErr
	}

	headers := map[string]string{
		"Authorization": "Bearer " + *accessToken,
	}

	repoResp, repoErr := utils.Request("GET", fmt.Sprintf("https://api.github.com/repositories/%d", githubId), &headers, nil, nil)
	if repoErr != nil {
		return "", repoErr
	}

	defer repoResp.Body.Close()

	repoBody, repoReadErr := io.ReadAll(repoResp.Body)
	if repoReadErr != nil {
		return "", repoErr
	}

	type RepoResponse struct {
		Url string `json:"url"`
	}

	var Response RepoResponse

	deconstructorErr := json.Unmarshal(repoBody, &Response)
	if deconstructorErr != nil {
		return "", deconstructorErr
	}

	return Response.Url, nil
}
