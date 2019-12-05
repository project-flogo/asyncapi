# Kafka example

## Description
This example has an asyncapi application connect to a kafka server and consume messages.

## Installation
* [Docker](https://www.docker.com/)
* [Go](https://golang.org/)
* [Flogo](https://github.com/project-flogo/cli)
* [kafka-console-producer](https://github.com/Shopify/sarama/tree/master/tools/kafka-console-producer)

## Setup
Install flogo with:
```bash
go get -u github.com/project-flogo/cli/...
```

Install kafka-console-producer with:
```bash
go get github.com/Shopify/sarama/tools/kafka-console-producer
```

Fetch and install asyncapi outside of your GOPATH:
```bash
git clone https://github.com/project-flogo/asyncapi.git
cd asyncapi
go install
```

## Testing
Start kafak server:
```bash
cd examples/kafka
docker-compose up
```

In a new terminal build and start asyncapi kafka example:
```bash
asyncapi -input asyncapi.yml -type flogodescriptor
flogo create --cv v0.9.3-0.20190610180641-336db421a17a -f flogo.json kafka
mv support.go kafka/src/
cd kafka
flogo build
bin/kafka
```

In a new terminal run:
```bash
kafka-console-producer -topic message -value '{"message": "hello world"}' -brokers :9092
```

The message will be logged in the asyncapi kafka terminal.
