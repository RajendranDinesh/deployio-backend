package project

import (
	"encoding/json"
	"httpServer/config"
	"httpServer/utils"
	"io"
	"net/http"
	"os"
	"strings"
)

func (p ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	userId := utils.GetUserIdFromContext(w, r)
	if userId == nil {
		utils.HandleError(utils.TokenExpired, nil, w, nil)
		return
	}

	var projects []ListProject

	query := `SELECT id, name, github_id, build_command, output_folder, created_at FROM "deploy-io".projects WHERE user_id = $1`
	rows, queryErr := config.DataBase.Query(query, userId)
	if queryErr != nil {
		utils.HandleError(utils.ErrInvalid, queryErr, w, nil)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var project ListProject
		rowsErr := rows.Scan(&project.Id, &project.Name, &project.GithubId, &project.BuildCommand, &project.OutputFolder, &project.CreatedAt)
		if rowsErr != nil {
			utils.HandleError(utils.ErrInternal, rowsErr, w, nil)
			return
		}

		projects = append(projects, project)
	}

	responseBody := map[string][]ListProject{
		"projects": projects,
	}

	response, constructorErr := json.Marshal(responseBody)
	if constructorErr != nil {
		utils.HandleError(utils.ErrInternal, constructorErr, w, nil)
		return
	}

	w.Write(response)
}

func (p ProjectHandler) CreateNewProject(w http.ResponseWriter, r *http.Request) {
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

	projectId, dbErr := insertProjectIntoDB(*userId, project.Name, project.GithubId, *project.BuildCommand, *project.OutputFolder)
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

	response, constructorErr := json.Marshal(responseBody)
	if constructorErr != nil {
		utils.HandleError(utils.ErrInternal, constructorErr, w, nil)
		return
	}

	w.Write([]byte(response))
}

func insertProjectIntoDB(userId int, name string, githubId int, buildCommand string, outputFolder string) (*int, error) {
	var projectId int
	query := "INSERT INTO \"deploy-io\".projects (user_id, name, github_id, build_command, output_folder) VALUES($1, $2, $3, $4, $5) RETURNING id"
	err := config.DataBase.QueryRow(query, userId, name, githubId, buildCommand, outputFolder).Scan(&projectId)
	if err != nil {
		return nil, err
	}

	return &projectId, nil
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
