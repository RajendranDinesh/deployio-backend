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

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", username, password, host, port, databaseName)

	DataBase, err = sql.Open("postgres", connStr)

	if err != nil {
		log.Println(err)
		log.Fatalln("[DATABASE] Connection probs..")
	}

	err = DataBase.Ping()

	if err != nil {
		log.Println(err)
		log.Fatalln("[DATABASE] Could not ping the db.")
	}

	fmt.Printf("[DATABASE] Connected to %s\n", databaseName)
}

func IsStringEmpty(str string) bool {
	return len(strings.TrimSpace(str)) == 0
}
