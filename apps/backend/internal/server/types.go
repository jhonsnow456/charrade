package server

import (
	"encoding/json"
	"time"

	"github.com/hey-amanthakur/charrade/apps/backend/internal/game"
)

const (
	sendBuffer     = 64
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = pongWait * 9 / 10
	maxMessageSize = 64 * 1024
)

// createRoomRequest is the JSON body for room creation and joining.
type createRoomRequest struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// createRoomResponse is returned by the room creation and join endpoints.
type createRoomResponse struct {
	RoomID   string `json:"roomId"`
	PlayerID string `json:"playerId"`
	HostID   string `json:"hostId"`
}

// clientMessage is an inbound message from a browser over the WebSocket.
type clientMessage struct {
	Type    string          `json:"type"`
	Text    string          `json:"text"`
	To      string          `json:"to,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// signalMessage is a point-to-point WebRTC signaling message relayed between
// clients in the same room. Payload holds an SDP offer/answer or ICE candidate.
type signalMessage struct {
	Type    string          `json:"type"`
	From    string          `json:"from"`
	To      string          `json:"to,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

// stateMessage is the full room state broadcast after every mutation.
type stateMessage struct {
	Type string    `json:"type"`
	Room roomState `json:"room"`
}

// errorMessage is sent only to the client that caused a failed action.
type errorMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// roomState and playerState alias the game types so the transport layer can be
// swapped without touching the game engine.
type roomState = game.Room
type playerState = game.Player
type roundState = game.Round

func stateFor(room *game.Room) stateMessage {
	return stateMessage{Type: "state", Room: room.ViewFor("")}
}

func errMessage(text string) errorMessage {
	return errorMessage{Type: "error", Message: text}
}
