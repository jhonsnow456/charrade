package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hey-amanthakur/charrade/apps/backend/internal/game"
)

var (
	validAvatar = regexp.MustCompile(`^avatar-([1-9]|1[0-4])$`)
	upgrader    = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

type Server struct {
	store    *RoomStore
	hub      *Hub
	newID    func() string
	guardsMu sync.Mutex
	guards   map[string]*sync.Mutex
}

func New() http.Handler {
	s := &Server{
		store:  NewRoomStore(),
		hub:    NewHub(),
		newID:  randomID,
		guards: make(map[string]*sync.Mutex),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/rooms", s.handleCreateRoom)
	mux.HandleFunc("GET /api/rooms/{id}", s.handleGetRoom)
	mux.HandleFunc("POST /api/rooms/{id}/players", s.handleAddPlayer)
	mux.HandleFunc("GET /api/rooms/{id}/ws", s.handleWebSocket)
	return withCORS(mux)
}

func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	var req createRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errMessage("invalid request body"))
		return
	}
	player, err := s.validatedPlayer(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errMessage(err.Error()))
		return
	}

	room := game.NewRoom(s.newID(), player)
	s.store.Create(room)
	writeJSON(w, http.StatusCreated, createRoomResponse{
		RoomID:   room.ID,
		PlayerID: player.ID,
		HostID:   room.HostID,
	})
}

func (s *Server) handleAddPlayer(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("id")
	room, ok := s.store.Get(roomID)
	if !ok {
		writeJSON(w, http.StatusNotFound, errMessage("room not found"))
		return
	}
	var req createRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errMessage("invalid request body"))
		return
	}
	player, err := s.validatedPlayer(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errMessage(err.Error()))
		return
	}
	g := s.roomGuard(roomID)
	g.Lock()
	err = s.store.Join(roomID, &player)
	g.Unlock()
	if err != nil {
		status := http.StatusConflict
		writeJSON(w, status, errMessage(err.Error()))
		return
	}
	s.broadcastState(roomID)
	writeJSON(w, http.StatusCreated, createRoomResponse{
		RoomID:   room.ID,
		PlayerID: player.ID,
		HostID:   room.HostID,
	})
}

func (s *Server) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("id")
	g := s.roomGuard(roomID)
	g.Lock()
	room, ok := s.store.Get(roomID)
	if !ok {
		g.Unlock()
		writeJSON(w, http.StatusNotFound, errMessage("room not found"))
		return
	}
	msg := stateFor(room)
	g.Unlock()
	writeJSON(w, http.StatusOK, msg)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("id")
	playerID := r.URL.Query().Get("playerId")

	room, ok := s.store.Get(roomID)
	if !ok {
		writeJSON(w, http.StatusNotFound, errMessage("room not found"))
		return
	}
	if _, ok := room.Player(playerID); !ok {
		writeJSON(w, http.StatusForbidden, errMessage("player not in room"))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}
	client := &Client{
		conn:     conn,
		send:     make(chan []byte, sendBuffer),
		playerID: playerID,
	}
	s.hub.Join(roomID, client)
	s.broadcastState(roomID)

	go client.writePump()
	go client.readPump(roomID, func(c *Client, msg clientMessage) error {
		broadcast, err := s.dispatch(roomID, c, msg)
		if err == nil && broadcast {
			s.broadcastState(roomID)
		}
		return err
	}, func() {
		s.hub.Leave(roomID, client)
		s.handlePlayerDisconnect(roomID, client.playerID)
	})
}

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
			if err := room.StartRound(room.HostID, roundDuration); err != nil {
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
			return room.StartRound(actorID, roundDuration)
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
		log.Printf("room %s: %s guessed %q correct=%v", roomID, c.playerID, msg.Text, correct)
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

func (s *Server) scheduleRoundDeadline(roomID string, startedAt time.Time) {
	delay := time.Until(startedAt.Add(roundDuration))
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
	time.AfterFunc(nextRoundDelay, func() {
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
			if err := room.StartRound(next, roundDuration); err != nil {
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

func (s *Server) roomGuard(roomID string) *sync.Mutex {
	s.guardsMu.Lock()
	defer s.guardsMu.Unlock()
	if s.guards[roomID] == nil {
		s.guards[roomID] = &sync.Mutex{}
	}
	return s.guards[roomID]
}

func (s *Server) withRoomLock(roomID string, fn func(*game.Room)) {
	g := s.roomGuard(roomID)
	g.Lock()
	defer g.Unlock()
	if room, ok := s.store.Get(roomID); ok {
		fn(room)
	}
}

func (s *Server) validatedPlayer(req createRoomRequest) (game.Player, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 24 {
		return game.Player{}, fmt.Errorf("name must be 1-24 characters")
	}
	if !validAvatar.MatchString(req.Avatar) {
		return game.Player{}, fmt.Errorf("invalid avatar %q", req.Avatar)
	}
	return game.Player{
		ID:     s.newID(),
		Name:   name,
		Avatar: req.Avatar,
	}, nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func mustJSON(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return raw
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func randomID() string {
	buf := make([]byte, 3)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}
