package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/hey-amanthakur/charrade/apps/backend/internal/game"
)

var (
	validAvatar = regexp.MustCompile(`^avatar-([1-9]|1[0-4])$`)
	upgrader    = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

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
		slog.Error("ws upgrade", "error", err)
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
