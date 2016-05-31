package main

// Command chatsvr starts the web-based chat server.
//
// Usage: chatsvr [host:port] [chat URL]
//
// If [host:port] is not specified, then chatsvr defaults to localhost:8083
// If [chat URL] is not specified, then chat server defaults to host:port/chat
//
// The approach was lifted (almost verbatim) from chapter 8 of "The Go
// Programming Language".

import (
	"bufio"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

// client represents a client
type client struct {
	name  string
	msgch chan string // Send messages to the client
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
	cliName := getClientName(wsc)
	cli := client{cliName, make(chan string)}
	go clientWriter(wsc, &cli)

	cli.msgch <- "You are " + cli.name
	messages <- cli.name + " has arrived"
	entering <- &cli

	input := bufio.NewScanner(wsc)
	wsc.SetReadDeadline(time.Now().Add(time.Minute * 30))
	for input.Scan() {
		if err := input.Err(); err != nil {
			log.Println("input.Err():", input.Err())
			break
		}
		wsc.SetReadDeadline(time.Now().Add(time.Minute * 30))
		messages <- cli.name + ": " + input.Text()
	}

	leaving <- &cli
	messages <- cli.name + " has left"
	wsc.Close()
}

func clientWriter(conn net.Conn, clip *client) {
	for msg := range clip.msgch {
		fmt.Fprint(conn, msg)
	}
}

// Allow the client to set their name
func getClientName(wsc *websocket.Conn) string {
	defaultName := wsc.Request().RemoteAddr

	// Assume that first word is client's name
	input := bufio.NewScanner(wsc)
	input.Split(bufio.ScanWords)
	input.Scan()
	if err := input.Err(); err != nil {
		log.Println("Error getting client name:", err)

		// Default name to ip:port
		return defaultName
	}

	name := input.Text()
	if strings.Trim(name, "-") == "" {
		name = defaultName
	}
	return name
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
			// Broadcast incoming message to all clients' outgoing message
			// channels.
			for cli := range clients {
				cli.msgch <- msg
			}

		case cli := <-entering:
			// Tell arriving client who else is here
			for c := range clients {
				cli.msgch <- c.name + " is here."
			}
			clients[cli] = true

		case cli := <-leaving:
			log.Printf("%v is leaving.\n", cli.name)
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
