package websocket

import (
	"errors"
	"fmt"
	"net/http"
)

var (
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
func (s *Subprotocol) Handle(topic string, handler MessageHandler) {
	if _, ok := s.handlers[topic]; ok {
		panic(fmt.Sprintf("websocket: duplicate handler registered (%s)", topic))
	}
	s.handlers[topic] = handler
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

func (s *Subprotocol) encodeMessage(topic string, payload []byte) (msg []byte, err error) {
	var msgLen uint8
	if l := len(topic); l < 256 {
		msgLen = uint8(l)
	} else {
		return nil, ErrMessageTooLong
	}

	// Prepend topic length and topic name to payload
	msg = make([]byte, len(payload)+len(topic)+1)
	msg[0] = msgLen
	copy(msg[1:], topic)
	copy(msg[msgLen+1:], payload)

	return payload, nil
}

func (s *Subprotocol) decodeMessage(msg []byte) (topic string, payload []byte) {
	msgLen := msg[0]
	return string(msg[1:msgLen+1]), msg[msgLen+1:]
}

func (s *Subprotocol) handleMessage(socket *Socket, msg []byte) {
	topic, payload := s.decodeMessage(msg)

	if handler, ok := s.handlers[topic]; ok {
		handler(socket, payload)
		return
	}

	if s.FallbackHandler != nil {
		s.FallbackHandler(socket, topic, payload)
	}
}

func (s *Subprotocol) SendToSocket(socket *Socket, topic string, payload []byte) error {
	msg, err := s.encodeMessage(topic, payload)
	if err != nil {
		return err
	}

	socket.send(msg)
	return nil
}

func (s *Subprotocol) SendToClient(id string, topic string, payload []byte) error {
	msg, err := s.encodeMessage(topic, payload)
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

func (s *Subprotocol) SendToRoom(name string, topic string, payload []byte) error {
	msg, err := s.encodeMessage(topic, payload)
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

func (s *Subprotocol) Broadcast(topic string, payload []byte) error {
	msg, err := s.encodeMessage(topic, payload)
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
