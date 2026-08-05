package game

import (
	"testing"
	"time"
)

func TestNewRoom(t *testing.T) {
	host := Player{ID: "p1", Name: "Alice", Avatar: "avatar-1"}
	room := NewRoom("room-1", host)

	if room.ID != "room-1" {
		t.Errorf("room.ID = %q, want %q", room.ID, "room-1")
	}
	if room.Phase != PhaseLobby {
		t.Errorf("room.Phase = %q, want %q", room.Phase, PhaseLobby)
	}
	if room.HostID != "p1" {
		t.Errorf("room.HostID = %q, want %q", room.HostID, "p1")
	}
	if got := len(room.Players); got != 1 {
		t.Fatalf("len(room.Players) = %d, want 1", got)
	}
	if room.Players[0].Score != 0 {
		t.Errorf("host score = %d, want 0", room.Players[0].Score)
	}
}

func TestAddPlayer(t *testing.T) {
	room := NewRoom("room-1", Player{ID: "p1", Name: "Alice"})

	err := room.AddPlayer(Player{ID: "p2", Name: "Bob"})
	if err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	if got := len(room.Players); got != 2 {
		t.Fatalf("len(room.Players) = %d, want 2", got)
	}

	err = room.AddPlayer(Player{ID: "p2", Name: "Bob Again"})
	if err == nil {
		t.Error("AddPlayer duplicate: want error, got nil")
	}
}

func TestAddPlayerAfterStart(t *testing.T) {
	room := newPlayingRoom(t)

	err := room.AddPlayer(Player{ID: "p9", Name: "Eve"})
	if err == nil {
		t.Error("AddPlayer after start: want error, got nil")
	}
}

func TestRemovePlayer(t *testing.T) {
	room := NewRoom("room-1", Player{ID: "p1", Name: "Alice"})
	room.AddPlayer(Player{ID: "p2", Name: "Bob"})

	err := room.RemovePlayer("p2")
	if err != nil {
		t.Fatalf("RemovePlayer: %v", err)
	}
	if _, ok := room.Player("p2"); ok {
		t.Error("player p2 still present after removal")
	}
}

func TestRemovePlayerTransfersHost(t *testing.T) {
	room := NewRoom("room-1", Player{ID: "p1", Name: "Alice"})
	room.AddPlayer(Player{ID: "p2", Name: "Bob"})

	if err := room.RemovePlayer("p1"); err != nil {
		t.Fatalf("RemovePlayer: %v", err)
	}
	if room.HostID != "p2" {
		t.Errorf("room.HostID = %q, want transferred to p2", room.HostID)
	}
}

func TestRemovePlayerNotFound(t *testing.T) {
	room := NewRoom("room-1", Player{ID: "p1", Name: "Alice"})

	if err := room.RemovePlayer("nope"); err == nil {
		t.Error("RemovePlayer unknown: want error, got nil")
	}
}

func TestPlayer(t *testing.T) {
	room := NewRoom("room-1", Player{ID: "p1", Name: "Alice"})

	p, ok := room.Player("p1")
	if !ok {
		t.Fatal("Player p1 not found")
	}
	if p.Name != "Alice" {
		t.Errorf("player name = %q, want %q", p.Name, "Alice")
	}
	if _, ok := room.Player("missing"); ok {
		t.Error("Player missing: want not found")
	}
}

func TestStartRequiresAtLeastTwoPlayers(t *testing.T) {
	room := NewRoom("room-1", Player{ID: "p1", Name: "Alice"})

	err := room.Start()
	if err == nil {
		t.Error("Start with 1 player: want error, got nil")
	}
}

func TestStart(t *testing.T) {
	room := NewRoom("room-1", Player{ID: "p1", Name: "Alice"})
	room.AddPlayer(Player{ID: "p2", Name: "Bob"})

	if err := room.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if room.Phase != PhasePlaying {
		t.Errorf("room.Phase = %q, want %q", room.Phase, PhasePlaying)
	}
}

func TestStartTwice(t *testing.T) {
	room := newPlayingRoom(t)

	if err := room.Start(); err == nil {
		t.Error("Start while playing: want error, got nil")
	}
}

func TestStartRound(t *testing.T) {
	room := newPlayingRoom(t)
	room.pickWord = func() string { return "octopus" }

	err := room.StartRound("p1", time.Minute)
	if err != nil {
		t.Fatalf("StartRound: %v", err)
	}
	if room.Round == nil {
		t.Fatal("room.Round is nil after StartRound")
	}
	if room.Round.ActorID != "p1" {
		t.Errorf("round.ActorID = %q, want p1", room.Round.ActorID)
	}
	if room.Round.Word != "octopus" {
		t.Errorf("round.Word = %q, want octopus", room.Round.Word)
	}
	if room.Round.Duration != time.Minute {
		t.Errorf("round.Duration = %v, want 1m", room.Round.Duration)
	}
}

func TestStartRoundRequiresPlayingPhase(t *testing.T) {
	room := NewRoom("room-1", Player{ID: "p1", Name: "Alice"})

	if err := room.StartRound("p1", time.Minute); err == nil {
		t.Error("StartRound in lobby: want error, got nil")
	}
}

