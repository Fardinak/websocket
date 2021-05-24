package websocket

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type Socket struct {
	conn      *websocket.Conn
	subp      *Subprotocol
	clientID  string
	log       *logrus.Entry
	sendQueue chan []byte
}

func NewSocket(conn *websocket.Conn, subp *Subprotocol, log *logrus.Entry) *Socket {
	return &Socket{
		subp:      subp,
		conn:      conn,
		sendQueue: make(chan []byte),
		log:       log,
	}
}

// send queues the message to be sent
func (s *Socket) send(msg []byte) {
	s.sendQueue <- msg
}

func (s *Socket) readPump() {
	defer func() {
		_ = s.conn.Close()
	}()

	s.conn.SetReadLimit(maxMessageSize)
	err := s.conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		s.log.WithError(err).Warn("Failed to set read deadline")
	}

	s.conn.SetPongHandler(func(string) error {
		err := s.conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			s.log.WithError(err).Warn("Failed to set read deadline")
		}
		return nil
	})

	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.log.WithError(err).Error("Connection closed unexpectedly")
			}
			break
		}

		s.subp.handleMessage(s, message)
	}
}

func (s *Socket) writePump() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		_ = s.conn.Close()
	}()

	for {
		select {
		case message, ok := <-s.sendQueue:
			err := s.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				s.log.WithError(err).Warn("Failed to set write deadline")
			}

			if !ok {
				// The channel was closed
				err = s.conn.WriteMessage(websocket.CloseMessage, []byte{})
				if err != nil {
					s.log.WithError(err).Warn("Writing close message on channel close failed")
				}
				return
			}

			err = s.conn.WriteMessage(websocket.BinaryMessage, message)
			if err != nil {
				s.log.WithError(err).Error("Writing message failed")
			}

		case <-ticker.C:
			err := s.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				s.log.WithError(err).Warn("Failed to set write deadline")
			}

			if err = s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.log.WithError(err).Warn("Failed to write ping message")
				return
			}
		}
	}
}

func (s *Socket) Close() error {
	return s.conn.WriteMessage(websocket.CloseMessage, []byte{})
}

// Send is a helper method to send a message to current socket
func (s *Socket) Send(topic string, payload []byte) error {
	return s.subp.SendToSocket(s, topic, payload)
}

// SendToClient is a helper method to send a message to all sockets of the same client
func (s *Socket) SendToClient(topic string, payload []byte) error {
	return s.subp.SendToClient(s.clientID, topic, payload)
}
