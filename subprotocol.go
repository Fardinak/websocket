package websocket

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrUnregisteredSender = errors.New("websocket: unregistered sender")
)

type Subprotocol struct {
	name          string
	senders       map[Message]messageCode
	receivers     map[Message]messageCode
	handlers      map[messageCode]MessageHandler
	lastSender    messageCode
	lastReceiver  messageCode
	openSockets   []*Socket
	clientSockets map[string][]*Socket
	openRooms     map[string][]string

	FallbackHandler func(socket *Socket, message messageCode, payload []byte)
	ClientID        func(r *http.Request) (clientID string)
}

type MessageHandler func(*Socket, []byte)

type Message string

type messageCode uint8

func newSubProtocol(name string) *Subprotocol {
	return &Subprotocol{
		name:         name,
		senders:      map[Message]messageCode{},
		receivers:    map[Message]messageCode{},
		handlers:     map[messageCode]MessageHandler{},
		lastSender:   0,
		lastReceiver: 0,

		FallbackHandler: nil,
		ClientID:        nil,
	}
}

func (s *Subprotocol) RegisterSender(message Message) {
	if _, ok := s.senders[message]; ok {
		panic(fmt.Sprintf("websocket: duplicate sender registered (%s)", message))
	}
	s.senders[message], s.lastSender = s.lastSender, s.lastSender+1
}

// RegisterReceiver defines a new message and assigns a handler to it. Handlers
// are not called concurrently and will block further messages from being
// handled on the same Socket to ensure message ordering
func (s *Subprotocol) RegisterReceiver(message Message, handler MessageHandler) {
	if _, ok := s.receivers[message]; ok {
		panic(fmt.Sprintf("websocket: duplicate receiver registered (%s)", message))
	}
	s.receivers[message], s.lastReceiver = s.lastReceiver, s.lastReceiver+1
	s.handlers[s.receivers[message]] = handler
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

func (s *Subprotocol) handleMessage(socket *Socket, msg []byte) {
	if handler, ok := s.handlers[messageCode(msg[0])]; ok {
		handler(socket, msg[1:])
		return
	}

	if s.FallbackHandler != nil {
		s.FallbackHandler(socket, messageCode(msg[0]), msg[1:])
	}
}

func (s *Subprotocol) SendToSocket(socket *Socket, message Message, payload []byte) error {
	code, ok := s.senders[message]
	if !ok {
		return ErrUnregisteredSender
	}

	socket.send(code, payload)
	return nil
}

func (s *Subprotocol) SendToClient(id string, message Message, payload []byte) error {
	code, ok := s.senders[message]
	if !ok {
		return ErrUnregisteredSender
	}

	if clientSockets, ok := s.clientSockets[id]; ok {
		for _, socket := range clientSockets {
			socket.send(code, payload)
		}
	}

	return nil
}

func (s *Subprotocol) SendToRoom(name string, message Message, payload []byte) error {
	code, ok := s.senders[message]
	if !ok {
		return ErrUnregisteredSender
	}

	if room, ok := s.openRooms[name]; ok {
		for _, cid := range room {
			for _, socket := range s.clientSockets[cid] {
				socket.send(code, payload)
			}
		}
	}

	return nil
}

func (s *Subprotocol) Broadcast(message Message, payload []byte) error {
	code, ok := s.senders[message]
	if !ok {
		return ErrUnregisteredSender
	}

	for _, socket := range s.openSockets {
		socket.send(code, payload)
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
