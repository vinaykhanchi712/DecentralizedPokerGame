package server

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type GameVariant uint8

const (
	TexasHoldem GameVariant = iota
	Other
)

func (g *GameVariant) string() string {
	switch *g {
	case TexasHoldem:
		return "TEXAS HOLDEM"
	case Other:
		return "Other"
	default:
		return "Unknown"
	}
}

type ServerConfig struct {
	Version     string
	ListenAddr  string
	GameVariant GameVariant
}

type Server struct {
	ServerConfig //this is struct embedding
	transport    *TCPTransport
	peerLock     sync.RWMutex
	peers        map[string]*Peer
	addPeer      chan *Peer
	msgChan      chan *Message
	delPeer      chan *Peer
	broadcastch  chan any
	GameState    *GameState
}

func NewServer(cnf ServerConfig) *Server {
	s := &Server{
		ServerConfig: cnf,
		peers:        make(map[string]*Peer),
		addPeer:      make(chan *Peer),
		msgChan:      make(chan *Message),
		delPeer:      make(chan *Peer),
		broadcastch:  make(chan any),
	}

	s.GameState = NewGameState(s.ListenAddr, s.broadcastch)

	// TO DO -----------------------hardcoded for now
	if s.ListenAddr == "127.0.0.1:3000" {
		s.GameState.isDealer = true
	}

	s.transport = NewTCPTransport(cnf.ListenAddr)
	s.transport.addPeer = s.addPeer
	s.transport.delPeer = s.delPeer
	return s
}

func (s *Server) Start() {

	go s.loop()

	logrus.WithFields(logrus.Fields{
		"Port":    s.ListenAddr,
		"Variant": s.GameVariant.string(),
	}).Info("Starting new game server on ")
	s.transport.ListenAndAccept()
}

func (s *Server) loop() {
	for {
		select {
		case peer := <-s.delPeer:
			addr := peer.listenAddr
			delete(s.peers, addr)
			logrus.Info("Player disconnected: ", addr)
		case peer := <-s.addPeer:

			if err := s.handleNewPeer(peer); err != nil {
				logrus.Error("Error handling new peer", err)
				continue
			}

		case msg := <-s.msgChan:
			go func() {
				if err := s.handleMessage(msg); err != nil {
					logrus.Error("Error handling message", err)
				}
			}()
		case msg := <-s.broadcastch:
			if err := s.Broadcast(msg); err != nil {
				logrus.Errorf("error broadcasting msg : %+v", err)
			}
		}

	}
}

func (s *Server) Connect(addr string) error {

	if s.isInPeerList(addr) {
		return nil
	}

	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		return err
	}

	peer := &Peer{
		conn:     conn,
		Outbound: true,
	}

	s.addPeer <- peer
	return s.SendHandshake(peer)
}

func (s *Server) handleMessage(msg *Message) error {
	logrus.WithFields(logrus.Fields{
		"From":    msg.From,
		"Payload": msg.Payload,
	}).Info("Received message from")

	switch v := msg.Payload.(type) {
	case MessagePeerList:
		return s.handlePeerList(v)
	case MessageCards:
		logrus.WithFields(logrus.Fields{
			"broadcast msg": msg,
		}).Info("HERE =>>")
	}
	return nil
}

func (s *Server) handShake(p *Peer) (*Handshake, error) {
	hs := &Handshake{}
	if err := gob.NewDecoder(p.conn).Decode(hs); err != nil {
		return nil, err
	}

	if s.GameVariant != hs.GameVariant {
		return nil, fmt.Errorf("invalid game variant %s", hs.GameVariant.string())
	}
	if s.Version != hs.Version {
		return nil, fmt.Errorf("invalid game version %s", hs.Version)
	}

	p.listenAddr = hs.ListenAddr

	logrus.WithFields(logrus.Fields{
		"Peer":        p.conn.RemoteAddr(),
		"GameVariant": hs.GameVariant.string(),
		"Version":     hs.Version,
		"GameStatus":  hs.GameStatus.String(),
		"ListenAddr":  hs.ListenAddr,
	}).Info("Received handshake from")

	return hs, nil
}

