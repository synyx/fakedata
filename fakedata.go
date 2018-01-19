package main

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"log"
	"strings"
)

type rabbitConf struct {
	hostname string
	port     int
	username string
	password string
}

type rabbitArtifacts struct {
	queriesExchangeName string
	queriesQueueName    string
}

type rabbitMqDestination struct {
	destination string
	routingKey string
}




func main() {
	rabbitConfig := readRabbitConf()
	conn := connectRabbit(rabbitConfig)

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")

	rabbitArtifacts := setupRabbitMqTopicsAndQueues(ch, "queries", "fakedata.queries")

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
			fmt.Println(string(msg.Body))
			rabbitMqDest, err := extractDestinationAndRoutingKeyFromReplyTo(msg.ReplyTo)
			logOnError(err, "failed to parse reply-to: %s")
			if err != nil {
				msg.Nack(false, false)
			} else {
				fmt.Println(fmt.Sprintf("received a query message and will send repsonse to %s", rabbitMqDest))
				answersToSend <- rabbitMqDest
				msg.Ack(false)
			}
		}
	}()

	go func(channel *amqp.Channel)  {

		rabbitDest := <-answersToSend
		fmt.Println(fmt.Sprintf("would send a msg to %s", rabbitDest))
	}(ch)

	<-forever
	defer ch.Close()
	defer conn.Close()
}

func extractDestinationAndRoutingKeyFromReplyTo(replyTo string) (rabbitMqDestination, error) {
	if len(replyTo) == 0 {
		return rabbitMqDestination{"", ""}, fmt.Errorf("cannot create destination and/or routing key from empty reply-to")
	}

	if strings.Contains(replyTo, "/") {
		destinationAndRoutingKey := strings.Split(replyTo, "/")
		if len(destinationAndRoutingKey) != 2 {
			return rabbitMqDestination{"", ""}, fmt.Errorf("cannot create destination and/or routing key from reply-to with more than two slashes (/)")
		}
		return rabbitMqDestination{destinationAndRoutingKey[0], destinationAndRoutingKey[1]}, nil
	} else {
		return rabbitMqDestination{replyTo, ""}, nil
	}
}

func readRabbitConf() rabbitConf {
	viper.SetConfigFile("config.properties")
	viper.SetConfigType("properties")

	//TODO: build default values localhost:5672 no credentials
	confErr := viper.ReadInConfig()
	logOnError(confErr, "No configuration file loaded - using defaults {}")
		fmt.Println()
	hostname := viper.GetString("rabbitmq.hostname")
	port := viper.GetInt("rabbitmq.port")
	username := viper.GetString("rabbitmq.username")
	password := viper.GetString("rabbitmq.password")

	return rabbitConf{hostname: hostname, port: port, username: username, password: password}
}

func connectRabbit(conf rabbitConf) *amqp.Connection {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/", conf.username, conf.password, conf.hostname, conf.port))
	failOnError(err, "failed to connect to rabbitmq")
	fmt.Println("connected to rabbitmq")
	return conn
}

func setupRabbitMqTopicsAndQueues(channel *amqp.Channel, queriesExchangeName string, queriesQueueName string) rabbitArtifacts {
	exchangeErr := channel.ExchangeDeclare(queriesExchangeName, "topic", true, false, false, false, nil)
	failOnError(exchangeErr, "failed to declare queries exchange")

	_, queriesErr := channel.QueueDeclare(
		queriesQueueName,
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(queriesErr, "Failed to declare queries queue")

	//TODO make configurable by users input data
	bindErr := channel.QueueBind(queriesQueueName, "profiles.all", queriesExchangeName, false, nil)
	failOnError(bindErr, "Failed to bind queries queue to topic exchange")

	fmt.Println(fmt.Sprintf("created topics and queues %s, %s", queriesQueueName, queriesExchangeName))

	return rabbitArtifacts{queriesExchangeName: queriesExchangeName, queriesQueueName: queriesQueueName}
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
