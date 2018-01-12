package main

import (
	"fmt"
	"github.com/spf13/viper"
)

type RabbbitConf struct {
	hostname string
	port int
	username string
	password string
}




func main() {
	fmt.Println("vim-go")
	viper.SetConfigFile("config.properties")
	viper.SetConfigType("properties")
	confErr := viper.ReadInConfig()
	if confErr != nil {
		fmt.Println("No configuration file loaded - using defaults")
	}


	fooVal := viper.GetInt64("rabbitmq.port")
	fmt.Println(fooVal)
}
