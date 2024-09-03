package main

import (
	"buildServer/config"
	"buildServer/rabbit"
	"buildServer/utils"
	"flag"

	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

var IsOnProd bool

func init() {
	parseFlags()

	if !IsOnProd {
		initGoDotENV()
	}

	config.InitDBConnection()
	config.InitMinioConnection()
	utils.CreateTmpDir()
}

func main() {
	conn, err := amqp.Dial(getRabbitMQConnectionString())
	failOnError(err, "[rabbitMQ] failed to connect")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "[rabbitMQ] failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"build_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnError(err, "[rabbitMQ] failed to declare a queue")

	err = ch.Qos(
		1,
		0,
		false,
	)
	failOnError(err, "[rabbitMQ] failed to set QOS")

	msgs, err := ch.Consume(
		q.Name,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	failOnError(err, "[rabbitMQ] failed to register a worker")

	forever := make(chan int)

	go rabbit.ConsumeRabbitQueue(msgs)

	log.Printf("[SERVER] waiting for build jobs..\n")
	<-forever
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func getRabbitMQConnectionString() string {
	host, hostExists := os.LookupEnv("MQ_HOST")
	port, portExists := os.LookupEnv("MQ_PORT")
	user, userExists := os.LookupEnv("MQ_USER")
	pass, passExists := os.LookupEnv("MQ_PASS")

	if !hostExists || !portExists || !userExists || !passExists {
		log.Fatalln("[SERVER] check environment configuration")
	}

	return fmt.Sprintf("amqp://%s:%s@%s:%s", user, pass, host, port)
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
