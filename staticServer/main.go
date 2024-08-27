package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"staticServer/config"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/cache"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func init() {
	initGoDotENV()
	config.InitMinioConnection()
}

func getFile(fileName string) ([]byte, error) {
	bucket, bucketExists := os.LookupEnv("MIO_BUCKET")
	if !bucketExists {
		return []byte(""), fmt.Errorf("[BUCKET] bucket name was not found in env")
	}

	object, getErr := config.Minio.GetObject(context.Background(), bucket, fileName, minio.GetObjectOptions{})
	if getErr != nil {
		fmt.Println(getErr)
		return []byte(""), getErr
	}

	defer object.Close()

	content, readErr := io.ReadAll(object)
	if readErr != nil {
		return []byte(""), readErr
	}

	return content, nil
}

func main() {
	// Initialize a new Fiber app
	app := fiber.New()

	// Cache configuration
	app.Use(cache.New(cache.Config{
		Expiration:   1 * time.Minute,
		CacheControl: true,
	}))

	// MIME type map
	mimeTypes := map[string]string{
		".html": "text/html",
		".js":   "application/javascript",
		".css":  "text/css",
		".json": "application/json",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
	}

	// Route to handle all GET requests
	app.Get("*", func(c fiber.Ctx) error {
		projectName := strings.Split(c.Hostname(), ".")[0]
		path := c.Path()

		// Determine the file name
		fileName := projectName + path
		if path == "/" || !strings.Contains(path, ".") {
			fileName = projectName + "/index.html"
		}

		// Get the file content
		file, err := getFile(fileName)
		if err != nil {
			log.Printf("File not found: %s", fileName)
			return c.Status(fiber.StatusNotFound).SendString("File not found")
		}

		// Set the content type based on the file extension
		ext := filepath.Ext(fileName)
		if mimeType, found := mimeTypes[ext]; found {
			c.Set("Content-Type", mimeType)
		} else {
			c.Type("text")
		}

		return c.Send(file)
	})

	// HTTP/2 server setup
	http2Server := &http2.Server{}
	app.Use(adaptor.HTTPHandler(h2c.NewHandler(adaptor.FiberApp(app), http2Server)))

	// Start the server on port 3000
	log.Fatal(app.Listen(":3000"))
}

func initGoDotENV() {
	err := godotenv.Load()

	if err != nil {
		log.Fatalln("[SERVER] Error Loading .env file")
	}
}
