package server

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
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
	}

	s.transport = NewTCPTransport(cnf.ListenAddr)
	s.transport.AddPeer = s.addPeer
	s.transport.DelPeer = s.delPeer
	return s
}

func (s *Server) Start() {

	go s.loop()

	logrus.WithFields(logrus.Fields{
		"Port":    s.ListenAddr,
		"Variant": s.GameVariant.string(),
	}).Info("Starting server on ")
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
			s.SendHandshake(peer)
			if err := s.handShake(peer); err != nil {
				logrus.Error("Error handshaking with peer ", peer.conn.RemoteAddr(), " with error:  ", err)
				continue
			}
			//TODO checl max player and other logic
			go peer.ReadLoop(s.msgChan)
			s.peers[peer.conn.RemoteAddr()] = peer
			logrus.Info("New Player connected: ", peer.conn.RemoteAddr(), " to:", s.ListenAddr)
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
		conn: conn,
	}

	s.addPeer <- peer
	return peer.Send([]byte(s.Version))
}

func (s *Server) handleMessage(msg *Message) error {
	b, err := io.ReadAll(msg.Payload)
	if err != nil {
		return err
	}
	logrus.Info("Message from ", msg.From, " : ", string(b))
	return nil
}

type Handshake struct {
	GameVariant GameVariant
	Version     string
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
	}).Info("Received handshake from")

	return nil
}

func (s *Server) SendHandshake(p *Peer) error {
	hs := &Handshake{
		GameVariant: s.GameVariant,
		Version:     s.Version,
	}

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(hs); err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"Peer":        p.conn.RemoteAddr(),
		"GameVariant": hs.GameVariant.string(),
		"Version":     hs.Version,
	}).Info("Sending handshake to ")

	return p.Send(buf.Bytes())
}
