# asyncapi
[AsyncAPI](https://github.com/asyncapi/asyncapi) to flogo app converter tool converts given AsyncAPI spec to its implementation based on flogo api/descriptor model using the [Microgateway Action](https://github.com/project-flogo/microgateway).

Currently this tool accepts below arguments.
```sh
Usage of asyncapi:
  -type string
        conversion type like flogoapiapp or flogodescriptor (default "flogoapiapp")
  -input string
        input async api file (default "asyncapi.yml")
  -output string
        path to store generated file (default ".")
```

## Setup
To install the tool, simply open a terminal and enter the below commands
```sh
git clone https://github.com/project-flogo/asyncapi.git
cd asyncapi/
go install
```

## Usage
### Flogo app api model.
```sh
cd asyncapi/
mdkir test
asyncapi -input examples/http/asyncapi.yml -type flogoapiapp -output test/
```
The resulting output is `app.go` which can be built into a working flogo application:
```sh
cd test
go build
./test
```

### Flogo app descriptor model.
```sh
cd asyncapi/
asyncapi -input examples/http/asyncapi.yml -type flogodescriptor
```
The resulting output is `flogo.json` which can be built into a working flogo application:
```sh
flogo create -f flogo.json flogoapp
mv support.go flogoapp/src/
cd flogoapp
flogo build
./bin/flogoapp
```

## Flogo Plugin Support
This tool can be integrated into [flogocli](https://github.com/project-flogo/cli).
```sh
# Install your plugin
$ flogo plugin install github.com/project-flogo/asyncapi/cmd

# Run your new plugin command for api app model
$ flogo asyncapi -i asyncapi.yml -t flogoapiapp  -o test/

# Run your new plugin command for descriptor app model
$ flogo asyncapi -i asyncapi.yml -t flogodescriptor

# Remove your plugin
$ flogo plugin remove github.com/project-flogo/asyncapi/cmd
```
