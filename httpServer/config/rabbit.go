package config

import (
	"fmt"
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

var RabbitConnection *amqp.Connection
var RabbitChannel *amqp.Channel
var RabbitQueue amqp.Queue

func InitRabbitConnection() {
	var err error
	RabbitConnection, err = amqp.Dial(getRabbitMQConnectionString())
	if err != nil {
		log.Fatalln("[rabbitMQ] failed to connect " + err.Error())
	}

	RabbitChannel, err = RabbitConnection.Channel()
	if err != nil {
		log.Fatalln("[rabbitMQ] failed to open a channel " + err.Error())
	}

	RabbitQueue, err = RabbitChannel.QueueDeclare(
		"build_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalln("[rabbitMQ] failed to declare a queue " + err.Error())
	}

	log.Println("[rabbitMQ] Connection Established")
}

func getRabbitMQConnectionString() string {
	host, hostExists := os.LookupEnv("MQ_HOST")
	port, portExists := os.LookupEnv("MQ_PORT")
	user, userExists := os.LookupEnv("MQ_USER")
	pass, passExists := os.LookupEnv("MQ_PASS")

	if !hostExists || !portExists || !userExists || !passExists {
		log.Fatalln("[RABBIT] check environment configuration")
	}

	return fmt.Sprintf("amqp://%s:%s@%s:%s", user, pass, host, port)
}
