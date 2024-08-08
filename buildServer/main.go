package main

import (
	auth "buildServer/auth"
	build "buildServer/build"
	"buildServer/config"
	"buildServer/upload"
	"buildServer/utils"

	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

func init() {
	initGoDotENV()
	utils.CreateTmpDir()
	config.InitDBConnection()
	config.InitMinioConnection()
}

func main() {
	conn, err := amqp.Dial(getRabbitMQConnectionString())
	failOnError(err, "[rabbitMQ] failed to connect")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "[rabbitMQ] failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"build_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnError(err, "[rabbitMQ] failed to declare a queue")

	err = ch.Qos(
		1,
		0,
		false,
	)
	failOnError(err, "[rabbitMQ] failed to set QOS")

	msgs, err := ch.Consume(
		q.Name,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	failOnError(err, "[rabbitMQ] failed to register a worker")

	forever := make(chan int)

	go func() {
		for d := range msgs {
			var request Request

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
			userId, projectId, githubId, err := getUserIdAndProjectId(request.BuildId)
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

			archiveURL, archiveErr := getArchiveURL(*githubId, *userId)
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

			workingDir, cloneErr := CloneAndExtractRepository(archiveURL, *userId, request.BuildId)
			if cloneErr != nil {
				utils.UpdateBuildLog(request.BuildId, cloneErr.Error())
				utils.SetBuildStatus(request.BuildId, "failure")
				log.Println("[CLONE&EXT] failed to clone and extract repo " + cloneErr.Error())
				continue
			}

			directory, installCommand, buildCommand, outputFolder, nodeVersion, getInstallCmdErr := getDefaults(request.BuildId)
			if getInstallCmdErr != nil {
				utils.UpdateBuildLog(request.BuildId, getInstallCmdErr.Error())
				utils.SetBuildStatus(request.BuildId, "failure")
				log.Println("[GETi&bCMD] failed to get install or build command " + getInstallCmdErr.Error())
				continue
			}

			if outputFolder[0] != '/' {
				outputFolder = "/" + outputFolder
			}

			projectDir := utils.GetCurDir() + "/tmp/" + workingDir

			if directory != "./" {
				projectDir = projectDir + directory
			}

			installErr := InstallDependencies(request.BuildId, nodeVersion, installCommand, projectDir)
			if installErr != nil {
				utils.UpdateBuildLog(request.BuildId, installErr.Error())
				utils.SetBuildStatus(request.BuildId, "failure")
				log.Println("[SERVER] failed to install dependencies " + installErr.Error())
				continue
			}

			builderr := build.BuildProject(*projectId, request.BuildId, nodeVersion, buildCommand, projectDir, outputFolder)
			if builderr != nil {
				utils.UpdateBuildLog(request.BuildId, builderr.Error())
				utils.SetBuildStatus(request.BuildId, "failure")
				log.Println("[SERVER] failed to build project " + builderr.Error())
				continue
			}

			uploadErr := upload.UploadProjectFiles(request.BuildId, *userId, workingDir)
			if uploadErr != nil {
				utils.UpdateBuildLog(request.BuildId, uploadErr.Error())
				utils.SetBuildStatus(request.BuildId, "failure")
				log.Println("[UPLOAD] failed to upload stuff " + uploadErr.Error())
				continue
			}

			setFalseQuery := `UPDATE "deploy-io".deployments SET status = false WHERE project_id = $1`
			_, setFalseErr := config.DataBase.Exec(setFalseQuery, projectId)
			if setFalseErr != nil {
				utils.UpdateBuildLog(request.BuildId, setFalseErr.Error())
				utils.SetBuildStatus(request.BuildId, "failure")
				log.Println("[UPDATE] failed to update existing status to false " + setFalseErr.Error())
				continue
			}

			insQuery := `INSERT INTO "deploy-io".deployments (project_id, build_id) VALUES ($1, $2)`
			_, insErr := config.DataBase.Exec(insQuery, projectId, request.BuildId)
			if insErr != nil {
				utils.UpdateBuildLog(request.BuildId, insErr.Error())
				utils.SetBuildStatus(request.BuildId, "failure")
				log.Println("[INSERT] failed to insert new build into deployments " + insErr.Error())
				continue
			}
		}
	}()

	log.Printf("[SERVER] waiting for build jobs..\n")
	<-forever
}

func InstallDependencies(buildId, nodeVersion int, installCommand, dir string) error {
	command := strings.Fields(installCommand)

	cmdName := command[0]
	cmdArgs := command[1:]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if cmdName != "npm" {
		return fmt.Errorf("[INSTALL] something other than npm was used")
	}

	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
	cmd.Dir = dir

	nvmEnv, err := utils.LoadNvmEnv(nodeVersion)
	if err != nil {
		return fmt.Errorf("error loading nvm environment: %v", err)
	}

	env := os.Environ()

	env = append(env, nvmEnv...)

	cmd.Env = env

	var updateErr error

	updateErr = utils.UpdateBuildLog(buildId, "[INSTALL] Installing dependencies\n")
	if updateErr != nil {
		return updateErr
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("[INSTALL] took so long")
		}
		return fmt.Errorf("[INSTALL | OP] " + string(output) + "[ERROR]" + err.Error())
	}

	updateErr = utils.UpdateBuildLog(buildId, string(output))
	if updateErr != nil {
		return updateErr
	}

	return nil
}

func getArchiveURL(githubId int, userId int) (string, error) {
	accessToken, accessTokenErr := auth.GetAccessToken(userId)
	if accessTokenErr != nil {
		return "", accessTokenErr
	}

	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
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

	var repoResponse RepoResponse

	deconstructorErr := json.Unmarshal(repoBody, &repoResponse)
	if deconstructorErr != nil {
		return "", deconstructorErr
	}

	return repoResponse.ArchiveURL, nil
}

func getDefaults(buildId int) (string, string, string, string, int, error) {
	var installCommand, buildCommand, outputFolder, directory string
	var nodeVersion int

	retQuery := `SELECT p.directory, p.install_command, p.build_command, p.output_folder, p.node_version FROM "deploy-io".projects p JOIN "deploy-io".builds b ON p.id = b.project_id WHERE b.id = $1`
	queryErr := config.DataBase.QueryRow(retQuery, buildId).Scan(&directory, &installCommand, &buildCommand, &outputFolder, &nodeVersion)
	if queryErr != nil {
		return "", "", "", "", 0, queryErr
	}

	updateErr := utils.UpdateBuildLog(buildId, "[CMD] got installation ("+installCommand+") and build ("+buildCommand+") commands")
	if updateErr != nil {
		return "", "", "", "", 0, nil
	}

	return directory, installCommand, buildCommand, outputFolder, nodeVersion, nil
}

func getUserIdAndProjectId(buildId int) (*int, *int, *int, error) {
	var userId, projectId, githubId int

	retQuery := `SELECT p.user_id, p.id, p.github_id FROM "deploy-io".projects p JOIN "deploy-io".builds b ON p.id = b.project_id WHERE b.id = $1`
	queryErr := config.DataBase.QueryRow(retQuery, buildId).Scan(&userId, &projectId, &githubId)
	if queryErr != nil {
		return nil, nil, nil, queryErr
	}

	return &userId, &projectId, &githubId, nil
}

func CloneAndExtractRepository(archiveURL string, userId int, buildId int) (string, error) {
	accessToken, accessTokenErr := auth.GetAccessToken(userId)
	if accessTokenErr != nil {
		return "", accessTokenErr
	}

	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}

	resp, err := utils.Request("GET", archiveURL, &headers, nil, nil)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	file, err := os.Create(fmt.Sprintf("%s%d.tar", utils.GetCurDir()+"/tmp/", buildId))
	if err != nil {
		return "", err
	}

	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("tar", "-xvzf", file.Name(), "-C", utils.GetCurDir()+"/tmp")

	extractionOutput, err := cmd.Output()
	if err != nil {
		return "", err
	}

	files := strings.Split(string(extractionOutput), "\n")

	directoryName := files[0]

	removeErr := os.Remove(file.Name())
	if removeErr != nil {
		return "", removeErr
	}

	updateErr := utils.UpdateBuildLog(buildId, "[CLONE] Extracted repository and placed at "+directoryName)
	if updateErr != nil {
		return "", updateErr
	}

	return directoryName, nil
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func getRabbitMQConnectionString() string {
	host, hostExists := os.LookupEnv("MQ_HOST")
	port, portExists := os.LookupEnv("MQ_PORT")
	user, userExists := os.LookupEnv("MQ_USER")
	pass, passExists := os.LookupEnv("MQ_PASS")

	if !hostExists || !portExists || !userExists || !passExists {
		log.Fatalln("[SERVER] check environment configuration")
	}

	return fmt.Sprintf("amqp://%s:%s@%s:%s", user, pass, host, port)
}

func initGoDotENV() {
	err := godotenv.Load()

	if err != nil {
		log.Fatalln("[SERVER] Error Loading .env file")
	}
}
