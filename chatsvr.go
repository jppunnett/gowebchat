package main

// Command chatsvr starts the web-based chat server.
//
// Usage: chatsvr [port]
//
// If [port] is not specified, then chatsvr defaults to 8080
import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"

	"golang.org/x/net/websocket"
)

// port to listen on for incoming HTTP requests
var port string = "8080"

// Host and port on which to listen for incoming HTTP requests
var ListenAddr string

// Helper to return the "first" external IP that's up
// See https://play.golang.org/p/BDt3qEQ_2H
func externalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

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
func echoServer(ws *websocket.Conn) {
	io.Copy(ws, ws)
}

func rootHandler(w http.ResponseWriter, req *http.Request) {
	t, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, ListenAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	if len(os.Args) == 2 {
		port = os.Args[1]
	}

	if host, err := externalIP(); err != nil {
		panic("externalIP: " + err.Error())
	} else {
		ListenAddr = fmt.Sprintf("%s:%s", host, port)
		fmt.Printf("chatsvr listening at %v\n", ListenAddr)

		http.HandleFunc("/", rootHandler)
		http.Handle("/echo", ingoreOriginHandler(echoServer))

		err := http.ListenAndServe(ListenAddr, nil)
		if err != nil {
			panic("ListenAndServe: " + err.Error())
		}
	}

}
