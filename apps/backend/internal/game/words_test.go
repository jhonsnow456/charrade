package game

import (
	"math/rand/v2"
	"testing"
)

func TestRandomWord(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for range 50 {
		word := RandomWord(rng)
		if word == "" {
			t.Fatal("RandomWord returned empty word")
		}
		if !wordInList(word) {
			t.Errorf("RandomWord %q not found in word list", word)
		}
	}
}

func TestRandomWordDeterministic(t *testing.T) {
	a := rand.New(rand.NewPCG(42, 1))
	b := rand.New(rand.NewPCG(42, 1))

	wa, wb := RandomWord(a), RandomWord(b)
	if wa != wb {
		t.Errorf("same seed produced different words: %q vs %q", wa, wb)
	}
}

func TestWordListNotEmpty(t *testing.T) {
	if len(words) == 0 {
		t.Fatal("word list is empty")
	}
}

func wordInList(w string) bool {
	for _, candidate := range words {
		if candidate == w {
			return true
		}
	}
	return false
}
