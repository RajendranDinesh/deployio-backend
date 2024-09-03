package rabbit

import (
	"buildServer/build"
	"buildServer/config"
	"buildServer/upload"
	"buildServer/utils"
	"encoding/json"
	"log"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func ConsumeRabbitQueue(msgs <-chan amqp.Delivery) {
	for d := range msgs {
		var request struct {
			BuildId int
		}

		deconstructorErr := json.Unmarshal(d.Body, &request)
		if deconstructorErr != nil {
			utils.UpdateBuildLog(request.BuildId, deconstructorErr.Error())
			utils.SetBuildStatus(request.BuildId, "failure")
			log.Println("[JSON] erred while deconstructing request from client")
			continue
		}

		// set ack to true otherwise rabbitmq would redistribute the request to other workers, since the build would take some time
		d.Ack(true)

		log.Printf("[BUILD] Received job with build id %d", request.BuildId)
		userId, projectId, githubId, err := build.GetUserIdAndProjectId(request.BuildId)
		if err != nil {
			utils.UpdateBuildLog(request.BuildId, err.Error())
			utils.SetBuildStatus(request.BuildId, "failure")
			log.Println("[GETu&pId] " + err.Error())
			continue
		}

		query := `UPDATE "deploy-io".builds SET start_time = $1, status = 'running' WHERE id = $2`
		_, qErr := config.DataBase.Exec(query, time.Now(), request.BuildId)
		if qErr != nil {
			log.Fatalln("[DATABASE] " + qErr.Error())
		}

		archiveURL, archiveErr := build.GetArchiveURL(*githubId, *userId)
		if archiveErr != nil {
			utils.UpdateBuildLog(request.BuildId, archiveErr.Error())
			utils.SetBuildStatus(request.BuildId, "failure")
			log.Println("[GETarcURL] erred while getting archieve url " + archiveErr.Error())
			continue
		}

		// replace `{archive_format}` with `tarball`
		archiveURL = strings.ReplaceAll(archiveURL, "{archive_format}", "tarball")

		// remove "{/ref}"
		archiveURL = strings.ReplaceAll(archiveURL, "{/ref}", "")

		workingDir, cloneErr := build.CloneAndExtractRepository(archiveURL, *userId, request.BuildId)
		if cloneErr != nil {
			utils.UpdateBuildLog(request.BuildId, cloneErr.Error())
			utils.SetBuildStatus(request.BuildId, "failure")
			log.Println("[CLONE&EXT] failed to clone and extract repo " + cloneErr.Error())
			continue
		}

		projectDir := utils.GetCurDir() + "/tmp/" + workingDir

		directory, installCommand, buildCommand, outputFolder, nodeVersion, getInstallCmdErr := build.GetDefaults(request.BuildId)
		if getInstallCmdErr != nil {
			utils.UpdateBuildLog(request.BuildId, getInstallCmdErr.Error())
			utils.SetBuildStatus(request.BuildId, "failure")
			utils.DeleteDirectory(projectDir)
			log.Println("[GETi&bCMD] failed to get install or build command " + getInstallCmdErr.Error())
			continue
		}

		if outputFolder[0] != '/' {
			outputFolder = "/" + outputFolder
		}

		if directory != "./" {
			projectDir = projectDir + directory
		}

		installErr := build.InstallDependencies(request.BuildId, nodeVersion, installCommand, projectDir)
		if installErr != nil {
			utils.UpdateBuildLog(request.BuildId, installErr.Error())
			utils.SetBuildStatus(request.BuildId, "failure")
			utils.DeleteDirectory(projectDir)
			log.Println("[SERVER] failed to install dependencies " + installErr.Error())
			continue
		}

		builderr := build.BuildProject(*projectId, request.BuildId, nodeVersion, buildCommand, projectDir, outputFolder)
		if builderr != nil {
			utils.UpdateBuildLog(request.BuildId, builderr.Error())
			utils.SetBuildStatus(request.BuildId, "failure")
			utils.DeleteDirectory(projectDir)
			log.Println("[SERVER] failed to build project " + builderr.Error())
			continue
		}

		uploadErr := upload.UploadProjectFiles(request.BuildId, *userId, workingDir)
		if uploadErr != nil {
			utils.UpdateBuildLog(request.BuildId, uploadErr.Error())
			utils.SetBuildStatus(request.BuildId, "failure")
			utils.DeleteDirectory(projectDir)
			log.Println("[UPLOAD] failed to upload stuff " + uploadErr.Error())
			continue
		}

		setFalseQuery := `UPDATE "deploy-io".deployments SET status = false WHERE project_id = $1`
		_, setFalseErr := config.DataBase.Exec(setFalseQuery, projectId)
		if setFalseErr != nil {
			utils.UpdateBuildLog(request.BuildId, setFalseErr.Error())
			utils.SetBuildStatus(request.BuildId, "failure")
			utils.DeleteDirectory(projectDir)
			log.Println("[UPDATE] failed to update existing status to false " + setFalseErr.Error())
			continue
		}

		insQuery := `INSERT INTO "deploy-io".deployments (project_id, build_id) VALUES ($1, $2)`
		_, insErr := config.DataBase.Exec(insQuery, projectId, request.BuildId)
		if insErr != nil {
			utils.UpdateBuildLog(request.BuildId, insErr.Error())
			utils.SetBuildStatus(request.BuildId, "failure")
			utils.DeleteDirectory(projectDir)
			log.Println("[INSERT] failed to insert new build into deployments " + insErr.Error())
			continue
		}
	}
}
