package server

import (
	"bytes"
	"io"
	"net"

	"github.com/sirupsen/logrus"
)

type Message struct {
	From    net.Addr
	Payload io.Reader
}

type Peer struct {
	conn net.Conn
}

func (p *Peer) Send(data []byte) error {
	_, err := p.conn.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (p *Peer) ReadLoop(msgc chan *Message) {
	buf := make([]byte, 1024)
	for {
		n, err := p.conn.Read(buf)
		if err != nil {
			break
		}

		msgc <- &Message{
			From:    p.conn.RemoteAddr(),
			Payload: bytes.NewReader(buf[:n]),
		}

	}
	// TODO : unregister this peer!!!
	p.conn.Close()
}

type TCPTransport struct {
	listenAddr string
	listener   net.Listener
	AddPeer    chan *Peer
	DelPeer    chan *Peer
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
			conn: conn,
		}
		t.AddPeer <- peer
	}

	//return fmt.Errorf("Listener closed")
}
