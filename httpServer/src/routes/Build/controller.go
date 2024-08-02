package build

import (
	"encoding/json"
	"fmt"
	"httpServer/config"
	auth "httpServer/src/routes/Auth"
	"httpServer/utils"
	"io"
	"net/http"
)

func (b BuildHandler) CreateBuild(w http.ResponseWriter, r *http.Request) {
	body, readBodyErr := io.ReadAll(r.Body)
	if readBodyErr != nil {
		utils.HandleError(utils.ErrInvalid, readBodyErr, w, nil)
		return
	}

	var requestBody InsertBuildBody

	jsonDestructErr := json.Unmarshal(body, &requestBody)
	if jsonDestructErr != nil {
		utils.HandleError(utils.ErrInvalid, jsonDestructErr, w, nil)
		return
	}

	userId := utils.GetUserIdFromContext(w, r)
	if userId == nil {
		utils.HandleError(utils.TokenExpired, nil, w, nil)
		return
	}

	githubId, dbErr := getGithubId(requestBody.ProjectId)
	if dbErr != nil {
		utils.HandleError(utils.ErrInvalid, dbErr, w, nil)
		return
	}

	commitSha, shaErr := getCommitSha(*githubId, *userId)
	if shaErr != nil {
		errString := "[BUILD] error while requesting commit sha"
		utils.HandleError(utils.ErrInvalid, shaErr, w, &errString)
		return
	}

	buildId, buildInsertErr := insertIntoDB(requestBody.ProjectId, commitSha)
	if buildInsertErr != nil {
		utils.HandleError(utils.ErrInvalid, buildInsertErr, w, nil)
		return
	}

	response := map[string]int{
		"build_id": *buildId,
	}

	responseBody, constructorErr := json.Marshal(response)
	if constructorErr != nil {
		utils.HandleError(utils.ErrInternal, constructorErr, w, nil)
		return
	}

	w.Write(responseBody)
}

func insertIntoDB(projectId int, commitSha string) (*int, error) {
	var buildId int

	insertQuery := `
		INSERT INTO "deploy-io".builds(project_id, build_status, triggered_by, commit_hash)
		VALUES($1, 'in queue', 'manual', $2) RETURNING id
	`
	insertErr := config.DataBase.QueryRow(insertQuery, projectId, commitSha).Scan(&buildId)
	if insertErr != nil {
		return nil, insertErr
	}

	return &buildId, nil
}

func getGithubId(projectId int) (*int, error) {
	var githubId int

	searchQuery := `SELECT github_id FROM "deploy-io".projects p WHERE p.id = $1`

	searchErr := config.DataBase.QueryRow(searchQuery, projectId).Scan(&githubId)
	if searchErr != nil {
		return nil, searchErr
	}

	return &githubId, nil
}

func getCommitSha(githubId int, userId int) (string, error) {
	accessToken, accessTokenErr := auth.GetAccessToken(userId)
	if accessTokenErr != nil {
		return "", accessTokenErr
	}

	header := map[string]string{
		"Authorization": `Bearer ` + *accessToken,
	}

	response, repoErr := utils.Request("GET", fmt.Sprintf("https://api.github.com/repositories/%d", githubId), &header, nil, nil)
	if repoErr != nil {
		return "", repoErr
	}

	defer response.Body.Close()

	responseBody, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return "", readErr
	}

	var RepoAPIResponse RepoAPIResponse

	deconstructorErr := json.Unmarshal(responseBody, &RepoAPIResponse)
	if deconstructorErr != nil {
		return "", deconstructorErr
	}

	// todo
	// call the commit url and return the sha of the first objects

	return "", nil
}

func (b BuildHandler) ListBuilds(w http.ResponseWriter, r *http.Request) {

}

func (b BuildHandler) Build(w http.ResponseWriter, r *http.Request) {

}
