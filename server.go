package websocket

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type WebSocketServer struct {
	log          *logrus.Entry
	upgrader     websocket.Upgrader
	subprotocols map[string]*Subprotocol
}

func NewWebSocketServer() *WebSocketServer {
	return &WebSocketServer{
		log: logrus.NewEntry(logrus.New()),
		upgrader: websocket.Upgrader{
			ReadBufferSize:    1024,
			WriteBufferSize:   1024,
			EnableCompression: true,
			Subprotocols:      []string{},
		},
		subprotocols: map[string]*Subprotocol{},
	}
}

func (ws *WebSocketServer) SetLogger(logger *logrus.Entry) {
	ws.log = logger
}

func (ws *WebSocketServer) Subprotocol(name string) (subp *Subprotocol) {
	if _, ok := ws.subprotocols[name]; ok {
		panic("WebSocketServer: Duplicate Subprotocol")
	}

	subp = newSubProtocol(name)
	ws.subprotocols[name] = subp
	ws.upgrader.Subprotocols = append(ws.upgrader.Subprotocols, name)

	return
}

// ServeHTTP implements the http.Handler interface
func (ws *WebSocketServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		ws.log.WithError(err).Error("Connection upgrade failed")
		return
	}

	// REVIEW: What happens if client specifies no subprotocols
	// use a default subprotocol to handle those cases

	subp := ws.subprotocols[conn.Subprotocol()]
	socket := NewSocket(conn, subp, ws.log)

	go socket.writePump()
	go socket.readPump()

	subp.newConnection(r, socket)
}
