package server

import (
	"sync"

	"github.com/hey-amanthakur/charrade/apps/backend/internal/game"
)

// RoomStore is the in-memory registry of rooms and their players.
type RoomStore struct {
	mu         sync.Mutex
	rooms      map[string]*game.Room
	playerRoom map[string]string
}

func NewRoomStore() *RoomStore {
	return &RoomStore{
		rooms:      make(map[string]*game.Room),
		playerRoom: make(map[string]string),
	}
}

func (s *RoomStore) Create(room *game.Room) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rooms[room.ID] = room
	for _, p := range room.Players {
		s.playerRoom[p.ID] = room.ID
	}
}

func (s *RoomStore) Get(id string) (*game.Room, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	room, ok := s.rooms[id]
	return room, ok
}

func (s *RoomStore) RoomOf(playerID string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	roomID, ok := s.playerRoom[playerID]
	return roomID, ok
}

// Join adds a player to a room and records the mapping.
func (s *RoomStore) Join(roomID string, player *game.Player) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	room, ok := s.rooms[roomID]
	if !ok {
		return game.ErrPlayerNotFound
	}
	if err := room.AddPlayer(*player); err != nil {
		return err
	}
	s.playerRoom[player.ID] = roomID
	return nil
}
