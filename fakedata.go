package main

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"io/ioutil"
	"log"
	"strings"
	"time"
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

type rabbitMqResponse struct {
	destination string        //the destination exchange
	routingKey  string        //the routing key to be used
	channel     *amqp.Channel //the channel that is used for sending
}

type rabbitMqDeliveryWithChannel struct {
	delivery amqp.Delivery
	channel  *amqp.Channel
}

func main() {
	rabbitConfig := readRabbitConf()

	//read content
	content, err := ioutil.ReadFile(rabbitConfig.filename)
	failOnError(err, "failed to read data file")

	forever := make(chan bool)
	answersToSend := make(chan rabbitMqResponse)
	requestForReconnect := make(chan bool)
	rabbitMqReconnect := make(chan *amqp.Channel, 5)

	rabbitmqDeliveryChannel := make(chan rabbitMqDeliveryWithChannel)

	go reconnectRabbit(rabbitConfig, rabbitMqReconnect, requestForReconnect)
	requestForReconnect <- true //initial connect

	//setup rabbitmq artifacts, only to be done once
	rabbitArtifacts := setupRabbitMqTopicsAndQueues(rabbitMqReconnect, rabbitConfig.queriesExchange, rabbitConfig.queriesQueue, rabbitConfig.queriesRoutingKey)

	go consumeFromChannel(rabbitMqReconnect, rabbitArtifacts, rabbitmqDeliveryChannel)

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	go func() {
		for mqDeliveryWithChannel := range rabbitmqDeliveryChannel {
			rabbitMqResponse, err := extractDestinationAndRoutingKeyFromReplyTo(mqDeliveryWithChannel.delivery.ReplyTo)
			rabbitMqResponse.channel = mqDeliveryWithChannel.channel
			rabbitMqDelivery := mqDeliveryWithChannel.delivery
			logOnError(err, "failed to parse reply-to: %s")
			if err != nil {
				rabbitMqDelivery.Nack(false, false)
			} else {
				log.Println(fmt.Sprintf("received a query message and will send repsonse to %s", rabbitMqResponse))
				answersToSend <- rabbitMqResponse
				rabbitMqDelivery.Ack(false)
			}
		}
	}()

	go func(body []byte) {

		for {
			rabbitMqResponse := <-answersToSend
			sendErr := rabbitMqResponse.channel.Publish(rabbitMqResponse.destination, rabbitMqResponse.routingKey, false, false,
				amqp.Publishing{
					ContentType: "text/plain",
					Body:        body,
				})
			if err != nil {
				requestForReconnect <- true       //request a reconnect
				answersToSend <- rabbitMqResponse //retry until this succeeds
			}
			logOnError(sendErr, "failed to send response")
		}

	}(content)
	<-forever
}

func consumeFromChannel(rabbitReconnectChannel chan *amqp.Channel, rabbitArtifacts rabbitArtifacts, rabbitmqDeliveryChannel chan rabbitMqDeliveryWithChannel) {
	for {
		rabbitChannel := <-rabbitReconnectChannel
		msgs, consumeErr := rabbitChannel.Consume(
			rabbitArtifacts.queriesQueueName,
			"",
			false,
			false,
			false,
			false,
			nil,
		)

		failOnError(consumeErr, "failed to consume messages from queue")
		for rabbitmqDelivery := range msgs {
			rabbitmqDeliveryChannel <- rabbitMqDeliveryWithChannel{rabbitmqDelivery, rabbitChannel}
		}
	}
}

func reconnectRabbit(conf rabbitConf, rabbitReconnectChannel chan *amqp.Channel, requestsForReconnect chan bool) {
	for {
		<-requestsForReconnect

		rabbitMqErrorListener := make(chan *amqp.Error)

		go func() {
			amqpError := <-rabbitMqErrorListener
			log.Println(fmt.Sprintf("amqp error: %s", amqpError))
			log.Println("will issue request for reconnect")
			requestsForReconnect <- true
		}()

		conn := connectRabbit(conf)
		conn.NotifyClose(rabbitMqErrorListener) // connection erros will go here
		channel, err := conn.Channel()
		failOnError(err, "Failed to open a channel")
		rabbitReconnectChannel <- channel // send pointer to rabbit channel to interested parties
	}
}

func connectRabbit(conf rabbitConf) *amqp.Connection {
	for {
		conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/", conf.username, conf.password, conf.hostname, conf.port))
		if err == nil && conn != nil {
			log.Println("connected to rabbitmq")
			return conn
		} else {
			log.Println(fmt.Sprintf("failed to connect to rabbitmq will retry in %d. current cause: %s", conf.timeout, err))
			time.Sleep(conf.timeout)
		}
	}
}

func extractDestinationAndRoutingKeyFromReplyTo(replyTo string) (rabbitMqResponse, error) {
	if len(replyTo) == 0 {
		return rabbitMqResponse{"", "", nil}, fmt.Errorf("cannot create destination and/or routing key from empty reply-to")
	}

	if strings.Contains(replyTo, "/") {
		destinationAndRoutingKey := strings.Split(replyTo, "/")
		if len(destinationAndRoutingKey) != 2 {
			return rabbitMqResponse{"", "", nil}, fmt.Errorf("cannot create destination and/or routing key from reply-to with more than two slashes (/)")
		}
		return rabbitMqResponse{destinationAndRoutingKey[0], destinationAndRoutingKey[1], nil}, nil
	} else {
		return rabbitMqResponse{replyTo, "", nil}, nil
	}
}

func readRabbitConf() rabbitConf {
	viper.SetConfigFile("config.properties")
	viper.SetConfigType("properties")

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

func setupRabbitMqTopicsAndQueues(rabbitReconnect chan *amqp.Channel, queriesExchangeName string, queriesQueueName string, queriesRoutingKey string) rabbitArtifacts {
	channel := <-rabbitReconnect
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

	rabbitReconnect <- channel
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
