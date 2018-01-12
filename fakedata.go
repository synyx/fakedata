package main

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"log"
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

func main() {
	rabbitConfig := readRabbitConf()
	conn := connectRabbit(rabbitConfig)

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")

	setupRabbitMqTopicsAndQueues(ch, "queries", "fakedata.queries")

	defer ch.Close()
	defer conn.Close()
}

func readRabbitConf() rabbitConf {
	viper.SetConfigFile("config.properties")
	viper.SetConfigType("properties")

	//TODO: build default values localhost:5672 no credentials
	confErr := viper.ReadInConfig()
	if confErr != nil {
		fmt.Println("No configuration file loaded - using defaults")
	}
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
