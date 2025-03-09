package p2p

type Message struct {
	From    string
	Payload any
}
type BroadcastTo struct {
	To      []string
	Payload any
}

func NewMessage(from string, payload any) *Message {
	return &Message{
		From:    from,
		Payload: payload,
	}
}

type Handshake struct {
	GameVariant GameVariant
	Version     string
	GameStatus  GameStatus
	ListenAddr  string
}

type MessagePeerList struct {
	Peers []string
}

type MessageEncDeck struct {
	Deck [][]byte
}

type MessageReady struct{}

func (msg MessageReady) String() string {
	return "MSG: READY"
}

type MessagePreFlop struct {
}

func (msg MessagePreFlop) String() string {
	return "MSG: PreFlop"
}

type MessagePlayerAction struct {
	//current status of the player
	CurrentGameStatus GameStatus
	//action took by the player
	Action PlayerAction
	//value of the bet, if any
	Value int
}
