package p2p

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

type PlayersReady struct {
	mu          sync.RWMutex
	recStatutes map[string]bool
}

func NewPlayerReady() *PlayersReady {
	return &PlayersReady{
		recStatutes: make(map[string]bool),
	}
}
func (pr *PlayersReady) addRecStatus(from string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	logrus.WithFields(logrus.Fields{
		"playerAddr": from,
	}).Info("Rec New Player ready")

	pr.recStatutes[from] = true
}

func (pr *PlayersReady) Len() int {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	return len(pr.recStatutes)
}

func (pr *PlayersReady) clear() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.recStatutes = make(map[string]bool)
}

type Game struct {
	ListenAdrr  string
	broadcastch chan BroadcastTo
	//status of a player/server atomically change
	CurrentStatus GameStatus
	//atomic var to mark player dealer, when no one dealer default value is -1
	CurrentDealer int32

	//map to track ready players && should use lock for concurrency
	PlayersReady *PlayersReady

	//list of connected players
	PlayerList PlayersList
}

func NewGame(addr string, broadcast chan BroadcastTo) *Game {

	game := &Game{
		ListenAdrr:    addr,
		broadcastch:   broadcast,
		CurrentStatus: Status_Connected,

		PlayersReady: NewPlayerReady(),

		PlayerList:    PlayersList{},
		CurrentDealer: -1,
	}

	game.PlayerList = append(game.PlayerList, addr)

	go game.loop()

	return game
}

func (g *Game) getCurrentDealerAddr() (string, bool) {
	currentDealer := g.PlayerList[0] // if current dealer ==-1

	if g.CurrentDealer > -1 {
		currentDealer = g.PlayerList[g.CurrentDealer]
	}

	return currentDealer, g.ListenAdrr == currentDealer
}

/*
public set status func
*/
func (g *Game) SetStatus(s GameStatus) {
	//only update status when status if different
	g.setStatus(s)
}

/*
atomically set game status
*/
func (g *Game) setStatus(status GameStatus) {
	//only update status when status if different
	if g.CurrentStatus != status {
		atomic.StoreInt32((*int32)(&g.CurrentStatus), int32(status))
	}
}

/*
-> It receives message from other player saying they are ready.
so we mark them that they are ready
*/
func (g *Game) SetPlayerReady(from string) {
	logrus.WithFields(logrus.Fields{
		"we":     g.ListenAdrr,
		"Player": from,
	}).Info("Setting player status to ready")
	//mark other player ready
	g.PlayersReady.addRecStatus(from)

	/*
		-->Have a doubt here in the logic, cause we must have conditon on len of list of player list and
		len of player ready. then start some game by choosing one dealer, small blind etc.
	*/
	// not enough players are ready to start
	if g.PlayersReady.Len() < 2 {
		return
	}

	// for next round.
	//g.PlayersReady.clear()

	//We got enough players, we can start
	if _, ok := g.getCurrentDealerAddr(); ok {
		g.InitiateAndShuffleDeal()
	}

}

/*
--> it is used to set our own status/player ready from our button
and inform other player that we are ready.
*/
func (g *Game) SetReady() {
	//set our own player ready.
	g.PlayersReady.addRecStatus(g.ListenAdrr)
	g.setStatus(Status_Ready)
	//Inform other player that we are ready.
	g.SendToPlayers(MessageReady{}, g.getOtherPlayers()...)
}

func (g *Game) SendToPlayers(payload any, addr ...string) {
	g.broadcastch <- BroadcastTo{
		To:      addr,
		Payload: payload,
	}
	logrus.WithFields(logrus.Fields{
		"payload": payload,
		"player":  addr,
		"We":      g.ListenAdrr,
	}).Info("Sending payload to player")
}

func (g *Game) AddPlayer(from string) {
	//if a player is added to the game. we gonna assume he is ready to play
	g.PlayerList = append(g.PlayerList, from)
	sort.Sort(g.PlayerList)
}

func (g *Game) getOtherPlayers() []string {
	players := []string{}

	for _, addr := range g.PlayerList {
		if addr == g.ListenAdrr {
			continue
		}
		players = append(players, addr)
	}
	return players
}

