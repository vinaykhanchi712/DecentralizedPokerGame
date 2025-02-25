package server

type Message struct {
	From    string
	Payload any
}

type Handshake struct {
	GameVariant GameVariant
	Version     string
	GameStatus  GameStatus
}

type MessagePeerList struct {
	Peers []string
}
