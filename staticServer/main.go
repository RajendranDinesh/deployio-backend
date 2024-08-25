package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
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

	app.Use(cache.New(cache.Config{
		Expiration:   1 * time.Minute,
		CacheControl: true,
	}))

	app.Get("*", func(c fiber.Ctx) error {
		projectName := strings.Split(c.Hostname(), ".")[0]
		fileName := projectName + c.Path()
		if c.Path() == "/" || !strings.Contains(c.Path(), ".") {
			fileName = projectName + "/index.html"
		}

		file, err := getFile(fileName)
		if err != nil {
			return c.Status(fiber.StatusNotFound).SendString("File not found")
		}

		// Set the appropriate content type based on the file extension
		switch {
		case strings.HasSuffix(fileName, ".html"):
			c.Type("html")
		case strings.HasSuffix(fileName, ".js"):
			c.Set("Content-Type", "application/javascript")
		case strings.HasSuffix(fileName, ".css"):
			c.Type("css")
		case strings.HasSuffix(fileName, ".json"):
			c.Type("json")
		case strings.HasSuffix(fileName, ".png"):
			c.Type("png")
		case strings.HasSuffix(fileName, ".jpg"), strings.HasSuffix(fileName, ".jpeg"):
			c.Type("jpeg")
		case strings.HasSuffix(fileName, ".gif"):
			c.Type("gif")
		case strings.HasSuffix(fileName, ".svg"):
			c.Set("Content-Type", "image/svg+xml")
		default:
			c.Type("text")
		}

		return c.Send(file)
	})

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
