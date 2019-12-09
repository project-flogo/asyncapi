# EFTL example

## Description
This example has an asyncapi application connect to an eftl server and consume messages.

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
Start the eftl server:
```bash
docker run --name eftl -d -p 9191:9191 -p 8585:8585 pointlander/eftl
```

In a new terminal build and start asyncapi eftl example:
```bash
asyncapi -input asyncapi.yml -type flogodescriptor
flogo create --cv v0.9.3-0.20190610180641-336db421a17a -f flogo.json eftlapp
mv support.go eftlapp/src/
cd eftlapp
flogo build
bin/eftlapp
```

In a new terminal send a eftl message:
```bash
go build
./eftl -client
```

You should see messages printed in the asyncapi eftl terminal.
