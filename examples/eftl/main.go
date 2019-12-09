package main

import (
	"flag"

	"github.com/project-flogo/eftl/lib"
)

var (
	client = flag.Bool("client", false, "send a message")
)

func main() {
	flag.Parse()

	if *client {
		errChannel := make(chan error, 1)
		options := &lib.Options{
			ClientID: "test",
		}
		connection, err := lib.Connect("ws://localhost:9191/channel", options, errChannel)
		if err != nil {
			panic(err)
		}
		defer connection.Disconnect()
		connection.Publish(lib.Message{
			"_dest":   "message",
			"content": []byte(`{"message": "hello world"}`),
		})
	} else {
		flag.PrintDefaults()
	}
}
