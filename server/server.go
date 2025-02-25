package server

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"

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
	peers        map[net.Addr]*Peer
	addPeer      chan *Peer
	msgChan      chan *Message
	delPeer      chan *Peer
	GameState    *GameState
}

func NewServer(cnf ServerConfig) *Server {
	s := &Server{
		ServerConfig: cnf,
		peers:        make(map[net.Addr]*Peer),
		addPeer:      make(chan *Peer),
		msgChan:      make(chan *Message),
		delPeer:      make(chan *Peer),
		GameState:    NewGameState(),
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
			addr := peer.conn.RemoteAddr()
			delete(s.peers, addr)
			logrus.Info("Player disconnected: ", addr)
		case peer := <-s.addPeer:

			if err := s.handShake(peer); err != nil {
				logrus.Error("Error handshaking with peer ", peer.conn.RemoteAddr(), " with error:  ", err)
				continue
			}
			//TODO checl max player and other logic

			go peer.ReadLoop(s.msgChan)

			if !peer.Outbound {
				if err := s.SendHandshake(peer); err != nil {
					logrus.Error("outbound error sending handshake to peer ", peer.conn.RemoteAddr(), " with error:  ", err)
					peer.conn.Close()
					delete(s.peers, peer.conn.RemoteAddr())
					continue
				}
			}
			s.peers[peer.conn.RemoteAddr()] = peer
			logrus.Info("Handshake Sucessfull->New Player connected: ", peer.conn.RemoteAddr(), " to:", s.ListenAddr)

			if err := s.sendPeerList(peer); err != nil {
				logrus.Error("Error sending peer list to ", peer.conn.RemoteAddr(), " with error:  ", err)
				continue
			}

		case msg := <-s.msgChan:
			if err := s.handleMessage(msg); err != nil {
				logrus.Error("Error handling message", err)
			}
		}
	}
}

func (s *Server) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
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
	}
	return nil
}

func (s *Server) handShake(p *Peer) error {
	hs := &Handshake{}
	if err := gob.NewDecoder(p.conn).Decode(hs); err != nil {
		return err
	}

	if s.GameVariant != hs.GameVariant {
		return fmt.Errorf("invalid game variant %s", hs.GameVariant.string())
	}
	if s.Version != hs.Version {
		return fmt.Errorf("invalid game version %s", hs.Version)
	}

	logrus.WithFields(logrus.Fields{
		"Peer":        p.conn.RemoteAddr(),
		"GameVariant": hs.GameVariant.string(),
		"Version":     hs.Version,
		"GameStatus":  hs.GameStatus.String(),
	}).Info("Received handshake from")

	return nil
}

func (s *Server) SendHandshake(p *Peer) error {
	hs := &Handshake{
		GameVariant: s.GameVariant,
		Version:     s.Version,
		GameStatus:  s.GameState.GameStatus,
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
		Peers: make([]string, len(s.peers)),
	}
	itr := 0
	for address := range s.peers {
		peerList.Peers[itr] = address.String()
		itr++
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

func (s *Server) handlePeerList(l MessagePeerList) error {
	for i := 0; i < len(l.Peers); i++ {
		if err := s.Connect(l.Peers[i]); err != nil {
			logrus.Error("failed to dial peer", err)
			continue
		}
	}
	return nil
}

func init() {
	gob.Register(MessagePeerList{})
}
