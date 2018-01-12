package main

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"log"
)

type rabbitConf struct {
	hostname string
	port int
	username string
	password string
}

func readRabbitConf() (rabbitConf) {
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

	return rabbitConf{hostname: hostname, port: port, username: username, password:password}
}

func connectRabbit(conf rabbitConf) *amqp.Connection {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/", conf.username, conf.password, conf.hostname, conf.port))
	failOnError(err, "failed to connect to rabbitmq")
	fmt.Println("connected to rabbitmq")
	return conn
}

func main() {
	rabbitConfig := readRabbitConf()
	conn := connectRabbit(rabbitConfig)

	defer conn.Close()
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}
