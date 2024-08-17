package project

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"httpServer/config"
	github "httpServer/src/routes/Github"
	"httpServer/utils"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (p ProjectHandler) Project(w http.ResponseWriter, r *http.Request) {
	projectId := chi.URLParam(r, "id")

	userId := utils.GetUserIdFromContext(w, r)
	if userId == nil {
		utils.HandleError(utils.TokenExpired, nil, w, nil)
		return
	}

	query := `SELECT
			name, directory, node_version,
			install_command, build_command, output_folder,
			github_id
	FROM "deploy-io".projects p WHERE p.id = $1 AND p.user_id = $2`

	type ResponseBody struct {
		Name           string `json:"name"`
		Directory      string `json:"directory"`
		NodeVersion    string `json:"node_version"`
		InstallCommand string `json:"install_command"`
		BuildCommand   string `json:"build_command"`
		OutputFolder   string `json:"output_folder"`
		GithubURL      string `json:"github_url"`
	}

	type tempBody struct {
		GithubId int
	}

	var response ResponseBody
	var temp tempBody

	err := config.DataBase.QueryRow(query, projectId, &userId).Scan(&response.Name,
		&response.Directory, &response.NodeVersion, &response.InstallCommand,
		&response.BuildCommand, &response.OutputFolder, &temp.GithubId)
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			w.WriteHeader(404)
			w.Write([]byte(`not found`))
			return
		}

		utils.HandleError(utils.ErrInternal, err, w, nil)
		return
	}

	githubURL, fetchGHURLErr := github.GetGithubURL(temp.GithubId, *userId)
	if fetchGHURLErr != nil {
		utils.HandleError(utils.ErrInternal, fetchGHURLErr, w, nil)
		return
	}

	response.GithubURL = githubURL

	responseBody, jsonConstructorErr := json.Marshal(response)
	if jsonConstructorErr != nil {
		utils.HandleError(utils.ErrInternal, jsonConstructorErr, w, nil)
		return
	}

	w.Write(responseBody)
}

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

		_, insertErr := insertStatement.Exec(requestBody.ProjectId, environment.Key, val)
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

	query := `SELECT p.id, p.name, p.install_command,
		p.build_command, p.output_folder, p.created_at,
		p.directory, p.node_version, 
		COALESCE (BOOL_OR(d.status), FALSE) AS is_active
		FROM "deploy-io".projects p LEFT JOIN "deploy-io".deployments d ON d.project_id = p.id
		WHERE p.user_id = $1 GROUP BY p.id;
	`

	rows, queryErr := config.DataBase.Query(query, userId)
	if queryErr != nil {
		utils.HandleError(utils.ErrInvalid, queryErr, w, nil)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var project ListProject
		rowsErr := rows.Scan(&project.Id, &project.Name, &project.InstallCommand, &project.BuildCommand, &project.OutputFolder, &project.CreatedAt, &project.Directory, &project.NodeVersion, &project.IsActive)
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

	installCommand, buildCommand, outputFolder, nodeVersion, directory := getDefaults()

	if project.InstallCommand == nil {
		project.InstallCommand = &installCommand
	}

	if project.BuildCommand == nil || len(strings.TrimSpace(*project.BuildCommand)) == 0 {
		project.BuildCommand = &buildCommand
	}

	if project.OutputFolder == nil || len(strings.TrimSpace(*project.OutputFolder)) == 0 {
		project.OutputFolder = &outputFolder
	}

	if project.NodeVersion == nil || len(strings.TrimSpace(*project.NodeVersion)) == 0 {
		project.NodeVersion = &nodeVersion
	}

	if project.Directory == nil || len(strings.TrimSpace(*project.Directory)) == 0 {
		project.Directory = &directory
	}

	userId := utils.GetUserIdFromContext(w, r)

	githubId, convErr := strconv.Atoi(project.GithubId)
	if convErr != nil {
		utils.HandleError(utils.ErrInvalid, convErr, w, nil)
		return
	}

	projectId, dbErr := insertProjectIntoDB(*userId, project.Name, githubId, *project.InstallCommand, *project.BuildCommand, removeLeadingAndTrailingSlashes(*project.OutputFolder), *project.NodeVersion, removeLeadingAndTrailingSlashes(*project.Directory))
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

func insertProjectIntoDB(userId int, name string, githubId int, installCommand string, buildCommand string, outputFolder string, nodeVersion string, directory string) (*int, error) {
	var projectId int
	query := "INSERT INTO \"deploy-io\".projects (user_id, name, github_id, install_command, build_command, output_folder, node_version, directory) VALUES($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id"
	err := config.DataBase.QueryRow(query, userId, name, githubId, installCommand, buildCommand, outputFolder, nodeVersion, directory).Scan(&projectId)
	if err != nil {
		return nil, err
	}

	return &projectId, nil
}

func getDefaults() (string, string, string, string, string) {
	var buildCommand, outputFolder string

	installCommand, installCommandExists := os.LookupEnv("DEFAULT_INSTALL_COMMAND")
	buildCommand, buildCommandExists := os.LookupEnv("DEFAULT_BUILD_COMMAND")
	outputFolder, outputFolderExists := os.LookupEnv("DEFAULT_BUILD_DIRECTORY")
	directory, directoryExists := os.LookupEnv("DEFAULT_PROJECT_DIRECTORY")
	nodeVersion, nodeVersionExists := os.LookupEnv("DEFAULT_NODE_VERSION")

	if !buildCommandExists || !outputFolderExists || !installCommandExists || !nodeVersionExists || !directoryExists {
		return "npm install", "npm run build", "./dist", "20", "./"
	}

	return installCommand, buildCommand, outputFolder, nodeVersion, directory
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

func removeLeadingAndTrailingSlashes(input string) string {
	input = strings.TrimLeft(input, "./")
	input = strings.TrimRight(input, "/")
	return input
}
