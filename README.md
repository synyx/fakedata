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

or clone the repository and call

```
go build fakedata.go
```

### Configuration

You will need a configuration file that is named config.properties:

```
rabbitmq.port=5672
rabbitmq.hostname=localhost
rabbitmq.username=guest
rabbitmq.password=guest
# how the queries exchange is named
rabbitmq.queries.exchange=queries
# which queue shall be created for this fakedata instance
rabbitmq.queries.queue=fakedata.queries
# which routing key is used to bind the created queue
rabbitmq.queries.routingkey=orders.all
# how long to wait after a connection failure
rabbitmq.timeout=5s
# the name of the datafile
filename=orders.json
```


### Containerization

If you are running your data consuming application in a container environment it is
obvious to run this program in a container itself.

Configuration and data files are need to reside in the same directory as the executable itself. In a docker-compose scenario you'd mount as volumes like this:

```
    volumes:
        - ./docker/fakedata-orders/config.properties:/config.properties
        - ./docker/fakedata-orders/orders.json:/orders.json
```

## TODO

* Tests ;-)
