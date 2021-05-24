# WebSocket
A WebSocket Server with Subprotocol support.

## Features
### Socket handling
We use goroutines + channels to read from and write to a WebSocket.
PING/PONG messages are also handled internally.

### Subprotocol registry
An API is provided to create subprotocols and register messages to them.

### Message registry
This library requires you to define incoming/outgoing messages, allowing the WebSocket Server to assign an 8bit
code to each message for better efficiency. Messages are byte arrays, so you can use any encoder/decoder you like.

### Socket management
Socket operations are handled internally with sane defaults. There is an optional `ClientID` function that will run on
each new connection, allowing you to group sockets by user ID or any other identifier.

### Rooms
You can further group clients into rooms 

## TODO
- [ ] Unit tests
- [ ] JS-Client implementation
- [ ] Client implementation
- [ ] More configurability
- [ ] Default subprotocol
- [ ] Logging interface instead of `logrus.Entry`
- [ ] More comments
- [ ] JSONSubprotocol: a subprotocol that works with `interface{}` instead of `[]byte` and encodes to/decodes from JSON

## Credit
This library is a wrapper around the `gorilla/websocket` implementation.
