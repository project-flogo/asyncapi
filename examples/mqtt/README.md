# MQTT example

## Description
This example has an asyncapi application connect to a mqtt server and consume messages.

## Installation
* [Docker](https://www.docker.com/)
* [Go](https://golang.org/)
* [Flogo](https://github.com/project-flogo/cli)

## Setup
Install flogo with:
```bash
go get -u github.com/project-flogo/cli/...
```

Fetch and install asyncapi outside of your GOPATH:
```bash
git clone https://github.com/project-flogo/asyncapi.git
cd asyncapi
go install
```

## Testing
Start the mqtt server:
```bash
docker run -it -p 1883:1883 -p 9001:9001 eclipse-mosquitto
```

In a new terminal build and start asyncapi mqtt example:
```bash
asyncapi -input asyncapi.yml -type flogodescriptor
flogo create --cv v0.9.3-0.20190610180641-336db421a17a -f flogo.json mqtt
mv support.go mqtt/src/
cd mqtt
flogo build
bin/mqtt
```

In a new terminal send a mqtt message:
```bash
docker ps
docker exec -it <MOSQUITTO CONTAINER ID> /bin/sh
mosquitto_pub -m '{"message": "hello world"}' -t message/1
```

You should see messages printed in the asyncapi mqtt terminal.
