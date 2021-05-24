package main

import (
	"encoding/json"
	"net/http"

	"github.com/Fardinak/websocket"

	"github.com/sirupsen/logrus"
)

const (
	MsgExploreCommand = websocket.Message("explore")
	MsgFlagCommand    = websocket.Message("flag")
	MsgStateRequest   = websocket.Message("state_request")

	MsgStateResponse = websocket.Message("state")
)

func main() {
	ws := websocket.NewWebSocketServer()
	ws.SetLogger(logrus.WithField("component", "websocket"))
	wsAPI := &WebSocketAPI{}

	subp := ws.Subprotocol("com.dowuz.minesweeper.api-v1")
	subp.RegisterReceiver(MsgExploreCommand, wsAPI.HandleExplore)
	subp.RegisterReceiver(MsgFlagCommand, wsAPI.HandleFlag)
	subp.RegisterReceiver(MsgStateRequest, wsAPI.HandleStateRequest)

	subp.RegisterSender(MsgStateResponse)

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

	_ = socket.Send(MsgStateResponse, response)
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

	_ = socket.Send(MsgStateResponse, response)
}

func (p WebSocketAPI) HandleStateRequest(socket *websocket.Socket, msg []byte) {
	var cmd StateRequest
	_ = json.Unmarshal(msg, &cmd)

	logrus.WithFields(logrus.Fields{
		"gid": cmd.GameID,
	}).Info("State request received")

	res := map[string]interface{}{}
	response, _ := json.Marshal(res)

	_ = socket.Send(MsgStateResponse, response)
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
