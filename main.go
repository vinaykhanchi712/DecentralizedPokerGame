package main

import (
	"DPokerGame/deck"
	"DPokerGame/server"
	"fmt"
	"log"
	"time"

	"github.com/sirupsen/logrus"
)

func main() {

	deck := deck.NewDeck()
	deck.Shuffle()
	fmt.Println(deck)
	s := server.NewServer(server.ServerConfig{
		Version:     "VK POKER V0.0.1",
		ListenAddr:  "127.0.0.1:3000",
		GameVariant: server.TexasHoldem,
	})
	go s.Start()
	remoteServer := server.NewServer(server.ServerConfig{
		Version:     "VK POKER V0.0.1",
		ListenAddr:  "127.0.0.1:4000",
		GameVariant: server.TexasHoldem,
	})

	go remoteServer.Start()
	time.Sleep(1 * time.Second)
	if err := remoteServer.Connect("127.0.0.1:3000"); err != nil {
		log.Fatal(err)
	}

	otherServer := server.NewServer(server.ServerConfig{
		Version:     "VK POKER V0.0.1",
		ListenAddr:  "127.0.0.1:5000",
		GameVariant: server.TexasHoldem,
	})
	go otherServer.Start()
	time.Sleep(2 * time.Second)
	if err := otherServer.Connect("127.0.0.1:4000"); err != nil {
		log.Fatal(err)
	}

	four := server.NewServer(server.ServerConfig{
		Version:     "VK POKER V0.0.1",
		ListenAddr:  "127.0.0.1:6000",
		GameVariant: server.TexasHoldem,
	})
	go four.Start()
	time.Sleep(2 * time.Second)
	if err := four.Connect("127.0.0.1:5000"); err != nil {
		log.Fatal(err)
	}

	time.Sleep(5 * time.Second)

	logrus.WithFields(logrus.Fields{
		"A": s.PeerConnectionList(),
		"B": remoteServer.PeerConnectionList(),
		"C": otherServer.PeerConnectionList(),
		"D": four.PeerConnectionList(),
	}).Info("Connections are ")

	select {}
}
