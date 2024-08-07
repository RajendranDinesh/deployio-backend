package build

import (
	"buildServer/auth"
	"buildServer/config"
	"buildServer/utils"
	"log"

	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func BuildProject(projectId, buildId int, buildCommand, dir, outputFolder string) error {
	command := strings.Fields(buildCommand)

	cmdName := command[0]
	cmdArgs := command[1:]

	if cmdName != "npm" {
		return fmt.Errorf("[BUILD] something other than npm was used")
	}

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

	var updateErr error

	updateErr = utils.UpdateBuildLog(buildId, "[BUILD] Running build command")
	if updateErr != nil {
		return updateErr
	}

	op, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("[INSTALL] took so long")
		}
		return fmt.Errorf("[BUILD]\n" + string(op))
	}

	// check if output folder exists else raise error
	if !utils.FolderExists(dir + outputFolder) {
		return fmt.Errorf("[BUILD] Specified output folder %s was not found after build", outputFolder)
	}

	updateErr = utils.UpdateBuildLog(buildId, string(op))
	if updateErr != nil {
		return updateErr
	}

	log.Printf("[BUILD] Completed build of %d\n", buildId)

	return nil
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
