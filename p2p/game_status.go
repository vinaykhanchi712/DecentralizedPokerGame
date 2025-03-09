package p2p

type PlayerAction byte

const (
	PlayerActionFold  PlayerAction = iota + 1 //1
	PlayerActionCheck                         //2
	PlayerActionBet                           //3
)

type GameStatus int32

func (g GameStatus) String() string {
	switch g {
	case Status_Connected:
		return "PLAYER CONNECTED"
	case Status_Ready:
		return "PLAYER READY"
	case Status_Dealing:
		return "DEALING"
	case Status_PreFlop:
		return "PREFLOP"
	case Status_Flop:
		return "FLOP"
	case Status_Turn:
		return "TURN"
	case Status_River:
		return "RIVER"
	default:
		return "unknown"
	}
}

const (
	Status_Connected GameStatus = iota
	Status_Ready
	Status_Dealing
	Status_PreFlop
	Status_Flop
	Status_Turn
	Status_River
)
