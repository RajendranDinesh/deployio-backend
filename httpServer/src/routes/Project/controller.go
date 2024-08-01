package project

import (
	"encoding/json"
	"fmt"
	"httpServer/config"
	auth "httpServer/src/routes/Auth"
	"httpServer/utils"
	"io"
	"net/http"
	"os"
	"strings"
)

func (a ProjectHandler) CreateNewProject(w http.ResponseWriter, r *http.Request) {

	body, readBodyErr := io.ReadAll(r.Body)
	if readBodyErr != nil {
		utils.HandleError(utils.ErrInvalid, readBodyErr, w, nil)
		return
	}

	var project Project

	jsonDestructErr := json.Unmarshal(body, &project)
	if jsonDestructErr != nil {
		utils.HandleError(utils.ErrInvalid, jsonDestructErr, w, nil)
		return
	}

	if len(strings.TrimSpace(project.Name)) == 0 {
		utils.HandleError(utils.ErrInvalid, nil, w, nil)
		return
	}

	buildCommand, outputFolder := getDefaultBuildCommandAndOpFolder()

	if project.BuildCommand == nil {
		project.BuildCommand = &buildCommand
	}

	if project.OutputFolder == nil {
		project.OutputFolder = &outputFolder
	}

	userId := utils.GetUserIdFromContext(w, r)
	accessToken, err := auth.GetAccessToken(*userId)
	if err != nil {
		errMsg := "[PROJECT] Error while reading user access token"
		utils.HandleError(utils.ErrInternal, err, w, &errMsg)
		return
	}

	url, cloneErr := getRepositoryCloneURL(project.GithubId, *accessToken)
	if cloneErr != nil {
		utils.HandleError(utils.ErrInternal, cloneErr, w, nil)
		return
	}

	projectId, dbErr := insertProjectIntoDB(*userId, project.Name, *url, *project.BuildCommand, *project.OutputFolder)
	if dbErr != nil {

		if strings.Contains(dbErr.Error(), "duplicate key") {
			utils.HandleError(utils.ErrAlreadyExists, dbErr, w, nil)
			return
		}

		utils.HandleError(utils.ErrInternal, dbErr, w, nil)
		return
	}

	responseBody := map[string]int{
		"project_id": int(*projectId),
	}

	response, constructerErr := json.Marshal(responseBody)
	if constructerErr != nil {
		utils.HandleError(utils.ErrInternal, constructerErr, w, nil)
		return
	}

	w.Write([]byte(response))
}

func insertProjectIntoDB(userId int, name string, url string, buildCommand string, outputFolder string) (*int, error) {
	var projectId int
	query := "INSERT INTO \"deploy-io\".projects (user_id, name, clone_url, build_command, output_folder) VALUES($1, $2, $3, $4, $5) RETURNING id"
	err := config.DataBase.QueryRow(query, userId, name, url, buildCommand, outputFolder).Scan(&projectId)
	if err != nil {
		return nil, err
	}

	return &projectId, nil
}

func getRepositoryCloneURL(repoId int, accessToken string) (*string, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}

	response, err := utils.Request("GET", fmt.Sprintf("https://api.github.com/repositories/%d", repoId), &headers, nil, nil)
	if err != nil {
		fmt.Println("[PROJECT] Error while calling GH's repositories API")
		return nil, err
	}

	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("[PROJECT] Error while reading GH's repositories API response body")
		return nil, err
	}

	var cloneURL RepositoriesAPIResponse

	unMarshalerErr := json.Unmarshal(responseBody, &cloneURL)
	if unMarshalerErr != nil {
		fmt.Println("[PROJECT] Error while constructing clone url from GH's repositories API response body")
		return nil, unMarshalerErr
	}

	return &cloneURL.CloneURL, nil
}

func getDefaultBuildCommandAndOpFolder() (string, string) {
	var buildCommand, outputFolder string

	buildCommand, buildCommandExists := os.LookupEnv("BUILD_COMMAND")
	outputFolder, outputFolderExists := os.LookupEnv("BUILD_DIRECTORY")

	if !buildCommandExists || !outputFolderExists {
		return "npm run build", "./dist"
	}

	return buildCommand, outputFolder
}
