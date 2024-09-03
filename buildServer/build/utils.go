package build

import (
	"buildServer/auth"
	"buildServer/config"
	"buildServer/utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

func InstallDependencies(buildId int, nodeVersion, installCommand, dir string) error {
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

func GetArchiveURL(githubId int, userId int) (string, error) {
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

	var repoResponse struct {
		ArchiveURL string
	}

	deconstructorErr := json.Unmarshal(repoBody, &repoResponse)
	if deconstructorErr != nil {
		return "", deconstructorErr
	}

	return repoResponse.ArchiveURL, nil
}

func GetDefaults(buildId int) (string, string, string, string, string, error) {
	var installCommand, buildCommand, outputFolder, directory, nodeVersion string

	retQuery := `SELECT p.directory, p.install_command, p.build_command, p.output_folder, p.node_version FROM "deploy-io".projects p JOIN "deploy-io".builds b ON p.id = b.project_id WHERE b.id = $1`
	queryErr := config.DataBase.QueryRow(retQuery, buildId).Scan(&directory, &installCommand, &buildCommand, &outputFolder, &nodeVersion)
	if queryErr != nil {
		return "", "", "", "", "", queryErr
	}

	updateErr := utils.UpdateBuildLog(buildId, "[CMD] got installation ("+installCommand+") and build ("+buildCommand+") commands")
	if updateErr != nil {
		return "", "", "", "", "", nil
	}

	return directory, installCommand, buildCommand, outputFolder, nodeVersion, nil
}

func GetUserIdAndProjectId(buildId int) (*int, *int, *int, error) {
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
