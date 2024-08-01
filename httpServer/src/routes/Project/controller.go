package project

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"httpServer/config"
	"httpServer/utils"
	"io"
	"net/http"
	"os"
	"strings"
)

func (p ProjectHandler) InsertEnvironments(w http.ResponseWriter, r *http.Request) {
	body, readBodyErr := io.ReadAll(r.Body)
	if readBodyErr != nil {
		utils.HandleError(utils.ErrInvalid, readBodyErr, w, nil)
		return
	}

	var requestBody InsertEnvironmentBody

	jsonDestructErr := json.Unmarshal(body, &requestBody)
	if jsonDestructErr != nil {
		utils.HandleError(utils.ErrInvalid, jsonDestructErr, w, nil)
		return
	}

	insertQuery := `INSERT INTO "deploy-io".environments(project_id, key, value) VALUES($1, $2, $3)`
	insertStatement, preparationErr := config.DataBase.Prepare(insertQuery)
	if preparationErr != nil {
		utils.HandleError(utils.ErrInternal, preparationErr, w, nil)
		return
	}

	defer insertStatement.Close()

	for _, environment := range requestBody.Environments {
		val, valErr := encrypt(environment.Value)

		if valErr != nil {
			utils.HandleError(utils.ErrInternal, valErr, w, nil)
			return
		}

		_, insertErr := insertStatement.Exec(environment.ProjectId, environment.Key, val)
		if insertErr != nil {
			utils.HandleError(utils.ErrInternal, insertErr, w, nil)
			return
		}
	}

	responseBody := map[string]string{
		"msg": "Inserted environments",
	}

	response, constructorErr := json.Marshal(responseBody)
	if constructorErr != nil {
		utils.HandleError(utils.ErrInternal, constructorErr, w, nil)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func (p ProjectHandler) ListEnvKeys(w http.ResponseWriter, r *http.Request) {
	body, readBodyErr := io.ReadAll(r.Body)
	if readBodyErr != nil {
		utils.HandleError(utils.ErrInvalid, readBodyErr, w, nil)
		return
	}

	userId := utils.GetUserIdFromContext(w, r)
	if userId == nil {
		utils.HandleError(utils.TokenExpired, nil, w, nil)
		return
	}

	var requestBody ListEnvKeysBody
	jsonDestructErr := json.Unmarshal(body, &requestBody)
	if jsonDestructErr != nil {
		utils.HandleError(utils.ErrInvalid, jsonDestructErr, w, nil)
		return
	}

	query := `SELECT e.key FROM "deploy-io".environments e JOIN "deploy-io".projects p ON p.id = e.project_id AND p.id = $1 AND p.user_id = $2`
	rows, queryErr := config.DataBase.Query(query, requestBody.ProjectId, userId)
	if queryErr != nil {
		utils.HandleError(utils.ErrInvalid, queryErr, w, nil)
		return
	}

	defer rows.Close()

	var envKeys []string

	for rows.Next() {
		var env string
		rowsErr := rows.Scan(&env)
		if rowsErr != nil {
			utils.HandleError(utils.ErrInternal, rowsErr, w, nil)
			return
		}

		envKeys = append(envKeys, env)
	}

	responseBody := map[string][]string{
		"keys": envKeys,
	}

	response, constructorErr := json.Marshal(responseBody)
	if constructorErr != nil {
		utils.HandleError(utils.ErrInternal, constructorErr, w, nil)
		return
	}

	w.Write(response)
}

func (p ProjectHandler) UpdateEnvValue(w http.ResponseWriter, r *http.Request) {
	body, readBodyErr := io.ReadAll(r.Body)
	if readBodyErr != nil {
		utils.HandleError(utils.ErrInvalid, readBodyErr, w, nil)
		return
	}

	var requestBody UpdateEnvironmentBody

	jsonDestructErr := json.Unmarshal(body, &requestBody)
	if jsonDestructErr != nil {
		utils.HandleError(utils.ErrInvalid, jsonDestructErr, w, nil)
		return
	}

	encryptedValue, encErr := encrypt(requestBody.Value)
	if encErr != nil {
		utils.HandleError(utils.ErrInvalid, encErr, w, nil)
		return
	}

	updateQuery := `UPDATE "deploy-io".environments SET value = $1 WHERE project_id = $2 AND key = $3`
	_, updateErr := config.DataBase.Exec(updateQuery, encryptedValue, requestBody.ProjectId, requestBody.Key)
	if updateErr != nil {
		utils.HandleError(utils.ErrInternal, updateErr, w, nil)
		return
	}

	responseBody := map[string]string{
		"msg": "Updated environment variable",
	}

	response, constructorErr := json.Marshal(responseBody)
	if constructorErr != nil {
		utils.HandleError(utils.ErrInternal, constructorErr, w, nil)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

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

func encrypt(text string) (string, error) {
	key, keyExists := os.LookupEnv("ENV_SECRET")
	if !keyExists {
		return "", fmt.Errorf("[ENC] env secret is not accessible")
	}

	keyBytes := []byte(key)

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	cipherText := make([]byte, aes.BlockSize+len(text))
	iv := cipherText[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], []byte(text))

	return hex.EncodeToString(cipherText), nil
}