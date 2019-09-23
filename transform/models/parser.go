package models

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/asyncapi/parser/pkg/parser"
)

var (
	noopMessageProcessor parser.MessageProcessor = func(_ *map[string]interface{}) error {
		return nil
	}
)

// Parse parses the async api file
func Parse(file string) (api AsyncAPI200Schema, err error) {
	parse := noopMessageProcessor.BuildParse()
	writer := bytes.NewBufferString("")
	reader, err := os.Open(file)
	if err != nil {
		return api, err
	}
	err = parse(reader, writer)
	if err != nil {
		return api, err
	}
	err = json.Unmarshal(writer.Bytes(), &api)
	if err != nil {
		return api, err
	}
	return api, nil
}