// only used by real dealer
func (g *Game) InitiateAndShuffleDeal() {
	next := g.GetNextPositionOnTable()
	dealToPlayerAddr := g.PlayerList[next]

	g.setStatus(Status_Dealing)

	g.SendToPlayers(MessageEncDeck{Deck: [][]byte{}}, dealToPlayerAddr)

	logrus.WithFields(logrus.Fields{
		"dealToplayer": dealToPlayerAddr,
		"We":           g.ListenAdrr,
	}).Info("dealing cards")

}

func (g *Game) ShuffleAndEnc(from string, deck [][]byte) error {

	prevPos := g.PrevPlayerPosition()
	prevPlayerAddr := g.PlayerList[prevPos]

	if from != prevPlayerAddr {
		return fmt.Errorf("receievd enc deck from wrong player from (%s) should be (%s)", from, prevPlayerAddr)
	}

	_, isDealer := g.getCurrentDealerAddr()

	if isDealer && from == prevPlayerAddr {
		logrus.Info("Shuffle round complete")
		g.setStatus(Status_PreFlop)
		g.SendToPlayers(MessagePreFlop{}, g.getOtherPlayers()...)
		return nil
	}

	nextIndex, nextAddr := g.getNextReadyPlayer(g.getPositionOnTable())
	if nextIndex == -1 {
		logrus.Info("Shuffle round complete")
		return nil
	}
	dealToPlayer := nextAddr

	g.SendToPlayers(MessageEncDeck{Deck: [][]byte{}}, dealToPlayer)
	g.setStatus(Status_Dealing)
	/*

	 */
	return nil
}

// returns the index of the next player
func (g *Game) GetNextPositionOnTable() int {
	current := g.getPositionOnTable()
	if current == -1 {
		return -1
	}
	return (current + 1) % g.PlayerList.Len()
}

func (g *Game) getPositionOnTable() int {
	for i, p := range g.PlayerList {
		if g.ListenAdrr == p {
			return i
		}
	}
	// if pos not found
	return -1
}

func (g *Game) PrevPlayerPosition() int {
	current := g.getPositionOnTable()
	if current == 0 {
		return len(g.PlayerList) - 1
	}
	return current - 1
}

func (g *Game) getNextReadyPlayer(current int) (int, string) {
	start := current
	for {
		next := (start + 1) % g.PlayerList.Len()
		if next == current {
			fmt.Println("round trip completed")
			return -1, "-1"
		}
		if g.PlayersReady.recStatutes[g.PlayerList[next]] {
			return next, g.PlayerList[next]
		}

		start = next
	}
}

func (g *Game) loop() {
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			logrus.WithFields(logrus.Fields{
				"we":     g.ListenAdrr,
				"status": g.CurrentStatus,
				"player": g.PlayerList,
			}).Info("Info of connected player==>")
		default:
		}
	}
}

// type GameState struct {
// 	isDealer    bool
// 	listenAddr  string
// 	GameStatus  GameStatus
// 	broadcastch chan BroadcastTo

// 	playersLock sync.RWMutex
// 	players     map[string]*Player
// 	playerList  PlayersList
// }

// func NewGameState(addr string, broadcast chan BroadcastTo) *GameState {
// 	g := &GameState{
// 		listenAddr:  addr,
// 		isDealer:    false,
// 		GameStatus:  Waiting,
// 		players:     make(map[string]*Player),
// 		broadcastch: broadcast,
// 		playerList:  PlayersList{},
// 	}
// 	//add ourselves first
// 	g.AddPlayer(addr, Waiting)

// 	go g.loop()

// 	return g
// }

// func (g *GameState) AddPlayer(addr string, status GameStatus) {

// 	g.playersLock.Lock()
// 	defer g.playersLock.Unlock()

// 	player := &Player{
// 		Status:     status,
// 		ListenAddr: addr,
// 	}

// 	g.players[addr] = player

// 	g.playerList = append(g.playerList, player)
// 	sort.Sort(g.playerList)
// 	g.SetPlayerStatus(addr, status)

// 	logrus.WithFields(logrus.Fields{
// 		"Addr":   addr,
// 		"status": status,
// 	}).Info("New Player joined")
// }

// func (g *GameState) SetPlayerStatus(addr string, status GameStatus) {

// 	players, ok := g.players[addr]
// 	if !ok {
// 		panic("player could not be found although it should exist")
// 	}

// 	players.Status = status

// 	g.CheckNeedDealCards()
// }

