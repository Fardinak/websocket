package main

import (
	"encoding/json"
	"net/http"

	"github.com/Fardinak/websocket"

	"github.com/sirupsen/logrus"
)

func main() {
	ws := websocket.NewWebSocketServer()
	ws.SetLogger(logrus.WithField("component", "websocket"))
	wsAPI := &WebSocketAPI{}

	subp := ws.Subprotocol("com.dowuz.minesweeper.api-v1")
	subp.Handle("explore", wsAPI.HandleExplore)
	subp.Handle("flag", wsAPI.HandleFlag)
	subp.Handle("state", wsAPI.HandleStateRequest)

	http.Handle("/ws", ws)
	_ = http.ListenAndServe(":8080", nil)
}

type WebSocketAPI struct{}

func (p WebSocketAPI) HandleExplore(socket *websocket.Socket, msg []byte) {
	var cmd ActionCommand
	_ = json.Unmarshal(msg, &cmd)

	logrus.WithFields(logrus.Fields{
		"gid":  cmd.GameID,
		"cell": cmd.Cell,
	}).Info("Explore command received")

	res := map[string]interface{}{}
	response, _ := json.Marshal(res)

	_ = socket.Send("state", response)
}

func (p WebSocketAPI) HandleFlag(socket *websocket.Socket, msg []byte) {
	var cmd ActionCommand
	_ = json.Unmarshal(msg, &cmd)

	logrus.WithFields(logrus.Fields{
		"gid":  cmd.GameID,
		"cell": cmd.Cell,
	}).Info("Flag command received")

	res := map[string]interface{}{}
	response, _ := json.Marshal(res)

	_ = socket.Send("state", response)
}

func (p WebSocketAPI) HandleStateRequest(socket *websocket.Socket, msg []byte) {
	var cmd StateRequest
	_ = json.Unmarshal(msg, &cmd)

	logrus.WithFields(logrus.Fields{
		"gid": cmd.GameID,
	}).Info("State request received")

	res := map[string]interface{}{}
	response, _ := json.Marshal(res)

	_ = socket.Send("state", response)
}

type ActionCommand struct {
	GameID     string `json:"game_id"`
	Cell       int    `json:"cell"`
	Difficulty string `json:"difficulty"`
	Mode       string `json:"mode"`
	Tournament string `json:"tournament"`
}

type StateRequest struct {
	GameID string `json:"game_id"`
}
