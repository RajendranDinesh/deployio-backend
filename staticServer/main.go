package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"staticServer/config"
	prom "staticServer/prometheus"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var IsOnProd bool

func init() {
	parseFlags()

	if !IsOnProd {
		initGoDotENV()
	}

	config.InitMinioConnection()

	prometheus.MustRegister(prom.FileRequestCounter)
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

	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

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
			return c.Status(fiber.StatusNotFound).SendFile("./public/404.html")
		}

		prom.FileRequestCounter.With(prometheus.Labels{"site": projectName, "file": fileName}).Inc()

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

func parseFlags() {
	env := flag.String("env", "dev", "The Environment in which the program is running, possible values are\n1. prod \n2. dev")

	flag.Parse()

	err := isOnProd(*env)

	if err == flag.ErrHelp {
		flag.PrintDefaults()
		os.Exit(2)
	}
}

// Changes the global variable IsOnProd
func isOnProd(env string) error {
	env = strings.TrimSpace(env)

	if env == "dev" || len(env) == 0 {
		IsOnProd = false
		return nil
	}

	if env != "prod" {
		return flag.ErrHelp
	}

	IsOnProd = true

	return nil
}
