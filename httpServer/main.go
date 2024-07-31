package main

import (
	"context"
	"fmt"
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
)

func main() {
	initGoDotENV()

	printTitle()

	ipAddress := getIPAddress()
	port := getPortNumber()

	config.InitDBConnection()

	fmt.Printf("[SERVER] Exposed On %[1]s:%[2]s\n", ipAddress, port)

	// graceful shutdown implementation

	server := &http.Server{Addr: ipAddress + ":" + port, Handler: src.Service()}

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
