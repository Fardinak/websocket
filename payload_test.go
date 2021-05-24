package websocket

import "testing"

func BenchmarkPayloadAppend(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var payload = []byte{99, 99, 99, 99}
		var message = "message_of_some_size"
		var msgLen = uint8(len(message))

		payload = append(payload, []byte(string(msgLen)+message)...)
		copy(payload[msgLen+1:], payload)
		copy(payload[1:], message)
		payload[0] = msgLen
	}
}

func BenchmarkPayloadNewSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var payload = []byte{99, 99, 99, 99}
		var message = "message_of_some_size"
		var msgLen = uint8(len(message))

		newPayload := make([]byte, len(payload)+len(message)+1)
		copy(newPayload[msgLen+1:], payload)
		copy(newPayload[1:], message)
		newPayload[0] = msgLen
	}
}
