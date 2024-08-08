package utils

import (
	"buildServer/config"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func FolderExists(folderPath string) bool {
	info, err := os.Stat(folderPath)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func DeleteDirectory(folderPath string) error {
	err := os.RemoveAll(folderPath)
	if err != nil {
		return err
	}
	return nil
}

func GetCurDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return "./"
	}

	return cwd
}

func CreateTmpDir() error {
	cwd := GetCurDir()
	tmpDir := cwd + "/tmp"

	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		if err := os.Mkdir(tmpDir, 0755); err != nil {
			return fmt.Errorf("[SERVER] error creating directory: %v", err)
		}
		log.Println("[SERVER] Directory 'tmp' created.")
	} else {
		log.Println("[SERVER] Directory 'tmp' already exists.")
	}

	return nil
}

func UpdateBuildLog(buildId int, log string) error {
	query := `UPDATE "deploy-io".builds SET logs = COALESCE(logs || E'\n', '') || $1, end_time = $2 WHERE id = $3`
	_, queErr := config.DataBase.Exec(query, log, time.Now(), buildId)
	if queErr != nil {
		return queErr
	}

	return nil
}

func SetBuildStatus(buildId int, status string) error {
	query := `UPDATE "deploy-io".builds SET status = $1::"deploy-io".build_status WHERE id = $2`
	_, queErr := config.DataBase.Exec(query, status, buildId)
	if queErr != nil {
		return queErr
	}

	return nil
}

func LoadNvmEnv(nodeVersion int) ([]string, error) {
	cmd := exec.Command("bash", "-c", "source ~/.nvm/nvm.sh && nvm install "+fmt.Sprintf("%v", nodeVersion)+" && nvm use "+fmt.Sprintf("%v", nodeVersion)+" && env")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error loading nvm environment: %v", err)
	}
	env := strings.Split(string(output), "\n")
	return env, nil
}
