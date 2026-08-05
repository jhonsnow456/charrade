package game

import "testing"

func TestViewForHidesWordFromNonActors(t *testing.T) {
	room := newPlayingRoom(t)
	room.pickWord = func() string { return "octopus" }
	if err := room.StartRound("p1", 1); err != nil {
		t.Fatalf("StartRound: %v", err)
	}

	view := room.ViewFor("p2")
	if view.Round == nil {
		t.Fatal("view.Round is nil")
	}
	if view.Round.Word != "" {
		t.Errorf("non-actor saw word %q, want empty", view.Round.Word)
	}
	if view.Round.ActorID != "p1" {
		t.Errorf("view.Round.ActorID = %q, want p1", view.Round.ActorID)
	}
	if len(view.Players) != 3 {
		t.Errorf("len(view.Players) = %d, want 3", len(view.Players))
	}
}

func TestViewForShowsWordToActor(t *testing.T) {
	room := newPlayingRoom(t)
	room.pickWord = func() string { return "octopus" }
	if err := room.StartRound("p1", 1); err != nil {
		t.Fatalf("StartRound: %v", err)
	}

	view := room.ViewFor("p1")
	if view.Round == nil {
		t.Fatal("view.Round is nil")
	}
	if view.Round.Word != "octopus" {
		t.Errorf("actor saw word %q, want octopus", view.Round.Word)
	}
}

func TestViewForEmptyGuessesNotNil(t *testing.T) {
	room := newPlayingRoom(t)
	room.pickWord = func() string { return "octopus" }
	if err := room.StartRound("p1", 1); err != nil {
		t.Fatalf("StartRound: %v", err)
	}

	view := room.ViewFor("p2")
	if view.Round == nil {
		t.Fatal("view.Round is nil")
	}
	if view.Round.Guesses == nil {
		t.Error("view.Round.Guesses is nil, want empty slice so JSON is [] not null")
	}
}

func TestViewForNoRound(t *testing.T) {
	room := NewRoom("room-1", Player{ID: "p1", Name: "Alice"})

	view := room.ViewFor("p1")
	if view.Round != nil {
		t.Error("view.Round should be nil before any round starts")
	}
}
