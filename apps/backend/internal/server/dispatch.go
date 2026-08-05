package server

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hey-amanthakur/charrade/apps/backend/internal/game"
)

func (s *Server) dispatch(roomID string, c *Client, msg clientMessage) (bool, error) {
	g := s.roomGuard(roomID)
	g.Lock()
	defer g.Unlock()

	room, ok := s.store.Get(roomID)
	if !ok {
		return false, fmt.Errorf("room not found")
	}

	switch msg.Type {
	case "start":
		hosted, err := s.requireHost(room, c, func() error {
			if err := room.Start(); err != nil {
				return err
			}
			if err := room.StartRound(room.HostID, s.cfg.RoundDuration); err != nil {
				return err
			}
			return nil
		})
		if err == nil && hosted {
			s.scheduleRoundDeadline(roomID, room.Round.StartedAt)
		}
		return hosted, err
	case "startRound":
		hosted, err := s.requireHost(room, c, func() error {
			actorID := room.HostID
			if room.Round != nil && room.Round.Completed {
				next, err := room.NextActorID(room.Round.ActorID)
				if err != nil {
					return err
				}
				actorID = next
			}
			return room.StartRound(actorID, s.cfg.RoundDuration)
		})
		if err == nil && hosted {
			s.scheduleRoundDeadline(roomID, room.Round.StartedAt)
		}
		return hosted, err
	case "endRound":
		hosted, err := s.requireHost(room, c, room.EndRound)
		if err == nil && hosted {
			s.scheduleNextRound(roomID)
		}
		return hosted, err
	case "guess":
		if msg.Text == "" {
			return false, fmt.Errorf("guess text is empty")
		}
		correct, err := room.Guess(c.playerID, msg.Text)
		if err != nil {
			return false, err
		}
		slog.Info("guess", "room", roomID, "player", c.playerID, "text", msg.Text, "correct", correct)
		if correct {
			s.scheduleNextRound(roomID)
		}
		return true, nil
	case "signal":
		return false, s.relaySignal(room, c, msg)
	default:
		return false, fmt.Errorf("unknown message type %q", msg.Type)
	}
}

func (s *Server) requireHost(room *game.Room, c *Client, fn func() error) (bool, error) {
	if c.playerID != room.HostID {
		return false, fmt.Errorf("only the host can do that")
	}
	return true, fn()
}

func (s *Server) scheduleRoundDeadline(roomID string, startedAt time.Time) {
	delay := time.Until(startedAt.Add(s.cfg.RoundDuration))
	if delay < 0 {
		delay = 0
	}
	time.AfterFunc(delay, func() {
		ended := false
		s.withRoomLock(roomID, func(room *game.Room) {
			if room.Round == nil || room.Round.Completed || !room.Round.StartedAt.Equal(startedAt) {
				return
			}
			if err := room.EndRound(); err == nil {
				ended = true
			}
		})
		if ended {
			s.scheduleNextRound(roomID)
		}
		s.broadcastState(roomID)
	})
}

func (s *Server) scheduleNextRound(roomID string) {
	time.AfterFunc(s.cfg.NextRoundDelay, func() {
		s.withRoomLock(roomID, func(room *game.Room) {
			if room.Phase != game.PhasePlaying {
				return
			}
			if room.Round == nil || !room.Round.Completed {
				return
			}
			next, err := room.NextActorID(room.Round.ActorID)
			if err != nil {
				return
			}
			if err := room.StartRound(next, s.cfg.RoundDuration); err != nil {
				return
			}
			s.scheduleRoundDeadline(roomID, room.Round.StartedAt)
		})
		s.broadcastState(roomID)
	})
}

func (s *Server) handlePlayerDisconnect(roomID, playerID string) {
	changed := false
	s.withRoomLock(roomID, func(room *game.Room) {
		if room.Phase == game.PhaseFinished {
			return
		}
		if s.hub.Find(roomID, playerID) != nil {
			return
		}
		wasActor := room.Round != nil && room.Round.ActorID == playerID && !room.Round.Completed
		if err := room.RemovePlayer(playerID); err != nil {
			return
		}
		changed = true
		if wasActor {
			if err := room.EndRound(); err == nil {
				s.scheduleNextRound(roomID)
			}
		}
		if room.Phase == game.PhasePlaying && len(room.Players) < 2 {
			room.Phase = game.PhaseFinished
		}
	})
	if changed {
		s.broadcastState(roomID)
	}
}

func (s *Server) broadcastState(roomID string) {
	g := s.roomGuard(roomID)
	g.Lock()
	room, ok := s.store.Get(roomID)
	if !ok {
		g.Unlock()
		return
	}
	type outbound struct {
		client *Client
		msg    []byte
	}
	msgs := make([]outbound, 0, 4)
	for _, client := range s.hub.Clients(roomID) {
		view := room.ViewFor(client.playerID)
		msgs = append(msgs, outbound{client, mustJSON(stateMessage{Type: "state", Room: view})})
	}
	g.Unlock()
	for _, m := range msgs {
		m.client.sendBestEffort(m.msg)
	}
}
