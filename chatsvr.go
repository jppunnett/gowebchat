package main

// Command chatsvr starts the web-based chat server.
//
// Usage: chatsvr [host:port] [chat URL]
//
// If [host:port] is not specified, then chatsvr defaults to localhost:8083
// If [chat URL] is not specified, then chat server defaults to host:port/chat
import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"

	"golang.org/x/net/websocket"
)

// Host and port on which to listen for incoming HTTP requests
var listenAddr string = "localhost:8083"
var chatURL string = listenAddr + "/chat"

// ingoreOriginHandler is a simple interface to a WebSocket browser client.
// It ingnores the Origin request header by default.
// See websocket.OriginHandler
type ingoreOriginHandler func(*websocket.Conn)

// ServeHTTP implements the http.Handler interface for a WebSocket but does
// not check the origin of the request header. Useful when testing on localhost
func (h ingoreOriginHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s := websocket.Server{Handler: websocket.Handler(h)}
	s.ServeHTTP(w, req)
}

// Start a chat. (For now, only echoing)
func chatHandler(ws *websocket.Conn) {
	io.Copy(ws, ws)
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

	err := http.ListenAndServe(listenAddr, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
