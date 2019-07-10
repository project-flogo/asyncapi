package main

import (
	"flag"

	"github.com/project-flogo/asyncapi/transform"
)

func main() {
	input := flag.String("input", "asyncapi.yml", "input async api file")
	conversionType := flag.String("type", "flogoapiapp", "conversion type like flogoapiapp or flogodescriptor")
	role := flag.String("role", "server", "server or client; defaults to server")
	output := flag.String("output", ".", "path to store generated file")

	flag.Parse()
	transform.Transform(*input, *output, *conversionType, *role)
}
