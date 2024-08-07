package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"httpServer/config"
	"httpServer/utils"
	"io"
	"net/http"
	"os"

	"github.com/minio/minio-go/v7"
)

func (DeploymentHandler) DeleteDeployment(w http.ResponseWriter, r *http.Request) {
	userId := utils.GetUserIdFromContext(w, r)
	if userId == nil {
		utils.HandleError(utils.TokenExpired, nil, w, nil)
		return
	}

	requestBody, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		utils.HandleError(utils.ErrInternal, readErr, w, nil)
		return
	}

	var Request DeleteDeploymentReqBody
	deconstructorErr := json.Unmarshal(requestBody, &Request)
	if deconstructorErr != nil {
		utils.HandleError(utils.ErrInvalid, deconstructorErr, w, nil)
		return
	}

	var ProjectName string
	var ProjectId int

	query := `SELECT p.name, p.id FROM "deploy-io".projects p JOIN "deploy-io".builds b ON b.project_id = p.id WHERE b.id = $1`
	qErr := config.DataBase.QueryRow(query, Request.BuildId).Scan(&ProjectName, &ProjectId)
	if qErr != nil {
		utils.HandleError(utils.ErrInternal, qErr, w, nil)
		return
	}

	delErr := deleteFiles(ProjectName)
	if delErr != nil {
		utils.HandleError(utils.ErrInternal, delErr, w, nil)
		return
	}

	setFalseQuery := `UPDATE "deploy-io".deployments SET status = false WHERE project_id = $1`
	_, setFalseErr := config.DataBase.Exec(setFalseQuery, ProjectId)
	if setFalseErr != nil {
		utils.HandleError(utils.ErrInternal, setFalseErr, w, nil)
		return
	}

	responseBody := map[string]string{
		"message": "Done",
	}

	response, constructorErr := json.Marshal(responseBody)
	if constructorErr != nil {
		utils.HandleError(utils.ErrInternal, constructorErr, w, nil)
		return
	}

	w.Write(response)
}

func (DeploymentHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	userId := utils.GetUserIdFromContext(w, r)
	if userId == nil {
		utils.HandleError(utils.TokenExpired, nil, w, nil)
		return
	}

	requestBody, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		utils.HandleError(utils.ErrInternal, readErr, w, nil)
		return
	}

	var Request ListDeploymentsReqBody
	deconstructorErr := json.Unmarshal(requestBody, &Request)
	if deconstructorErr != nil {
		utils.HandleError(utils.ErrInvalid, deconstructorErr, w, nil)
		return
	}

	query := `SELECT d.id, d.build_id, d.status, d.created_at FROM "deploy-io".deployments d JOIN "deploy-io".projects p on d.project_id = p.id WHERE d.project_id = $1 AND p.user_id = $2`
	rows, qErr := config.DataBase.Query(query, Request.ProjectId, *userId)
	if qErr != nil {
		utils.HandleError(utils.ErrInternal, qErr, w, nil)
		return
	}

	var Deployments []Deployment

	for rows.Next() {
		var Deployment Deployment

		rows.Scan(&Deployment.Id, &Deployment.BuildId, &Deployment.Status, &Deployment.CreatedAt)

		Deployments = append(Deployments, Deployment)
	}

	response := map[string][]Deployment{
		"deployments": Deployments,
	}

	responseBody, constructorErr := json.Marshal(response)
	if constructorErr != nil {
		utils.HandleError(utils.ErrInternal, constructorErr, w, nil)
		return
	}

	w.Write(responseBody)
}

func deleteFiles(projectName string) error {
	bucketName, bucketExists := os.LookupEnv("MIO_BUCKET")
	if !bucketExists {
		return fmt.Errorf("[DEPLOYMENT] bucket name was not set in env variable")
	}

	var objects []minio.ObjectInfo
	for object := range config.Minio.ListObjects(context.Background(), bucketName, minio.ListObjectsOptions{Prefix: projectName, Recursive: true}) {
		if object.Err != nil {
			return object.Err
		}
		objects = append(objects, object)
	}

	var err error
	for _, object := range objects {
		err = removeObject(object, bucketName)
		if err != nil {
			return err
		}
	}

	return nil
}

func removeObject(object minio.ObjectInfo, bucketName string) error {
	opts := minio.RemoveObjectOptions{
		GovernanceBypass: true,
	}

	err := config.Minio.RemoveObject(context.Background(), bucketName, object.Key, opts)
	if err != nil {
		return err
	}

	return nil
}
