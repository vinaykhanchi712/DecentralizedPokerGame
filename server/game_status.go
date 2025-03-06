package server

type GameStatus int32

func (g GameStatus) String() string {
	switch g {
	case Waiting:
		return "READY"
	case ShuffleAndDeal:
		return "SHUFFLE AND DEAL"
	case Receiving:
		return "RECEIVING CARDS"
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
	ShuffleAndDeal
	Receiving
	Dealing
	PreFlop
	Flop
	Turn
	River
)
