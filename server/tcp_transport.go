package server

import (
	"encoding/gob"
	"net"

	"github.com/sirupsen/logrus"
)

type Peer struct {
	conn     net.Conn
	Outbound bool
}

func (p *Peer) Send(data []byte) error {
	_, err := p.conn.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (p *Peer) ReadLoop(msgc chan *Message) {

	for {

		msg := &Message{}
		if err := gob.NewDecoder(p.conn).Decode(msg); err != nil {
			logrus.Error("[tcp_transport]Error decoding message", err)
			break
		}

		msgc <- msg

	}
	// TODO : unregister this peer!!!
	p.conn.Close()
}

type TCPTransport struct {
	listenAddr string
	listener   net.Listener
	addPeer    chan *Peer
	delPeer    chan *Peer
}

func NewTCPTransport(addr string) *TCPTransport {
	return &TCPTransport{
		listenAddr: addr,
	}
}

func (t *TCPTransport) ListenAndAccept() error {
	ls, err := net.Listen("tcp", t.listenAddr)
	if err != nil {
		logrus.Error("Error listening", err)
		return err
	}
	t.listener = ls

	for {
		conn, err := t.listener.Accept()
		if err != nil {
			logrus.Error("Error accepting connection", err)
			continue
		}
		peer := &Peer{
			conn:     conn,
			Outbound: false,
		}
		t.addPeer <- peer
	}

	//return fmt.Errorf("Listener closed")
}
