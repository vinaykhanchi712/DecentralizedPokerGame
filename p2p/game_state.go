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

type AtomicInt struct {
	value int32
}

func NewAtomicInt(value int32) *AtomicInt {
	return &AtomicInt{
		value: value,
	}
}

func (a *AtomicInt) String() string {
	return fmt.Sprintf("%d", a.value)
}

func (a *AtomicInt) Set(value int32) {
	atomic.StoreInt32(&a.value, value)
}

func (a *AtomicInt) Get() int32 {
	return atomic.LoadInt32(&a.value)
}

func (a *AtomicInt) Increment() {
	curr := a.Get()
	a.Set(curr + 1)
}

type PlayerActionRec struct {
	mu         sync.RWMutex
	recvAction map[string]PlayerAction
}

func NewPlayerActionRec() *PlayerActionRec {
	return &PlayerActionRec{
		recvAction: make(map[string]PlayerAction),
	}
}

func (pa *PlayerActionRec) addAction(from string, a PlayerAction) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	pa.recvAction[from] = a
}

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
	//status of a game/server atomically change
	CurrentStatus GameStatus
	//status of player at the moment
	CurrentPlayerAction *AtomicInt
	//atomic var to mark player dealer, when no one dealer default value is -1
	CurrentDealer     *AtomicInt
	CurrentPlayerTurn *AtomicInt

	//map to track ready players && should use lock for concurrency
	PlayersReady *PlayersReady

	RecvPlayerAction PlayerActionRec
	//list of connected players
	PlayerList PlayersList
}

func NewGame(addr string, broadcast chan BroadcastTo) *Game {

	game := &Game{
		ListenAdrr:    addr,
		broadcastch:   broadcast,
		CurrentStatus: Status_Connected,

		PlayersReady:        NewPlayerReady(),
		RecvPlayerAction:    *NewPlayerActionRec(),
		PlayerList:          PlayersList{},
		CurrentDealer:       NewAtomicInt(0),
		CurrentPlayerTurn:   NewAtomicInt(0),
		CurrentPlayerAction: NewAtomicInt(0),
	}

	game.PlayerList = append(game.PlayerList, addr)

	go game.loop()

	return game
}

func (g *Game) getCurrentDealerAddr() (string, bool) {

	currentDealer := g.PlayerList[g.CurrentDealer.Get()]

	return currentDealer, g.ListenAdrr == currentDealer
}

func (g *Game) canTakeAction(from string) bool {
	currentPlayerAddr := g.PlayerList[g.CurrentPlayerTurn.Get()]
	return from == currentPlayerAddr
}

func (g *Game) handlePlayerAction(from string, action MessagePlayerAction) error {
	if !g.canTakeAction(from) {
		return fmt.Errorf("player (%s) taking action before his turn", from)
	}
	g.RecvPlayerAction.addAction(from, action.Action)
	g.CurrentPlayerTurn.Increment()
	return nil
}

func (g *Game) TakeAction(action PlayerAction, val int) (err error) {
	//check if player is eligible to fold i.e. its his turn
	if !g.canTakeAction(g.ListenAdrr) {
		err = fmt.Errorf("i am taking action before its my turn (%s)", g.ListenAdrr)
		return
	}

	defer func() {
		if err == nil {
			g.CurrentPlayerTurn.Increment()
		}
	}()

	switch action {
	case PlayerActionFold:
		err = g.fold()
	case PlayerActionCheck:
		err = g.check()
	case PlayerActionBet:
		err = g.bet(val)
	default:
		err = fmt.Errorf("performing invalid action")
	}

	if err != nil {
		return err
	}

	g.broadcastPlayerActionToOtherPlayers(action, val)

	return
}

func (g *Game) bet(val int) error {
	g.CurrentPlayerAction.Set(int32(PlayerActionBet))

	logrus.WithFields(logrus.Fields{
		"we":         g.ListenAdrr,
		"bet placed": true,
		"bet value":  val,
	}).Info("Bet placed successfully")

	return nil
}

func (g *Game) check() error {
	g.CurrentPlayerAction.Set(int32(PlayerActionCheck))
	return nil
}

func (g *Game) fold() error {

	g.CurrentPlayerAction.Set(int32(PlayerActionFold))
	return nil
}

func (g *Game) broadcastPlayerActionToOtherPlayers(action PlayerAction, val int) {
	g.SendToPlayers(MessagePlayerAction{
		Action:            action,
		CurrentGameStatus: g.CurrentStatus,
		Value:             val,
	}, g.getOtherPlayers()...)
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

	if status == Status_PreFlop {
		g.CurrentPlayerTurn.Increment()
	}
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
				"we":                        g.ListenAdrr,
				"gameStatus":                g.CurrentStatus,
				"player":                    g.PlayerList,
				"currentDealer":             g.PlayerList[g.CurrentDealer.Get()],
				"nextTurn":                  g.PlayerList[g.CurrentPlayerTurn.Get()],
				"playerActionStatus":        g.RecvPlayerAction.recvAction,
				"currentPlayerActionStatus": PlayerAction(g.CurrentPlayerAction.Get()),
			}).Info("Info of connected player==>")
		default:
		}
	}
}

type PlayersList []string

func (list PlayersList) Len() int      { return len(list) }
func (list PlayersList) Swap(i, j int) { list[i], list[j] = list[j], list[i] }
func (list PlayersList) Less(i, j int) bool {
	portI, _ := strconv.Atoi(list[i][1:])
	portJ, _ := strconv.Atoi(list[j][1:])
	return portI < portJ
}
