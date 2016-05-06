package main

// Command chatsvr starts the web-based chat server.
//
// Usage: chatsvr [port]
//
// If [port] is not specified, then chatsvr defaults to 8080
import (
	"fmt"
	"io"
	"net/http"
	"os"

	"golang.org/x/net/websocket"
)

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

// Echo the data received on the WebSocket.
func EchoServer(ws *websocket.Conn) {
	io.Copy(ws, ws)
}

func main() {
	var port string = "8080"
	if len(os.Args) == 2 {
		port = os.Args[1]
	}

	http.Handle("/echo", ingoreOriginHandler(EchoServer))

	listenAddr := fmt.Sprintf("localhost:%v", port)
	fmt.Printf("chatsvr listening at %v\n", listenAddr)

	err := http.ListenAndServe(listenAddr, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