func TestStartRoundRequiresKnownActor(t *testing.T) {
	room := newPlayingRoom(t)

	if err := room.StartRound("ghost", time.Minute); err == nil {
		t.Error("StartRound unknown actor: want error, got nil")
	}
}

func TestStartRoundTwice(t *testing.T) {
	room := newPlayingRoom(t)
	room.pickWord = func() string { return "octopus" }

	if err := room.StartRound("p1", time.Minute); err != nil {
		t.Fatalf("StartRound: %v", err)
	}
	if err := room.StartRound("p2", time.Minute); err == nil {
		t.Error("StartRound while active: want error, got nil")
	}
}

func TestGuessCorrect(t *testing.T) {
	room := newPlayingRoom(t)
	room.pickWord = func() string { return "octopus" }
	room.StartRound("p1", time.Minute)

	correct, err := room.Guess("p2", "OCTOPUS")
	if err != nil {
		t.Fatalf("Guess: %v", err)
	}
	if !correct {
		t.Error("Guess correct = false, want true")
	}
	if room.Round.Completed != true {
		t.Error("round not completed after correct guess")
	}
	guesser, _ := room.Player("p2")
	if guesser.Score != 1 {
		t.Errorf("guesser score = %d, want 1", guesser.Score)
	}
	actor, _ := room.Player("p1")
	if actor.Score != 1 {
		t.Errorf("actor score = %d, want 1", actor.Score)
	}
}

func TestGuessWrong(t *testing.T) {
	room := newPlayingRoom(t)
	room.pickWord = func() string { return "octopus" }
	room.StartRound("p1", time.Minute)

	correct, err := room.Guess("p2", "squid")
	if err != nil {
		t.Fatalf("Guess: %v", err)
	}
	if correct {
		t.Error("Guess correct = true, want false")
	}
	if room.Round.Completed {
		t.Error("round completed on wrong guess")
	}
	if got := len(room.Round.Guesses); got != 1 {
		t.Fatalf("len(round.Guesses) = %d, want 1", got)
	}
	if room.Round.Guesses[0].Text != "squid" {
		t.Errorf("recorded guess text = %q, want squid", room.Round.Guesses[0].Text)
	}
}

func TestGuessWithoutActiveRound(t *testing.T) {
	room := NewRoom("room-1", Player{ID: "p1", Name: "Alice"})
	room.AddPlayer(Player{ID: "p2", Name: "Bob"})

	if _, err := room.Guess("p2", "octopus"); err == nil {
		t.Error("Guess without round: want error, got nil")
	}
}

func TestGuessByActor(t *testing.T) {
	room := newPlayingRoom(t)
	room.pickWord = func() string { return "octopus" }
	room.StartRound("p1", time.Minute)

	if _, err := room.Guess("p1", "octopus"); err == nil {
		t.Error("actor guessing own word: want error, got nil")
	}
}

func TestGuessAfterCompletion(t *testing.T) {
	room := newPlayingRoom(t)
	room.pickWord = func() string { return "octopus" }
	room.StartRound("p1", time.Minute)
	room.Guess("p2", "octopus")

	if _, err := room.Guess("p3", "octopus"); err == nil {
		t.Error("guess after round completed: want error, got nil")
	}
}

func TestEndRound(t *testing.T) {
	room := newPlayingRoom(t)
	room.pickWord = func() string { return "octopus" }
	room.StartRound("p1", time.Minute)

	if err := room.EndRound(); err != nil {
		t.Fatalf("EndRound: %v", err)
	}
	if !room.Round.Completed {
		t.Error("round not marked completed")
	}
	if err := room.EndRound(); err == nil {
		t.Error("EndRound on completed round: want error, got nil")
	}
}

func TestNextActorID(t *testing.T) {
	room := NewRoom("room-1", Player{ID: "p1", Name: "Alice"})
	room.AddPlayer(Player{ID: "p2", Name: "Bob"})
	room.AddPlayer(Player{ID: "p3", Name: "Carol"})

	next, err := room.NextActorID("p1")
	if err != nil {
		t.Fatalf("NextActorID: %v", err)
	}
	if next != "p2" {
		t.Errorf("NextActorID after p1 = %q, want p2", next)
	}
	next, _ = room.NextActorID("p3")
	if next != "p1" {
		t.Errorf("NextActorID wraps = %q, want p1", next)
	}
	if _, err := room.NextActorID("ghost"); err == nil {
		t.Error("NextActorID unknown: want error, got nil")
	}
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"  OCTOPUS  ": "octopus",
		"Star   Wars": "star wars",
		"  ":          "",
	}
	for input, want := range cases {
		if got := Normalize(input); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", input, got, want)
		}
	}
}

func newPlayingRoom(t *testing.T) *Room {
	t.Helper()
	room := NewRoom("room-1", Player{ID: "p1", Name: "Alice"})
	room.AddPlayer(Player{ID: "p2", Name: "Bob"})
	room.AddPlayer(Player{ID: "p3", Name: "Carol"})
	if err := room.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return room
}
