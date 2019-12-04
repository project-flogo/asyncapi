package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	startServer()
}

// startServer starts echo websocket server on localhost:8080/ws
func startServer() {
	middleware := http.NewServeMux()
	middleware.HandleFunc("/ws", wsHandler)
	server := http.Server{
		Addr:    "localhost:8000",
		Handler: middleware,
	}
	fmt.Println("Starting server with echo websocket service at ws://localhost:8000")
	log.Fatal(server.ListenAndServe())
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("received incomming request")

	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("upgrade error", err)
	} else {
		defer conn.Close()
		//upgraded to websocket connection
		clientAdd := conn.RemoteAddr()
		fmt.Println("Upgraded to websocket protocol")
		fmt.Println("Remote address:", clientAdd)

		for {
			conn.WriteMessage(websocket.TextMessage, []byte(`{"message": "hello world"}`))
			if err != nil {
				fmt.Println("write error", err)
				break
			}
			time.Sleep(time.Second)
		}
		return
	}
}
