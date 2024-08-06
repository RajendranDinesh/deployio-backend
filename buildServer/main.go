package main

import (
	auth "buildServer/auth"
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
	failOnError(err, "[SERVER] failed to connect rabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "[SERVER] failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"build_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnError(err, "[SERVER] failed to declare a queue")

	err = ch.Qos(
		1,
		0,
		false,
	)
	failOnError(err, "[SERVER] failed to set QOS")

	msgs, err := ch.Consume(
		q.Name,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	failOnError(err, "[SERVER] failed to register a worker")

	forever := make(chan int)

	go func() {
		for d := range msgs {
			var request Request

			deconstructorErr := json.Unmarshal(d.Body, &request)
			if deconstructorErr != nil {
				failOnError(deconstructorErr, "[SERVER] erred while deconstructing request from client")
			}
			d.Ack(true)

			log.Printf("Received a job %d", request.BuildId)
			userId, projectId, githubId, err := getUserIdAndProjectId(request.BuildId)
			if err != nil {
				log.Print(err.Error())
			}

			archiveURL, archiveErr := getArchiveURL(*githubId, *userId)
			if archiveErr != nil {
				failOnError(archiveErr, "[SERVER] erred while getting archieve url")
			}

			// replace `{archive_format}` with `tarball`
			archiveURL = strings.ReplaceAll(archiveURL, "{archive_format}", "tarball")

			// remove "{/ref}"
			archiveURL = strings.ReplaceAll(archiveURL, "{/ref}", "")

			workingDir, cloneErr := CloneAndExtractRepository(archiveURL, *userId, request.BuildId)
			if cloneErr != nil {
				failOnError(cloneErr, "[SERVER] failed to clone and extract repo")
			}

			installCommand, buildCommand, outputFolder, getInstallCmdErr := getInstallAndBuildCommand(request.BuildId)
			if getInstallCmdErr != nil {
				failOnError(getInstallCmdErr, "[SERVER] failed to get installation or build command")
			}

			installErr := InstallDependencies(installCommand, utils.GetCurDir()+"/tmp/"+workingDir)
			if installErr != nil {
				failOnError(installErr, "[SERVER] failed to install dependencies")
			}

			builderr := BuildProject(*projectId, buildCommand, utils.GetCurDir()+"/tmp/"+workingDir, outputFolder)
			if builderr != nil {
				if strings.Contains(builderr.Error(), "Specified output folder") {
					return
				} else {
					failOnError(builderr, "[SERVER] failed to build project")
				}
			}

			uploadErr := upload.UploadProjectFiles(request.BuildId, *userId, workingDir)
			if uploadErr != nil {
				failOnError(uploadErr, "[UPLOAD] failed to upload stuff")
			}
		}
	}()

	log.Printf("[SERVER] waiting for build jobs..")
	<-forever

}

func BuildProject(projectId int, buildCommand, dir, outputFolder string) error {
	command := strings.Fields(buildCommand)

	cmdName := command[0]
	cmdArgs := command[1:]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
	cmd.Dir = dir

	environments, envErr := getEnvironmentVariables(projectId)
	if envErr != nil {
		return envErr
	}

	nvmEnv, err := loadNvmEnv()
	if err != nil {
		return fmt.Errorf("error loading nvm environment: %v", err)
	}

	env := os.Environ()

	env = append(env, nvmEnv...)
	env = append(env, environments...)

	cmd.Env = env

	op, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("[INSTALL] took so long")
		}
		println("[BUILD] " + string(op))
		return err
	}

	// check if output folder exists else raise error
	if !utils.FolderExists(dir + outputFolder) {
		return fmt.Errorf("[BUILD] Specified output folder %s was not found after build", outputFolder)
	}

	return nil
}

func getEnvironmentVariables(projectId int) ([]string, error) {
	query := `SELECT key, value FROM "deploy-io".environments WHERE project_id = $1`

	envs, dbErr := config.DataBase.Query(query, projectId)
	if dbErr != nil {
		return nil, dbErr
	}

	var environments []string

	for envs.Next() {
		var key, encValue string

		envs.Scan(&key, &encValue)

		value, decError := auth.Decrypt(encValue)
		if decError != nil {
			return nil, decError
		}

		environments = append(environments, fmt.Sprintf("%s=%s", key, value))
	}

	return environments, nil
}

func InstallDependencies(installCommand string, dir string) error {
	command := strings.Fields(installCommand)

	cmdName := command[0]
	cmdArgs := command[1:]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
	cmd.Dir = dir

	_, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("[INSTALL] took so long")
		}
		return err
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

func getInstallAndBuildCommand(buildId int) (string, string, string, error) {
	var installCommand, buildCommand, outputFolder string

	retQuery := `SELECT p.install_command, p.build_command, p.output_folder FROM "deploy-io".projects p JOIN "deploy-io".builds b ON p.id = b.project_id WHERE b.id = $1`
	queryErr := config.DataBase.QueryRow(retQuery, buildId).Scan(&installCommand, &buildCommand, &outputFolder)
	if queryErr != nil {
		return "", "", "", queryErr
	}

	return installCommand, buildCommand, outputFolder, nil
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

func loadNvmEnv() ([]string, error) {
	cmd := exec.Command("bash", "-c", "source ~/.nvm/nvm.sh && env")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error loading nvm environment: %v", err)
	}
	env := strings.Split(string(output), "\n")
	return env, nil
}
