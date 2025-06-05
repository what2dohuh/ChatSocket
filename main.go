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
	Type     string `json:"type"`
	Room     string `json:"room"`
	Username string `json:"username"`
	Content  string `json:"content"`
	Online   int16  `json:"online"`
}
type Client struct {
	conn     *websocket.Conn
	username string
	room     string
}

var rooms = make(map[string]map[*websocket.Conn]*Client)
var Clients = make(map[*websocket.Conn]string)

func ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Default().Printf("Error while upgrading connection: %v", err)
		return
	}
	defer conn.Close()
	var client *Client

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

		if client == nil {
			client = &Client{
				conn:     conn,
				username: msg.Username,
				room:     msg.Room,
			}

			if rooms[client.room] == nil {
				rooms[client.room] = make(map[(*websocket.Conn)]*Client)
			}
			rooms[client.room][conn] = client
			continue
		}

		if msg.Type == "switchRoom" {
			oldroom := client.room
			delete(rooms[client.room], conn)
			LeaveNotice := message{
				Username: client.username,
				Content:  "has leaved the room",
				Online:   int16(len(Clients)),
				Type:     "system",
				Room:     oldroom,
			}
			outdata, _ := json.Marshal(LeaveNotice)
			for conn := range rooms[oldroom] {
				if err := conn.WriteMessage(messageType, outdata); err != nil {
					log.Default().Printf("Error while writing message: %v", err)
					conn.Close()
					delete(Clients, conn)
					return
				}
			}
			if rooms[client.room] == nil || len(rooms[client.room]) == 0 {
				delete(rooms, client.room)
			}
			client.room = msg.Room
			if rooms[client.room] == nil {
				rooms[client.room] = make(map[*websocket.Conn]*Client)
			}
			rooms[client.room][conn] = client

			joinNotice := message{
				Username: client.username,
				Content:  "has joined the room",
				Online:   int16(len(rooms[client.room])),
				Type:     "system",
				Room:     client.room,
			}
			outJoin, _ := json.Marshal(joinNotice)
			for c := range rooms[client.room] {
				c.WriteMessage(messageType, outJoin)
			}
		}

		outgoin := message{
			Username: msg.Username,
			Content:  msg.Content,
			Online:   int16(len(Clients)),
			Type:     msg.Type,
			Room:     msg.Room,
		}
		outdata, _ := json.Marshal(outgoin)

		for conn := range rooms[client.room] {

			if err := conn.WriteMessage(messageType, outdata); err != nil {
				log.Default().Printf("Error while writing message: %v", err)
				conn.Close()
				delete(Clients, conn)
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
