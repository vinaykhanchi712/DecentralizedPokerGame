package server

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

type PlayersList []*Player

func (list PlayersList) Len() int      { return len(list) }
func (list PlayersList) Swap(i, j int) { list[i], list[j] = list[j], list[i] }
func (list PlayersList) Less(i, j int) bool {
	portI, _ := strconv.Atoi(list[i].ListenAddr[1:])
	portJ, _ := strconv.Atoi(list[j].ListenAddr[1:])
	return portI < portJ
}

type Player struct {
	Status     GameStatus
	ListenAddr string
}

func (p *Player) String() string {
	return fmt.Sprintf("[player [%s] : [%s] ]", p.ListenAddr, p.Status.String())
}

type GameState struct {
	isDealer    bool
	listenAddr  string
	GameStatus  GameStatus
	broadcastch chan BroadcastTo

	playersLock sync.RWMutex
	players     map[string]*Player
	playerList  PlayersList
}

func NewGameState(addr string, broadcast chan BroadcastTo) *GameState {
	g := &GameState{
		listenAddr:  addr,
		isDealer:    false,
		GameStatus:  Waiting,
		players:     make(map[string]*Player),
		broadcastch: broadcast,
		playerList:  PlayersList{},
	}
	//add ourselves first
	g.AddPlayer(addr, Waiting)

	go g.loop()

	return g
}

func (g *GameState) AddPlayer(addr string, status GameStatus) {

	g.playersLock.Lock()
	defer g.playersLock.Unlock()

	player := &Player{
		Status:     status,
		ListenAddr: addr,
	}

	g.players[addr] = player

	g.playerList = append(g.playerList, player)
	sort.Sort(g.playerList)
	g.SetPlayerStatus(addr, status)

	logrus.WithFields(logrus.Fields{
		"Addr":   addr,
		"status": status,
	}).Info("New Player joined")
}

func (g *GameState) SetStatus(status GameStatus) {
	//only update status when status if different
	if g.GameStatus != status {
		atomic.StoreInt32((*int32)(&g.GameStatus), int32(status))
		g.SetPlayerStatus(g.listenAddr, status)
	}
}

func (g *GameState) SetPlayerStatus(addr string, status GameStatus) {

	players, ok := g.players[addr]
	if !ok {
		panic("player could not be found although it should exist")
	}

	players.Status = status

	g.CheckNeedDealCards()
}

func (g *GameState) loop() {
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			logrus.WithFields(logrus.Fields{
				"addr":   g.listenAddr,
				"status": g.GameStatus,
				"player": g.playerList,
			}).Info("Info of connected player")
		default:
		}
	}
}

func (g *GameState) CheckNeedDealCards() {
	time.Sleep(4 * time.Second)
	playersWaiting := g.GetPlayersWaitingForCards()
	logrus.WithFields(logrus.Fields{
		"playerWaiting": playersWaiting,
		"conditionLen":  len(g.playerList),
	}).Info("+++++++++>>>>>>>>")
	if playersWaiting == len(g.playerList) && g.isDealer && g.GameStatus == Waiting {

		logrus.Info("Need to deal cards")
		//deal cards
		g.InitiateAndShuffleDeal()
	}
}

func (g *GameState) PrevPlayerPosition() int {
	current := g.getPositionOnTable()
	if current == 0 {
		return len(g.playerList) - 1
	}
	return -1
}

func (g *GameState) GetPlayersWaitingForCards() int {

	resp := 0

	for _, p := range g.playerList {
		if p.Status == Waiting {
			resp++
		}
	}
	return resp
}

func (g *GameState) ShuffleAndEnc(from string, deck [][]byte) error {
	g.SetPlayerStatus(from, ShuffleAndDeal)
	prevPos := g.PrevPlayerPosition()
	if prevPos != -1 {
		prev := g.playerList[prevPos]
		if g.isDealer && from == prev.ListenAddr {
			logrus.Info("Shuffle round complete")
			return nil
		}
	}
	next := g.GetNextPositionOnTable()
	dealToPlayer := g.playerList[next]

	g.SendToPlayer(dealToPlayer.ListenAddr, MessageEncDeck{Deck: [][]byte{}})
	g.SetStatus(ShuffleAndDeal)
	/*

	 */
	return nil
}

/*
--------------------logic--------------------------------
-- Bob (dealer)   , alice   , foo
bob (shuffleAndEncrypt-with his key) -> alice (shuffleAndEnc- with her key)--> foo (shuffleAndEnc - with their key) --> bob

*/

// only used by real dealer
func (g *GameState) InitiateAndShuffleDeal() {
	next := g.GetNextPositionOnTable()
	dealToPlayer := g.playerList[next]
	g.SetStatus(ShuffleAndDeal)
	g.SendToPlayer(dealToPlayer.ListenAddr, MessageEncDeck{Deck: [][]byte{}})

}

func (g *GameState) getPositionOnTable() int {
	for i, p := range g.playerList {
		if g.listenAddr == p.ListenAddr {
			return i
		}
	}
	return -1
}

// returns the index of the next player
func (g *GameState) GetNextPositionOnTable() int {
	current := g.getPositionOnTable()
	if current == -1 {
		return -1
	}

	if len(g.playerList)-1 == current {
		return 0
	}

	return current + 1

}

func (g *GameState) SendToPlayer(addr string, payload any) error {
	g.broadcastch <- BroadcastTo{
		To:      []string{addr},
		Payload: payload,
	}

	logrus.WithFields(logrus.Fields{
		"from-player": g.listenAddr,
		"to-player":   addr,
		"payload":     payload,
	}).Info("[GameState] sending payload to player")
	return nil
}

func (g *GameState) SendToPlayerWithStatus(payload any, status GameStatus) {
	players := g.GetPlayerWithStatus(status)

	g.broadcastch <- BroadcastTo{
		To:      players,
		Payload: payload,
	}
	logrus.WithFields(logrus.Fields{
		"from":    g.listenAddr,
		"payload": payload,
		"to":      players,
	}).Info("Sending message ")
}

func (g *GameState) GetPlayerWithStatus(s GameStatus) []string {
	players := []string{}
	for addr, player := range g.players {
		if s == player.Status {
			players = append(players, addr)
		}
	}
	return players
}
