package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

var DataBase *sql.DB

func InitDBConnection() {
	var err error

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	username := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASS")
	databaseName := os.Getenv("DB_NAME")

	if IsStringEmpty(host) || IsStringEmpty(port) || IsStringEmpty(username) || IsStringEmpty(password) || IsStringEmpty(databaseName) {
		log.Fatalln("[DATABASE] Env probs..")
		os.Exit(1)
	}

	DataBase, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", username, password, host, port, databaseName))

	if err != nil || DataBase == nil {
		log.Fatalln(err)
		log.Fatalln("[DATABASE] Connection probs..")
		os.Exit(1)
	}

	fmt.Printf("[DATABASE] Connected to %s\n", databaseName)
}

func IsStringEmpty(str string) bool {
	return len(strings.TrimSpace(str)) == 0
}
