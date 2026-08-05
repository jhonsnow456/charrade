package server

import (
	"fmt"

	"github.com/hey-amanthakur/charrade/apps/backend/internal/game"
)

func (s *Server) relaySignal(room *game.Room, c *Client, msg clientMessage) error {
	if msg.To == "" {
		return fmt.Errorf("signal requires a target")
	}
	if len(msg.Payload) == 0 {
		return fmt.Errorf("signal payload is empty")
	}
	target, ok := room.Player(msg.To)
	if !ok {
		return fmt.Errorf("signal target not in room")
	}
	targetClient := s.hub.Find(room.ID, target.ID)
	if targetClient == nil {
		return fmt.Errorf("signal target not connected")
	}
	targetClient.sendBestEffort(mustJSON(signalMessage{
		Type:    "signal",
		From:    c.playerID,
		To:      target.ID,
		Payload: msg.Payload,
	}))
	return nil
}
