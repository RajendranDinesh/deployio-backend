package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"httpServer/config"
	"httpServer/src"

	"github.com/joho/godotenv"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var IsOnProd bool

func init() {
	parseFlags()

	if !IsOnProd {
		initGoDotENV()
	}

	printTitle()
	config.InitDBConnection()
	config.InitRabbitConnection()
	config.InitMinioConnection()
}

func main() {

	ipAddress := getIPAddress()
	port := getPortNumber()

	log.Printf("[SERVER] Exposed On %[1]s:%[2]s\n", ipAddress, port)

	http2Server := &http2.Server{}

	server := &http.Server{Addr: ipAddress + ":" + port, Handler: h2c.NewHandler(src.Service(), http2Server)}

	// graceful shutdown implementation

	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)

	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sig

		shutdownCtx, shutdownStopCtx := context.WithTimeout(serverCtx, 10*time.Second)

		println("[SERVER] Shutting Down...")

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatalln("[SERVER] Shutdown Timed Out. Forcing Exit.")
			}
		}()

		err := server.Shutdown(shutdownCtx)
		if err != nil {
			log.Fatalln("[SERVER] ", err)
		}

		shutdownStopCtx()
		serverStopCtx()
	}()

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalln("[SERVER] ", err)
	}

	defer config.RabbitConnection.Close()
	defer config.RabbitChannel.Close()
	defer config.DataBase.Close()
	<-serverCtx.Done()
}

// loads secrets from .env
func initGoDotENV() {
	err := godotenv.Load()

	if err != nil {
		log.Fatalln("[SERVER] Error Loading .env file")
	}
}

func getPortNumber() string {
	port := os.Getenv("PORT")

	if len(strings.TrimSpace(port)) == 0 {
		log.Fatalln("[SERVER] PORT number was not provided")
	}

	return port
}

func getIPAddress() string {
	ipAddress := os.Getenv("IP")

	if len(strings.TrimSpace(ipAddress)) == 0 {
		log.Fatalln("[SERVER] IP Address was not provided")
	}

	return ipAddress
}

func printTitle() {
	println(`
░▒▓███████▓▒░░▒▓████████▓▒░▒▓███████▓▒░░▒▓█▓▒░      ░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░            ░▒▓█▓▒░░▒▓██████▓▒░       
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░     ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░            ░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░      
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░     ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░            ░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░      
░▒▓█▓▒░░▒▓█▓▒░▒▓██████▓▒░ ░▒▓███████▓▒░░▒▓█▓▒░     ░▒▓█▓▒░░▒▓█▓▒░░▒▓██████▓▒░             ░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░      
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░      ░▒▓█▓▒░     ░▒▓█▓▒░░▒▓█▓▒░  ░▒▓█▓▒░                ░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░      
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░      ░▒▓█▓▒░     ░▒▓█▓▒░░▒▓█▓▒░  ░▒▓█▓▒░                ░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░      
░▒▓███████▓▒░░▒▓████████▓▒░▒▓█▓▒░      ░▒▓████████▓▒░▒▓██████▓▒░   ░▒▓█▓▒░                ░▒▓█▓▒░░▒▓██████▓▒░       
`)
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
