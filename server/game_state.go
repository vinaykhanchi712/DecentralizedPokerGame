package server

type Round uint32

const (
	Dealing Round = iota
	PreFlop
	Flop
	Turn
	River
)

type GameState struct {
	Round uint32
}

func NewGameState() *GameState {
	return &GameState{}
}
