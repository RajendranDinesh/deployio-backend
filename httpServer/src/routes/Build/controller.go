package build

import (
	"context"
	"encoding/json"
	"fmt"
	"httpServer/config"
	auth "httpServer/src/routes/Auth"
	"httpServer/utils"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rabbitmq/amqp091-go"
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

	githubId, dbErr := getGithubId(requestBody.ProjectId, *userId)
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

	if len(commitSha) <= 0 {
		utils.HandleError(utils.ErrInvalid, fmt.Errorf("[BUILD] Sha doesn't exists"), w, nil)
		return
	}

	buildId, buildInsertErr := insertIntoDB(requestBody.ProjectId, commitSha)
	if buildInsertErr != nil || buildId == nil {
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

	buildCtx := context.Background()

	queueErr := config.RabbitChannel.PublishWithContext(buildCtx, "", config.RabbitQueue.Name, false, false, amqp091.Publishing{
		ContentType: "application/octet-stream",
		Body:        []byte(responseBody),
	})
	if queueErr != nil {
		UpdateBuildLog(*buildId, queueErr.Error())
		SetBuildStatus(*buildId, "failure")
		utils.HandleError(utils.ErrInternal, queueErr, w, nil)
		return
	}

	log.Printf("[rabbitMQ] sent %s", responseBody)

	w.Write(responseBody)
}

func insertIntoDB(projectId int, commitSha string) (*int, error) {
	var buildId int

	insertQuery := `
		INSERT INTO "deploy-io".builds(project_id, status, triggered_by, commit_hash)
		VALUES($1, 'in queue', 'manual', $2) RETURNING id
	`
	insertErr := config.DataBase.QueryRow(insertQuery, projectId, commitSha).Scan(&buildId)
	if insertErr != nil {
		return nil, insertErr
	}

	return &buildId, nil
}

func getGithubId(projectId int, userId int) (*int, error) {
	var githubId int

	searchQuery := `SELECT github_id FROM "deploy-io".projects p WHERE p.id = $1 AND p.user_id = $2`

	searchErr := config.DataBase.QueryRow(searchQuery, projectId, userId).Scan(&githubId)
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

	repoResponse, repoErr := utils.Request("GET", fmt.Sprintf("https://api.github.com/repositories/%d", githubId), &header, nil, nil)
	if repoErr != nil {
		return "", repoErr
	}

	defer repoResponse.Body.Close()

	repoResponseBody, readErr := io.ReadAll(repoResponse.Body)
	if readErr != nil {
		return "", readErr
	}

	var repoAPIResponse RepoAPIResponse

	deconstructorErr := json.Unmarshal(repoResponseBody, &repoAPIResponse)
	if deconstructorErr != nil {
		return "", deconstructorErr
	}

	commitsURL := repoAPIResponse.CommitsURL

	if len(commitsURL) <= 0 {
		return "", fmt.Errorf("[BUILD] Could not get commits url")
	}

	commitsURL = strings.ReplaceAll(repoAPIResponse.CommitsURL, "{/sha}", "")

	shaResponse, shaErr := utils.Request("GET", commitsURL, &header, nil, nil)
	if shaErr != nil {
		return "", repoErr
	}

	defer shaResponse.Body.Close()

	shaResponseBody, shaReadErr := io.ReadAll(shaResponse.Body)
	if shaReadErr != nil {
		return "", readErr
	}

	var shaAPIResponse []CommitObject

	shaDeconstructorErr := json.Unmarshal(shaResponseBody, &shaAPIResponse)
	if shaDeconstructorErr != nil {
		return "", deconstructorErr
	}

	if len(shaAPIResponse) <= 0 {
		return "", fmt.Errorf("[BUILD] No commits have been made")
	}

	return shaAPIResponse[0].Sha, nil
}

func (b BuildHandler) ListBuilds(w http.ResponseWriter, r *http.Request) {
	projectId := chi.URLParam(r, "id")

	userId := utils.GetUserIdFromContext(w, r)
	if userId == nil {
		utils.HandleError(utils.TokenExpired, nil, w, nil)
		return
	}

	var listBuilds []Build

	listBuildQuery := `SELECT b.id, b.status, b.triggered_by, b.commit_hash, b.created_at FROM "deploy-io".builds b
		JOIN "deploy-io".projects p ON p.id = b.project_id
		WHERE b.project_id = $1 AND p.user_id = $2 ORDER BY b.id DESC;
	`
	builds, rowsErr := config.DataBase.Query(listBuildQuery, projectId, userId)
	if rowsErr != nil {
		utils.HandleError(utils.ErrInternal, rowsErr, w, nil)
		return
	}

	for builds.Next() {
		var build Build
		builds.Scan(&build.Id, &build.Build_status, &build.Triggered_by, &build.Commit_hash, &build.Created_at)

		listBuilds = append(listBuilds, build)
	}

	response := map[string][]Build{
		"builds": listBuilds,
	}

	responseBody, responseErr := json.Marshal(response)
	if responseErr != nil {
		utils.HandleError(utils.ErrInternal, responseErr, w, nil)
		return
	}

	w.Write(responseBody)
}

func (b BuildHandler) Build(w http.ResponseWriter, r *http.Request) {
	buildId := chi.URLParam(r, "id")

	userId := utils.GetUserIdFromContext(w, r)
	if userId == nil {
		utils.HandleError(utils.TokenExpired, nil, w, nil)
		return
	}

	var build Build

	buildQuery := `SELECT id, status, triggered_by, commit_hash, logs, start_time, end_time, created_at, updated_at FROM "deploy-io".builds b WHERE b.id = $1`
	rowsErr := config.DataBase.QueryRow(buildQuery, buildId).Scan(&build.Id, &build.Build_status, &build.Triggered_by, &build.Commit_hash, &build.Build_logs, &build.Start_time, &build.End_time, &build.Created_at, &build.Updated_at)
	if rowsErr != nil {
		if strings.Contains(rowsErr.Error(), "no rows in result set") {
			w.WriteHeader(404)
			w.Write([]byte(`not found`))
			return
		}

		utils.HandleError(utils.ErrInternal, rowsErr, w, nil)
		return
	}

	responseBody, responseErr := json.Marshal(build)
	if responseErr != nil {
		utils.HandleError(utils.ErrInternal, responseErr, w, nil)
		return
	}

	w.Write(responseBody)
}

func UpdateBuildLog(buildId int, log string) error {
	query := `UPDATE "deploy-io".builds SET logs = COALESCE(logs || E'\n', '') || '$1', end_time = $2 WHERE id = $3`
	_, queErr := config.DataBase.Exec(query, log, time.Now(), buildId)
	if queErr != nil {
		return queErr
	}

	return nil
}

func SetBuildStatus(buildId int, status string) error {
	query := `UPDATE "deploy-io".builds SET status = '$1' WHERE id = $2`
	_, queErr := config.DataBase.Exec(query, status, buildId)
	if queErr != nil {
		return queErr
	}

	return nil
}
