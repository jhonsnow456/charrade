package game

import (
	"math/rand/v2"
)

var words = []string{
	"octopus",
	"firefighter",
	"tooth fairy",
	"time machine",
	"guitar",
	"skydiving",
	"mime",
	"opera singer",
	"lighthouse",
	"penguin",
	"rocket launch",
	"magnet",
	"vacuum cleaner",
	"surfing",
	"chess grandmaster",
	"light bulb",
	"karaoke",
	"iceberg",
	"seesaw",
	"playing a video game",
	"escalator",
	"paper airplane",
	"hot air balloon",
	"washing machine",
	"ballet dancer",
	"snake charmer",
	"drum",
	"ladder",
	"volcano eruption",
	"making a sandwich",
	"doorbell",
	"tightrope walker",
	"paint roller",
	"neon sign",
	"zipper",
	"swing",
	"watering plants",
	"stampede",
	"ice skater",
	"typing on a typewriter",
}

// RandomWord draws a word from the list using the provided source.
func RandomWord(rng *rand.Rand) string {
	return words[rng.IntN(len(words))]
}
