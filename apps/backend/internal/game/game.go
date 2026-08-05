package game

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

// Phase describes the lifecycle of a room.
type Phase string

const (
	PhaseLobby    Phase = "lobby"
	PhasePlaying  Phase = "playing"
	PhaseFinished Phase = "finished"
)

var (
	ErrGameInProgress      = errors.New("game already in progress")
	ErrNotEnoughPlayers    = errors.New("need at least 2 players to start")
	ErrPlayerNotFound      = errors.New("player not found")
	ErrPlayerAlreadyInRoom = errors.New("player already in room")
	ErrNoActiveRound       = errors.New("no active round")
	ErrRoundAlreadyActive  = errors.New("a round is already active")
	ErrRoundCompleted      = errors.New("round is already completed")
	ErrActorCannotGuess    = errors.New("the actor cannot guess their own word")
)

// Player is a participant in a room.
type Player struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Score  int    `json:"score"`
}

// Guess is a single guess attempt submitted during a round.
type Guess struct {
	PlayerID string    `json:"playerId"`
	Text     string    `json:"text"`
	Correct  bool      `json:"correct"`
	At       time.Time `json:"at"`
}

// Round is a single acting turn.
type Round struct {
	ActorID   string        `json:"actorId"`
	Word      string        `json:"word"`
	Duration  time.Duration `json:"duration"`
	StartedAt time.Time     `json:"startedAt"`
	Guesses   []Guess       `json:"guesses"`
	Completed bool          `json:"completed"`
}

// Room holds players and the current game state.
type Room struct {
	ID      string    `json:"id"`
	HostID  string    `json:"hostId"`
	Phase   Phase     `json:"phase"`
	Players []*Player `json:"players"`
	Round   *Round    `json:"round,omitempty"`

	pickWord func() string
	now      func() time.Time
}

// NewRoom creates a room with a single host player in the lobby phase.
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

// Player returns the player with the given ID.
func (r *Room) Player(id string) (*Player, bool) {
	for _, p := range r.Players {
		if p.ID == id {
			return p, true
		}
	}
	return nil, false
}

// AddPlayer joins a new player to the room.
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

// RemovePlayer drops a player and, if they were host, transfers host to the
// first remaining player.
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

// Start moves the room from the lobby into the playing phase.
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

// StartRound begins a new round with the given actor, drawing a word from the
// word list. Only one round may be active at a time.
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

// Guess submits a guess for the active round. A correct guess completes the
// round and awards a point to the guesser and the actor.
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

// EndRound marks the active round complete without awarding points.
func (r *Room) EndRound() error {
	if r.Round == nil || r.Round.Completed {
		return fmt.Errorf("end round: %w", ErrRoundCompleted)
	}
	r.Round.Completed = true
	return nil
}

// NextActorID returns the player that follows the given actor, cycling around
// the player list.
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

// ViewFor returns a copy of the room safe to send to a given viewer. The
// round word is blanked out for everyone except the actor so teammates cannot
// see it.
func (r *Room) ViewFor(viewerID string) Room {
	out := *r
	out.Players = r.Players
	if r.Round != nil {
		round := *r.Round
		round.Guesses = append([]Guess{}, r.Round.Guesses...)
		if viewerID != r.Round.ActorID {
			round.Word = ""
		}
		out.Round = &round
	}
	return out
}

// Normalize lowercases and trims a guess so matches are forgiving.
func Normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}
