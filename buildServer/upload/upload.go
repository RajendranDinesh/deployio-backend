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
	var outputFolder, projectName, directory string
	query := `SELECT p.name, p.output_folder, p.directory FROM "deploy-io".projects p JOIN "deploy-io".builds b ON p.id = b.project_id WHERE p.user_id = $1 AND b.id = $2`
	queryErr := config.DataBase.QueryRow(query, userId, buildId).Scan(&projectName, &outputFolder, &directory)
	if queryErr != nil {
		log.Fatalln("[UPLOAD] " + queryErr.Error())
	}

	proDelErr := deleteExistingFiles(projectName)
	if proDelErr != nil {
		return proDelErr
	}

	var srcFolder string
	if directory != "./" {
		srcFolder = getCurDir() + "/tmp/" + workingDir + directory + outputFolder
	} else {
		srcFolder = getCurDir() + "/tmp/" + workingDir + outputFolder
	}

	files, getFileErr := getFilePaths(srcFolder)
	if getFileErr != nil {
		return getFileErr
	}

	for _, file := range files {
		// Remove everything until dist/
		index := strings.Index(file, outputFolder)
		if index == -1 {
			return fmt.Errorf("%s not found in path: %s", outputFolder, file)
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

	statErr := utils.SetBuildStatus(buildId, "success")
	if statErr != nil {
		return statErr
	}

	buildErr := utils.UpdateBuildLog(buildId, "Completed")
	if buildErr != nil {
		return buildErr
	}

	log.Printf("[UPLOAD] Completed uploading assets of build with id %d\n", buildId)

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

func deleteExistingFiles(projectName string) error {
	bucketName, bucketExists := os.LookupEnv("MIO_BUCKET")
	if !bucketExists {
		return fmt.Errorf("[UPLOAD] bucket name was not set in env variable")
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
