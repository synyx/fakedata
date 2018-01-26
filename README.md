# Fake Data Provider for RabbitMQ

This small program connects to a given RabbitMQ server, declares a topic where to which it
binds a queue on which it consumes so called *query messages*. Concrete names for exchange,
queue and routing key used to bind the queue have to be configured in a file config.properties.

The query messages need to provide a reply_to property that contains an AMQP address to 
which the response message shall be sent. The format is [exchange_name]/[routing_key]
or just [exchange_name].

The program will not fill the response with any dynamic data but with the contents of
a file found at the configured filename.

### Installation

Simply call 

```
go get github.com/rjayasinghe/fakedata
```

or get the base docker image to extend it:
```
docker pull rjayasinghe/fakedata
```

or clone the repository and call
```
go build fakedata.go
```

### Configuration

For a sample configuration you can have a look at https://github.com/rjayasinghe/fakedata-orders 

### Containerization

If you are running your data consuming application in a container environment it is
obvious to run this program in a container itself. Please have a look at
https://github.com/rjayasinghe/fakedata-orders for a sample conatiner definition
inheriting from the base image defined in this repository.

## TODO
* Tests ;-)
* Improve default configuration settings in case of missing config properties
