# HTTP example

## Description
This example has an asyncapi application receive http messages.

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
In a new terminal build and start asyncapi websocket example:
```bash
asyncapi -input asyncapi.yml -type flogodescriptor
flogo create --cv v0.9.3-0.20190610180641-336db421a17a -f flogo.json http
mv support.go http/src/
cd http
flogo build
bin/http
```

Now send some messages:
```bash
curl -d '{"message":"hello world"}' -H "Content-Type: application/json" -X POST http://localhost:1234/test/message
```

You should see messages printed in the asyncapi http terminal.
