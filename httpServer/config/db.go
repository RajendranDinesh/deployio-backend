package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DataBase *sql.DB

func InitDBConnection() {
	var err error

	host, hostExists := os.LookupEnv("DB_HOST")
	port, portExists := os.LookupEnv("DB_PORT")
	username, dbUserExists := os.LookupEnv("DB_USER")
	password, dbPassExists := os.LookupEnv("DB_PASS")
	databaseName, dbNameExists := os.LookupEnv("DB_NAME")

	if !hostExists || !portExists || !dbUserExists || !dbPassExists || !dbNameExists {
		log.Fatalln("[DATABASE] Env probs..")
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
