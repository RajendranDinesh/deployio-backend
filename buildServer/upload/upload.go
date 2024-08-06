package upload

import (
	"buildServer/config"
	"buildServer/utils"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
)

func UploadProjectFiles(buildId int, userId int, workingDir string) error {
	var outputFolder string
	var projectName string
	query := `SELECT p.name, p.output_folder FROM "deploy-io".projects p JOIN "deploy-io".builds b ON p.id = b.project_id WHERE p.user_id = $1 AND b.id = $2`
	queryErr := config.DataBase.QueryRow(query, userId, buildId).Scan(&projectName, &outputFolder)
	if queryErr != nil {
		log.Fatalln("[UPLOAD] " + queryErr.Error())
	}

	srcFolder := getCurDir() + "/tmp/" + workingDir + outputFolder

	files, getFileErr := getFilePaths(srcFolder)
	if getFileErr != nil {
		return getFileErr
	}

	for _, file := range files {
		// Remove everything until dist/
		index := strings.Index(file, outputFolder)
		if index == -1 {
			return fmt.Errorf("dist/ not found in path: %s", file)
		}

		// Remove "dist/" and everything before it
		relPath := file[index+len(outputFolder):]

		destPath := filepath.Join(projectName, relPath)
		err := uploadFile(destPath, file)
		if err != nil {
			return err
		}
	}

	delErr := utils.DeleteDirectory(getCurDir() + "/tmp/" + workingDir)
	if delErr != nil {
		return delErr
	}

	return nil
}

func uploadFile(objectName string, filePath string) error {
	bucketName, bucketExists := os.LookupEnv("MIO_BUCKET")

	if !bucketExists {
		return fmt.Errorf("[UPLOAD] bucket name was not set in env variable")
	}

	ctx := context.Background()

	// Upload the test file
	// Change the value of filePath if the file is in another location
	contentType := "application/octet-stream"

	// Upload the test file with FPutObject
	_, err := config.Minio.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return err
	}

	return nil
}

func getFilePaths(folder string) ([]string, error) {
	var filePaths []string

	err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// If it's a file, add it to the list
		if !info.IsDir() {
			filePaths = append(filePaths, path)
		}
		return nil
	})

	return filePaths, err
}

func getCurDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return "./"
	}

	return cwd
}
