package server

import (
	"DPokerGame/deck"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

type GameStatus int32

func (g GameStatus) String() string {
	switch g {
	case Waiting:
		return "READY"
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

type Player struct {
	Status GameStatus
}

type GameState struct {
	isDealer               bool
	listenAddr             string
	GameStatus             GameStatus
	broadcastch            chan any
	playersWaitingForCards atomic.Int32
	playersLock            sync.RWMutex
	players                map[string]*Player
}

func NewGameState(addr string, broadcast chan any) *GameState {
	g := &GameState{
		listenAddr:  addr,
		isDealer:    false,
		GameStatus:  Waiting,
		players:     make(map[string]*Player),
		broadcastch: broadcast,
	}
	go g.loop()

	return g
}

func (g *GameState) AddPlayer(addr string, status GameStatus) {
	g.playersLock.Lock()

	if status == Waiting {
		g.AddPlayerWaitingForCards()
	}

	g.players[addr] = &Player{
		Status: status,
	}

	g.playersLock.Unlock()

	g.SetPlayerStatus(addr, status)

	logrus.WithFields(logrus.Fields{
		"Addr":   addr,
		"status": status,
	}).Info("New Player joined")
}

func (g *GameState) SetStatus(status GameStatus) {
	atomic.StoreInt32((*int32)(&g.GameStatus), int32(status))
	g.GameStatus = status
}

func (g *GameState) SetPlayerStatus(addr string, status GameStatus) {

	players, ok := g.players[addr]
	if !ok {
		panic("player could not be found although it should exist")
	}

	players.Status = status

	go g.CheckNeedDealCards()
}

func (g *GameState) PlayersConnectedWithLock() int {
	g.playersLock.RLock()
	defer g.playersLock.RUnlock()
	return len(g.players)
}

func (g *GameState) loop() {
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			logrus.WithFields(logrus.Fields{
				"players connected": g.PlayersConnectedWithLock(),
				"status":            g.GameStatus.String(),
			}).Info("Info of connected players")
		default:
		}
	}
}

func (g *GameState) AddPlayerWaitingForCards() {
	g.playersWaitingForCards.Add(1)
}

func (g *GameState) CheckNeedDealCards() {
	time.Sleep(20 * time.Second)
	playersWaiting := g.playersWaitingForCards.Load()

	if playersWaiting == int32(g.PlayersConnectedWithLock()) && g.isDealer && g.GameStatus == Waiting {

		logrus.Info("Need to deal cards")
		//deal cards
		g.DealCards()
	}
}

func (g *GameState) DealCards() {
	deck := deck.NewDeck()

	msg := MessageCards{
		Deck: deck,
	}

	g.broadcastch <- msg
}
