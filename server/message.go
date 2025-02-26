package server

type Message struct {
	From    string
	Payload any
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
