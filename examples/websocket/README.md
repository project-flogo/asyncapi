# Websocket example

## Description
This example has an asyncapi application connect to a websocket server and consume messages.

## Installation
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
Start the websocket server:
```bash
cd examples/websocket
go run helper.go
```

In a new terminal build and start asyncapi websocket example:
```bash
asyncapi -input asyncapi.yml -type flogodescriptor
flogo create --cv v0.9.3-0.20190610180641-336db421a17a -f flogo.json websocket
mv support.go websocket/src/
cd websocket
GOSUMDB=off flogo build
bin/websocket
```

You should see messages printed in the asyncapi websocket terminal.