func (s *Server) SendHandshake(p *Peer) error {
	hs := &Handshake{
		GameVariant: s.GameVariant,
		Version:     s.Version,
		GameStatus:  s.GameState.GameStatus,
		ListenAddr:  s.ListenAddr,
	}

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(hs); err != nil {
		return err
	}

	// logrus.WithFields(logrus.Fields{
	// 	"Peer":        p.conn.RemoteAddr(),
	// 	"GameVariant": hs.GameVariant.string(),
	// 	"Version":     hs.Version,
	// 	"GameStatus":  hs.GameStatus.String(),
	// }).Info("Sending handshake to ")

	return p.Send(buf.Bytes())
}

func (s *Server) sendPeerList(p *Peer) error {
	peerList := MessagePeerList{
		Peers: s.GetPeers(),
	}

	if len(peerList.Peers) == 0 {
		return nil
	}

	msg := &Message{
		From:    s.ListenAddr,
		Payload: peerList,
	}
	logrus.WithFields(logrus.Fields{
		"msg":  msg,
		"List": msg.Payload,
	}).Info("Sending peer list ")

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	return p.Send(buf.Bytes())
}

func (s *Server) isInPeerList(addr string) bool {
	for _, p := range s.peers {
		if p.listenAddr == addr {
			return true
		}
	}
	return false
}

func (s *Server) handlePeerList(l MessagePeerList) error {
	for i := range l.Peers {
		logrus.WithFields(logrus.Fields{
			"from": s.ListenAddr,
			"to":   l.Peers[i],
		}).Info("Dialing up connection ")
		if err := s.Connect(l.Peers[i]); err != nil {
			logrus.Error("failed to dial peer", err)
			continue
		}
	}
	return nil
}

func (s *Server) PeerConnectionList() []string {
	result := []string{}
	for _, p := range s.peers {
		result = append(result, p.listenAddr)
	}
	return result
}

func (s *Server) handleNewPeer(peer *Peer) error {

	hs, err := s.handShake(peer)
	if err != nil {
		peer.conn.Close()
		delete(s.peers, peer.listenAddr)
		return fmt.Errorf("error handshaking with peer  %+v  with error: %+v  ", peer.conn.RemoteAddr(), err)
	}
	//TODO checl max player and other logic
	go peer.ReadLoop(s.msgChan)

	if !peer.Outbound {
		if err := s.SendHandshake(peer); err != nil {
			peer.conn.Close()
			delete(s.peers, peer.listenAddr)
			return fmt.Errorf("outbound error sending handshake to peer %+v  with error: %+v ", peer.conn.RemoteAddr(), err)
		}
		if err := s.sendPeerList(peer); err != nil {
			return fmt.Errorf("error sending peer list to  %+v  with error: %+v  ", peer.conn.RemoteAddr(), err)
		}
	}
	s.addPeerWithLock(peer)
	logrus.Info("Handshake Sucessfull->New Player connected: ", peer.conn.RemoteAddr(), " to:", s.ListenAddr)
	s.GameState.AddPlayer(peer.listenAddr, hs.GameStatus)
	return nil
}

func (s *Server) addPeerWithLock(peer *Peer) {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()
	s.peers[peer.listenAddr] = peer
}

func (s *Server) GetPeers() []string {
	s.peerLock.RLock()
	defer s.peerLock.RUnlock()
	peers := []string{}

	for _, p := range s.peers {
		peers = append(peers, p.listenAddr)
	}
	return peers
}

func (s *Server) Broadcast(payload any) error {
	msg := NewMessage(s.ListenAddr, payload)

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range s.peers {
		go func(peer *Peer) {
			if err := peer.Send(buf.Bytes()); err != nil {
				logrus.Errorf("error while broadcasting to peer : [%s] : %+v ", peer.listenAddr, err)
			}
		}(peer)
	}

	return nil
}

func init() {
	gob.Register(MessagePeerList{})
	gob.Register(MessageCards{})
}