// func (g *GameState) CheckNeedDealCards() {
// 	time.Sleep(4 * time.Second)
// 	playersWaiting := g.GetPlayersWaitingForCards()
// 	logrus.WithFields(logrus.Fields{
// 		"playerWaiting": playersWaiting,
// 		"conditionLen":  len(g.playerList),
// 	}).Info("+++++++++>>>>>>>>")
// 	if playersWaiting == len(g.playerList) && g.isDealer && g.GameStatus == Waiting {

// 		logrus.Info("Need to deal cards")
// 		//deal cards
// 		g.InitiateAndShuffleDeal()
// 	}
// }

// func (g *GameState) PrevPlayerPosition() int {
// 	current := g.getPositionOnTable()
// 	if current == 0 {
// 		return len(g.playerList) - 1
// 	}
// 	return -1
// }

// func (g *GameState) GetPlayersWaitingForCards() int {

// 	resp := 0

// 	for _, p := range g.playerList {
// 		if p.Status == Waiting {
// 			resp++
// 		}
// 	}
// 	return resp
// }

// func (g *GameState) ShuffleAndEnc(from string, deck [][]byte) error {
// 	g.SetPlayerStatus(from, ShuffleAndDeal)
// 	prevPos := g.PrevPlayerPosition()
// 	if prevPos != -1 {
// 		prev := g.playerList[prevPos]
// 		if g.isDealer && from == prev.ListenAddr {
// 			logrus.Info("Shuffle round complete")
// 			return nil
// 		}
// 	}
// 	next := g.GetNextPositionOnTable()
// 	dealToPlayer := g.playerList[next]

// 	g.SendToPlayer(dealToPlayer.ListenAddr, MessageEncDeck{Deck: [][]byte{}})
// 	g.SetStatus(ShuffleAndDeal)
// 	/*

// 	 */
// 	return nil
// }

// /*
// --------------------logic--------------------------------
// -- Bob (dealer)   , alice   , foo
// bob (shuffleAndEncrypt-with his key) -> alice (shuffleAndEnc- with her key)--> foo (shuffleAndEnc - with their key) --> bob

// */

// // only used by real dealer
// func (g *GameState) InitiateAndShuffleDeal() {
// 	next := g.GetNextPositionOnTable()
// 	dealToPlayer := g.playerList[next]
// 	g.SetStatus(ShuffleAndDeal)
// 	g.SendToPlayer(dealToPlayer.ListenAddr, MessageEncDeck{Deck: [][]byte{}})

// }

// func (g *GameState) getPositionOnTable() int {
// 	for i, p := range g.playerList {
// 		if g.listenAddr == p.ListenAddr {
// 			return i
// 		}
// 	}
// 	return -1
// }

// 	if len(g.playerList)-1 == current {
// 		return 0
// 	}

// 	return current + 1

// }

// func (g *GameState) SendToPlayer(addr string, payload any) error {
// 	g.broadcastch <- BroadcastTo{
// 		To:      []string{addr},
// 		Payload: payload,
// 	}

// 	logrus.WithFields(logrus.Fields{
// 		"from-player": g.listenAddr,
// 		"to-player":   addr,
// 		"payload":     payload,
// 	}).Info("[GameState] sending payload to player")
// 	return nil
// }

// func (g *GameState) SendToPlayerWithStatus(payload any, status GameStatus) {
// 	players := g.GetPlayerWithStatus(status)

// 	g.broadcastch <- BroadcastTo{
// 		To:      players,
// 		Payload: payload,
// 	}
// 	logrus.WithFields(logrus.Fields{
// 		"from":    g.listenAddr,
// 		"payload": payload,
// 		"to":      players,
// 	}).Info("Sending message ")
// }

// func (g *GameState) GetPlayerWithStatus(s GameStatus) []string {
// 	players := []string{}
// 	for addr, player := range g.players {
// 		if s == player.Status {
// 			players = append(players, addr)
// 		}
// 	}
// 	return players
// }

type PlayersList []string

func (list PlayersList) Len() int      { return len(list) }
func (list PlayersList) Swap(i, j int) { list[i], list[j] = list[j], list[i] }
func (list PlayersList) Less(i, j int) bool {
	portI, _ := strconv.Atoi(list[i][1:])
	portJ, _ := strconv.Atoi(list[j][1:])
	return portI < portJ
}

// type Player struct {
// 	Status     GameStatus
// 	ListenAddr string
// }

// func (p *Player) String() string {
// 	return fmt.Sprintf("[player [%s] : [%s] ]", p.ListenAddr, p.Status.String())
// }
