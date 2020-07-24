package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/streadway/amqp"
)

type rabbitConf struct {
	hostname          string
	port              int
	username          string
	password          string
	filename          string
	queriesExchange   string
	queriesQueue      string
	queriesRoutingKey string
	timeout           time.Duration
}

type rabbitArtifacts struct {
	queriesExchangeName string
	queriesQueueName    string
}

type rabbitMqDestination struct {
	destination string
	routingKey  string
}

func readRabbitConf() rabbitConf {
	viper.SetConfigFile("config.properties")
	viper.SetConfigType("properties")

	viper.BindEnv("rabbitmq.hostname", "FAKEDATA_RABBITMQ_HOSTNAME")
	viper.BindEnv("rabbitmq.username", "FAKEDATA_RABBITMQ_USERNAME")
	viper.BindEnv("rabbitmq.password", "FAKEDATA_RABBITMQ_PASSWORD")
	viper.BindEnv("rabbitmq.port", "FAKEDATA_RABBITMQ_PORT")
	viper.BindEnv("rabbitmq.queries.exchange", "FAKEDATA_RABBITMQ_QUERIES_EXCHANGE")
	viper.BindEnv("rabbitmq.queries.queue", "FAKEDATA_RABBITMQ_QUERIES_QUEUE")
	viper.BindEnv("rabbitmq.queries.routingkey", "FAKEDATA_RABBITMQ_QUERIES_ROUTINGKEY")
	viper.BindEnv("rabbitmq.timeout", "FAKEDATA_RABBITMQ_TIMEOUT")
	viper.BindEnv("filename", "FAKEDATA_FILENAME")

	//default values suitable for vanilla rabbitmq docker container
	viper.SetDefault("rabbitmq.hostname", "localhost")
	viper.SetDefault("rabbitmq.port", "5672")
	viper.SetDefault("rabbitmq.username", "guest")
	viper.SetDefault("rabbitmq.password", "guest")
	viper.SetDefault("rabbitmq.timeout", "5s")
	viper.SetDefault("filename", "data.json")
	viper.SetDefault("rabbitmq.queries.exchange", "queries")

	//load config
	confErr := viper.ReadInConfig()
	logOnError(confErr, "No configuration file loaded - using defaults {}")

	return rabbitConf{
		hostname:          viper.GetString("rabbitmq.hostname"),
		port:              viper.GetInt("rabbitmq.port"),
		username:          viper.GetString("rabbitmq.username"),
		password:          viper.GetString("rabbitmq.password"),
		timeout:           viper.GetDuration("rabbitmq.timeout"),
		filename:          viper.GetString("filename"),
		queriesExchange:   viper.GetString("rabbitmq.queries.exchange"),
		queriesQueue:      viper.GetString("rabbitmq.queries.queue"),
		queriesRoutingKey: viper.GetString("rabbitmq.queries.routingkey"),
	}
}

func connectRabbit(conf rabbitConf) *amqp.Connection {
	for {
		conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/", conf.username, conf.password, conf.hostname, conf.port))
		if err == nil && conn != nil {
			log.Println("connected to rabbitmq")
			return conn
		}

		log.Println(fmt.Sprintf("failed to connect to rabbitmq will retry in %d. current cause: %s", conf.timeout, err))
		time.Sleep(conf.timeout)
	}
}

func extractDestinationAndRoutingKeyFromReplyTo(replyTo string) (rabbitMqDestination, error) {
	//empty
	if len(replyTo) == 0 {
		return rabbitMqDestination{"", ""}, fmt.Errorf("cannot create destination and/or routing key from empty reply-to")
	}

	//reply_to containing '/': first part is taken as exchange/destination, second part is taken as the routing key
	if strings.Contains(replyTo, "/") {
		destinationAndRoutingKey := strings.Split(replyTo, "/")
		if len(destinationAndRoutingKey) != 2 {
			return rabbitMqDestination{"", ""}, fmt.Errorf("cannot create destination and/or routing key from reply-to with more than two slashes (/)")
		}
		return rabbitMqDestination{destinationAndRoutingKey[0], destinationAndRoutingKey[1]}, nil
	}

	//plain reply_to (which is sent to default exchange with reply_to as routingkey)
	return rabbitMqDestination{"", replyTo}, nil
}

func setupRabbitMqTopicsAndQueues(channel *amqp.Channel, queriesExchangeName string, queriesQueueName string, queriesRoutingKey string) rabbitArtifacts {
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

	bindErr := channel.QueueBind(queriesQueueName, queriesRoutingKey, queriesExchangeName, false, nil)
	failOnError(bindErr, "Failed to bind queries queue to topic exchange")

	log.Println(fmt.Sprintf("created topics and queues %s, %s", queriesQueueName, queriesExchangeName))

	return rabbitArtifacts{queriesExchangeName: queriesExchangeName, queriesQueueName: queriesQueueName}
}
