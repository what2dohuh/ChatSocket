package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type message struct {
	Username string `json:"username"`
	Content  string `json:"content"`
	Online   int16  `json:"online"`
}

var Clients = make(map[*websocket.Conn]string)

func ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Default().Printf("Error while upgrading connection: %v", err)
		return
	}
	defer conn.Close()

	var username string

	Clients[conn] = ""
	log.Default().Printf("New client connected: %s", conn.RemoteAddr().String())

	for {
		messageType, msgdata, err := conn.ReadMessage()
		if err != nil {
			log.Default().Printf("Error while reading message: %v", err)
			delete(Clients, conn)
			return
		}
		var msg message
		if err := json.Unmarshal(msgdata, &msg); err != nil {
			log.Default().Printf("Error while unmarshalling message: %v", err)
		}

		if Clients[conn] == "" {
			Clients[conn] = msg.Username
		}
		username = Clients[conn]
		outgoin := message{
			Username: username,
			Content:  msg.Content,
			Online:   int16(len(Clients)),
		}
		outdata, _ := json.Marshal(outgoin)

		for client := range Clients {

			if err := client.WriteMessage(messageType, outdata); err != nil {
				log.Default().Printf("Error while writing message: %v", err)
				client.Close()
				delete(Clients, client)
				return
			}
		}
	}

}

func setUpRoutes() {
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/ws", ServeWs)
}

func home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the home page!")
}
func main() {
	setUpRoutes()
	fmt.Println("Server is running on port 8080")
	http.ListenAndServe(":8080", nil)

}
