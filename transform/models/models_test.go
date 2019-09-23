package models

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestModels(t *testing.T) {
	files := [...]string{
		"../../examples/eftl/asyncapi.yml",
		"../../examples/eftl/asyncapi_secure.yml",
		"../../examples/http/asyncapi.yml",
		"../../examples/http/asyncapi_secure.yml",
		"../../examples/kafka/asyncapi.yml",
		"../../examples/kafka/asyncapi_secure.yml",
		"../../examples/mqtt/asyncapi.yml",
		"../../examples/mqtt/asyncapi_secure.yml",
		"../../examples/websocket/asyncapi.yml",
		"../../examples/websocket/asyncapi_secure.yml",
		"../../examples/streetlights/streetlights.yml",
	}
	for _, file := range files {
		api, err := Parse(file)
		if err != nil {
			t.Fatal(err)
		}
		_ = api
	}

	api, err := Parse(files[0])
	if err != nil {
		t.Fatal(err)
	}
	writer := bytes.NewBufferString("")
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", " ")
	err = encoder.Encode(&api)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(writer.String())
}
