package main

import (
	"fmt"
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	rabbitConfig := readRabbitConf()
	conn := connectRabbit(rabbitConfig)
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	content, err := os.ReadFile(rabbitConfig.filename)
	failOnError(err, "failed to read data file")

	rabbitArtifacts := setupRabbitMqTopicsAndQueues(ch, rabbitConfig.queriesExchange, rabbitConfig.queriesQueue, rabbitConfig.queriesRoutingKey)

	msgs, consumeErr := ch.Consume(
		rabbitArtifacts.queriesQueueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(consumeErr, "failed to consume messages from queue")

	forever := make(chan bool)
	answersToSend := make(chan rabbitMqDestination)

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	go func() {
		for msg := range msgs {
			rabbitMqDest, err := extractDestinationAndRoutingKeyFromReplyTo(msg.ReplyTo)
			logOnError(err, "failed to parse reply-to: %s")
			if err != nil {
				err = msg.Nack(false, false)
				logOnError(err, fmt.Sprintf("failed to NACK message to %s", rabbitMqDest))
			} else {
				log.Println(fmt.Sprintf("received a query message and will send response to %s", rabbitMqDest))
				answersToSend <- rabbitMqDest
				err = msg.Ack(false)
				logOnError(err, fmt.Sprintf("failed to ACK message to to %s", rabbitMqDest))
			}
		}
	}()

	go func(channel *amqp.Channel, body []byte) {
		for {
			rabbitDest := <-answersToSend
			sendErr := channel.Publish(rabbitDest.destination, rabbitDest.routingKey, false, false,
				amqp.Publishing{
					ContentType: "text/plain",
					Body:        body,
				})
			logOnError(sendErr, "failed to send reply message:")
		}
	}(ch, content)

	<-forever
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}
func logOnError(err error, msg string) {
	if err != nil {
		log.Printf("%s: %s\n", msg, err)
	}
}
