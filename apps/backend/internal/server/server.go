package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"

	"github.com/hey-amanthakur/charrade/apps/backend/internal/config"
	"github.com/hey-amanthakur/charrade/apps/backend/internal/game"
)

type Server struct {
	store    *RoomStore
	hub      *Hub
	cfg      config.Config
	newID    func() string
	guardsMu sync.Mutex
	guards   map[string]*sync.Mutex
}

func New(cfg config.Config) http.Handler {
	s := &Server{
		store:  NewRoomStore(),
		hub:    NewHub(),
		cfg:    cfg,
		newID:  randomID,
		guards: make(map[string]*sync.Mutex),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/rooms", s.handleCreateRoom)
	mux.HandleFunc("GET /api/v1/rooms/{id}", s.handleGetRoom)
	mux.HandleFunc("POST /api/v1/rooms/{id}/players", s.handleAddPlayer)
	mux.HandleFunc("GET /api/v1/rooms/{id}/ws", s.handleWebSocket)
	mux.HandleFunc("GET /api/v1/health", handleHealth)
	return withCORS(mux)
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

func randomID() string {
	buf := make([]byte, 3)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}
