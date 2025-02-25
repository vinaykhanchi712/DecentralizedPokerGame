package server

type GameStatus uint32

func (g GameStatus) String() string {
	switch g {
	case Waiting:
		return "WAITING"
	case Dealing:
		return "DEALING"
	case PreFlop:
		return "PREFLOP"
	case Flop:
		return "FLOP"
	case Turn:
		return "TURN"
	case River:
		return "RIVER"
	default:
		return "unknown"
	}
}

const (
	Waiting GameStatus = iota
	Dealing
	PreFlop
	Flop
	Turn
	River
)

type GameState struct {
	isDealer   bool
	GameStatus GameStatus
}

func NewGameState() *GameState {
	return &GameState{}
}

func (g *GameState) loop() {
	return
}
