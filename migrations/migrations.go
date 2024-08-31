package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var IsOnProd bool

func init() {
	parseFlags()

	if !IsOnProd {
		initGoDotENV()
	}
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <create|migration-file-name> [<up|down>]\n", os.Args[0])
	}

	fileName := os.Args[1]

	if fileName == "create" {
		createMigration()
		return
	}

	direction := os.Args[2]

	if direction != "up" && direction != "down" {
		log.Fatalf("Invalid direction: %s. Use 'up' or 'down'.\n", direction)
	}

	// Construct the command
	cmd := exec.Command("migrate", "-path", "./schemas", "-database", GetDBConnString(), direction, "1")

	// Run the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to run migrate command: %v\nOutput: %s", err, output)
	}

	fmt.Printf("Migration output:\n%s", output)
}

func createMigration() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s create <migration-name>\n", os.Args[0])
	}

	migrationName := os.Args[2]

	// Construct the command to create a new migration
	cmd := exec.Command("migrate", "create", "-ext", "sql", "-dir", "./schemas", "-seq", migrationName)

	// Run the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to create migration: %v\nOutput: %s", err, output)
	}

	fmt.Printf("Migration created successfully:\n%s", output)
}

func GetDBConnString() string {
	var err error

	host, hostExists := os.LookupEnv("DB_HOST")
	port, portExists := os.LookupEnv("DB_PORT")
	username, dbUserExists := os.LookupEnv("DB_USER")
	password, dbPassExists := os.LookupEnv("DB_PASS")
	databaseName, dbNameExists := os.LookupEnv("DB_NAME")

	if !hostExists || !portExists || !dbUserExists || !dbPassExists || !dbNameExists {
		log.Fatalln("[DATABASE] Env probs..")
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&Timezone=Asia/Kolkata", username, password, host, port, databaseName)

	DataBase, err := sql.Open("postgres", connStr)

	if err != nil {
		log.Println(err)
		log.Fatalln("[DATABASE] Connection probs..")
	}

	err = DataBase.Ping()

	if err != nil {
		log.Println(err)
		log.Fatalln("[DATABASE] Could not ping the db.")
	}

	log.Printf("[DATABASE] Connected to %s\n", databaseName)

	return connStr
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

// loads secrets from .env
func initGoDotENV() {
	err := godotenv.Load()

	if err != nil {
		log.Fatalln("[SERVER] Error Loading .env file")
	}
}
