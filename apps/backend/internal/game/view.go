package game

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
