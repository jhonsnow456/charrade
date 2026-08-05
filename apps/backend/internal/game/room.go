package game

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

type Room struct {
	ID      string    `json:"id"`
	HostID  string    `json:"hostId"`
	Phase   Phase     `json:"phase"`
	Players []*Player `json:"players"`
	Round   *Round    `json:"round,omitempty"`

	pickWord func() string
	now      func() time.Time
}

func NewRoom(id string, host Player) *Room {
	host.Score = 0
	return &Room{
		ID:       id,
		HostID:   host.ID,
		Phase:    PhaseLobby,
		Players:  []*Player{&host},
		pickWord: func() string { return RandomWord(rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 1))) },
		now:      time.Now,
	}
}

func (r *Room) Player(id string) (*Player, bool) {
	for _, p := range r.Players {
		if p.ID == id {
			return p, true
		}
	}
	return nil, false
}

func (r *Room) AddPlayer(p Player) error {
	if r.Phase != PhaseLobby {
		return fmt.Errorf("add player: %w", ErrGameInProgress)
	}
	if _, ok := r.Player(p.ID); ok {
		return fmt.Errorf("add player: %w", ErrPlayerAlreadyInRoom)
	}
	p.Score = 0
	r.Players = append(r.Players, &p)
	return nil
}

func (r *Room) RemovePlayer(id string) error {
	idx := -1
	for i, p := range r.Players {
		if p.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("remove player: %w", ErrPlayerNotFound)
	}
	r.Players = append(r.Players[:idx], r.Players[idx+1:]...)
	if r.HostID == id && len(r.Players) > 0 {
		r.HostID = r.Players[0].ID
	}
	return nil
}

func (r *Room) Start() error {
	if r.Phase != PhaseLobby {
		return fmt.Errorf("start: %w", ErrGameInProgress)
	}
	if len(r.Players) < 2 {
		return fmt.Errorf("start: %w", ErrNotEnoughPlayers)
	}
	r.Phase = PhasePlaying
	return nil
}

func (r *Room) StartRound(actorID string, duration time.Duration) error {
	if r.Phase != PhasePlaying {
		return fmt.Errorf("start round: game not in progress")
	}
	if r.Round != nil && !r.Round.Completed {
		return fmt.Errorf("start round: %w", ErrRoundAlreadyActive)
	}
	if _, ok := r.Player(actorID); !ok {
		return fmt.Errorf("start round: %w", ErrPlayerNotFound)
	}
	r.Round = &Round{
		ActorID:   actorID,
		Word:      r.pickWord(),
		Duration:  duration,
		StartedAt: r.now(),
		Guesses:   []Guess{},
	}
	return nil
}

func (r *Room) Guess(playerID, text string) (bool, error) {
	if r.Round == nil || r.Round.Completed {
		return false, fmt.Errorf("guess: %w", ErrNoActiveRound)
	}
	if r.Round.ActorID == playerID {
		return false, fmt.Errorf("guess: %w", ErrActorCannotGuess)
	}
	guesser, ok := r.Player(playerID)
	if !ok {
		return false, fmt.Errorf("guess: %w", ErrPlayerNotFound)
	}

	correct := Normalize(text) == Normalize(r.Round.Word)
	r.Round.Guesses = append(r.Round.Guesses, Guess{
		PlayerID: playerID,
		Text:     text,
		Correct:  correct,
		At:       r.now(),
	})
	if correct {
		r.Round.Completed = true
		guesser.Score++
		if actor, ok := r.Player(r.Round.ActorID); ok {
			actor.Score++
		}
	}
	return correct, nil
}

func (r *Room) EndRound() error {
	if r.Round == nil || r.Round.Completed {
		return fmt.Errorf("end round: %w", ErrRoundCompleted)
	}
	r.Round.Completed = true
	return nil
}

func (r *Room) NextActorID(currentID string) (string, error) {
	for i, p := range r.Players {
		if p.ID != currentID {
			continue
		}
		next := r.Players[(i+1)%len(r.Players)]
		return next.ID, nil
	}
	return "", fmt.Errorf("next actor: %w", ErrPlayerNotFound)
}

func Normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}
