package websocket

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrUnregisteredSender = errors.New("websocket: unregistered sender")
	ErrMessageTooLong = errors.New("websocket: message too long")
)

type Subprotocol struct {
	name          string
	handlers      map[string]MessageHandler
	openSockets   []*Socket
	clientSockets map[string][]*Socket
	openRooms     map[string][]string

	FallbackHandler func(socket *Socket, message string, payload []byte)
	ClientID        func(r *http.Request) (clientID string)
}

type MessageHandler func(*Socket, []byte)

func newSubProtocol(name string) *Subprotocol {
	return &Subprotocol{
		name:         name,
		handlers:     map[string]MessageHandler{},

		FallbackHandler: nil,
		ClientID:        nil,
	}
}

// Handle defines the handler for a message. Handlers are not called
// concurrently and will block further messages from being handled on the same
// Socket to ensure message ordering.
func (s *Subprotocol) Handle(message string, handler MessageHandler) {
	if _, ok := s.handlers[message]; ok {
		panic(fmt.Sprintf("websocket: duplicate handler registered (%s)", message))
	}
	s.handlers[message] = handler
}

func (s *Subprotocol) newConnection(r *http.Request, socket *Socket) {
	if s.ClientID != nil {
		socket.clientID = s.ClientID(r)
	}

	s.openSockets = append(s.openSockets, socket)
	if socket.clientID != "" {
		s.clientSockets[socket.clientID] = append(s.clientSockets[socket.clientID], socket)
	}
}

func (s *Subprotocol) encodeMessage(message string, payload []byte) (msg []byte, err error) {
	var msgLen uint8
	if l := len(message); l < 256 {
		msgLen = uint8(l)
	} else {
		return nil, ErrMessageTooLong
	}

	// Prepend message length and message name to payload
	msg = make([]byte, len(payload)+len(message)+1)
	copy(msg[msgLen+1:], payload)
	copy(msg[1:], message)
	msg[0] = msgLen

	return payload, nil
}

func (s *Subprotocol) decodeMessage(msg []byte) (message string, payload []byte) {
	msgLen := msg[0]
	return string(msg[1:msgLen+1]), msg[msgLen+1:]
}

func (s *Subprotocol) handleMessage(socket *Socket, msg []byte) {
	message, payload := s.decodeMessage(msg)

	if handler, ok := s.handlers[message]; ok {
		handler(socket, payload)
		return
	}

	if s.FallbackHandler != nil {
		s.FallbackHandler(socket, message, payload)
	}
}

func (s *Subprotocol) SendToSocket(socket *Socket, message string, payload []byte) error {
	msg, err := s.encodeMessage(message, payload)
	if err != nil {
		return err
	}

	socket.send(msg)
	return nil
}

func (s *Subprotocol) SendToClient(id string, message string, payload []byte) error {
	msg, err := s.encodeMessage(message, payload)
	if err != nil {
		return err
	}

	if clientSockets, ok := s.clientSockets[id]; ok {
		for _, socket := range clientSockets {
			socket.send(msg)
		}
	}

	return nil
}

func (s *Subprotocol) SendToRoom(name string, message string, payload []byte) error {
	msg, err := s.encodeMessage(message, payload)
	if err != nil {
		return err
	}

	if room, ok := s.openRooms[name]; ok {
		for _, cid := range room {
			for _, socket := range s.clientSockets[cid] {
				socket.send(msg)
			}
		}
	}

	return nil
}

func (s *Subprotocol) Broadcast(message string, payload []byte) error {
	msg, err := s.encodeMessage(message, payload)
	if err != nil {
		return err
	}

	for _, socket := range s.openSockets {
		socket.send(msg)
	}

	return nil
}

func (s *Subprotocol) JoinRoom(name string, socket *Socket) {
	if _, ok := s.openRooms[name]; !ok {
		s.openRooms[name] = []string{socket.clientID}
		return
	}

	// Prevent duplicate sockets in same room
	for _, cid := range s.openRooms[name] {
		if cid == socket.clientID {
			return
		}
	}

	s.openRooms[name] = append(s.openRooms[name], socket.clientID)
}

func (s *Subprotocol) LeaveRoom(name string, socket *Socket) {
	if room, ok := s.openRooms[name]; ok {
		for i := range room {
			if room[i] == socket.clientID {
				// If found, remove socket from room
				room[len(room)-1], room[i] = room[i], room[len(room)-1]
				room = room[:len(room)-1]
				break
			}
		}

		// If the room is empty, delete it
		if len(room) == 0 {
			delete(s.openRooms, name)
		}
	}
}
