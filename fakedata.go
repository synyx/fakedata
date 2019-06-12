package main

import (
	"fmt"
	"github.com/streadway/amqp"
	"io/ioutil"
	"log"
)

func main() {
	rabbitConfig := readRabbitConf()
	conn := connectRabbit(rabbitConfig)
	defer conn.Close()

	ch, err := conn.Channel()
	defer ch.Close()

	failOnError(err, "Failed to open a channel")

	content, err := ioutil.ReadFile(rabbitConfig.filename)
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
				msg.Nack(false, false)
			} else {
				log.Println(fmt.Sprintf("received a query message and will send repsonse to %s", rabbitMqDest))
				answersToSend <- rabbitMqDest
				msg.Ack(false)
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

