package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

func init() {
	initGoDotENV()
}

func main() {
	conn, err := amqp.Dial(getRabbitMQConnectionString())
	failOnError(err, "[SERVER] failed to connect rabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "[SERVER] failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"build_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnError(err, "[SERVER] failed to declare a queue")

	err = ch.Qos(
		1,
		0,
		false,
	)
	failOnError(err, "[SERVER] failed to set QOS")

	msgs, err := ch.Consume(
		q.Name,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	failOnError(err, "[SERVER] failed to register a worker")

	forever := make(chan int)

	go func() {
		for d := range msgs {
			log.Printf("Received a job")
			d.Ack(true)
		}
	}()

	log.Printf("[SERVER] waiting for build jobs..")
	<-forever

}

func CloneAndExtractRepository() {
	client := &http.Client{}

	req, err := http.NewRequest("GET", "https://api.github.com/repos/RajendranDinesh/aerys/tarball", nil)
	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Set("Authorization", "Bearer ghu_eCMhx5SJOfm9KLL2ErKSC7KxfVsIc72ktW9O")

	resp, err := client.Do(req)
	if err != nil {
		print("erred")
		log.Fatalln(err)
	}

	defer resp.Body.Close()

	file, err := os.Create("a.tar")
	if err != nil {
		print("erred")
		log.Fatalln(err)
	}

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		print("erred")
		log.Fatalln(err)
	}

	defer file.Close()

	cmd := exec.Command("tar", "-xvzf", "a.tar")

	_, err = cmd.Output()
	if err != nil {
		log.Fatalln(err)
	}

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
