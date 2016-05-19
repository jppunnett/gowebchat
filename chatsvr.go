package main

// Command chatsvr starts the web-based chat server.
//
// Usage: chatsvr [host:port] [chat URL]
//
// If [host:port] is not specified, then chatsvr defaults to localhost:8083
// If [chat URL] is not specified, then chat server defaults to host:port/chat
//
// The approach was lifted (almost verbatim)from chapter 8 of The Go Programming
// Language.

import (
	"bufio"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"

	"golang.org/x/net/websocket"
)

type client struct {
	name  string
	msgch chan string
}

var (
	// Host and port on which to listen for incoming HTTP requests
	listenAddr = "localhost:8083"
	// Websocket URL. Default is same as listen address
	chatURL = listenAddr + "/chat"

	entering = make(chan *client)
	leaving  = make(chan *client)
	messages = make(chan string) // all incoming client messages
)

// ingoreOriginHandler is a simple interface to a WebSocket browser client.
// It ignores the Origin request header by default.
// See websocket.OriginHandler
type ingoreOriginHandler func(*websocket.Conn)

// ServeHTTP implements the http.Handler interface for a WebSocket but does
// not check the origin of the request header. Useful when testing on localhost
func (h ingoreOriginHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s := websocket.Server{Handler: websocket.Handler(h)}
	s.ServeHTTP(w, req)
}

// Start a chat.
func chatHandler(wsc *websocket.Conn) {
	clip := &client{getChattersName(wsc), make(chan string)}
	// ch := make(chan string) // outgoing client messages
	go clientWriter(wsc, clip)

	clip.msgch <- "You are " + clip.name
	messages <- clip.name + " has arrived"
	entering <- clip

	input := bufio.NewScanner(wsc)
	for input.Scan() {
		fmt.Printf("err = %v", input.Err())
		says := input.Text()
		fmt.Printf(" says = %v\n", says)

		messages <- clip.name + ": " + says
	}
	// NOTE: ignoring potential errors from input.Err()

	leaving <- clip
	messages <- clip.name + " has left"
	wsc.Close()
}

func clientWriter(conn net.Conn, clip *client) {
	for msg := range clip.msgch {
		n, err := fmt.Fprint(conn, msg)
		fmt.Printf("bytes written = %d, err = %v\n", n, err)
	}
}

func getChattersName(wsc *websocket.Conn) string {
	// TODO: Let client send their name
	return wsc.Request().RemoteAddr
}

func rootHandler(w http.ResponseWriter, req *http.Request) {
	t, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, chatURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func broadcaster() {
	clients := make(map[*client]bool) // all connected clients
	for {
		select {
		case msg := <-messages:
			// Broadcast incoming message to all
			// clients' outgoing message channels.
			for cli := range clients {
				cli.msgch <- msg
			}

		case cli := <-entering:
			// Tell cli who else is in the chat
			for c := range clients {
				cli.msgch <- c.name + " is here."
			}
			clients[cli] = true

		case cli := <-leaving:
			delete(clients, cli)
			close(cli.msgch)
		}
	}
}

func main() {
	if len(os.Args) >= 2 {
		listenAddr = os.Args[1]
		if len(os.Args) > 2 {
			chatURL = os.Args[2]
		}
	}

	fmt.Printf("chatsvr listening at %v\n", listenAddr)

	http.HandleFunc("/", rootHandler)
	http.Handle("/chat", ingoreOriginHandler(chatHandler))

	go broadcaster()

	err := http.ListenAndServe(listenAddr, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
