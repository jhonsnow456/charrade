package game

import (
	"errors"
	"time"
)

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

type Player struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Score  int    `json:"score"`
}

type Guess struct {
	PlayerID string    `json:"playerId"`
	Text     string    `json:"text"`
	Correct  bool      `json:"correct"`
	At       time.Time `json:"at"`
}

type Round struct {
	ActorID   string        `json:"actorId"`
	Word      string        `json:"word"`
	Duration  time.Duration `json:"duration"`
	StartedAt time.Time     `json:"startedAt"`
	Guesses   []Guess       `json:"guesses"`
	Completed bool          `json:"completed"`
}
