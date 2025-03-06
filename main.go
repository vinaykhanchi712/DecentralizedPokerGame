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
		ListenAddr:  ":3000",
		GameVariant: server.TexasHoldem,
	})
	go s.Start()
	remoteServer := server.NewServer(server.ServerConfig{
		Version:     "VK POKER V0.0.1",
		ListenAddr:  ":4000",
		GameVariant: server.TexasHoldem,
	})

	go remoteServer.Start()
	time.Sleep(500 * time.Millisecond)
	if err := remoteServer.Connect(":3000"); err != nil {
		log.Fatal(err)
	}

	otherServer := server.NewServer(server.ServerConfig{
		Version:     "VK POKER V0.0.1",
		ListenAddr:  ":5000",
		GameVariant: server.TexasHoldem,
	})
	go otherServer.Start()
	time.Sleep(500 * time.Millisecond)
	if err := otherServer.Connect(":4000"); err != nil {
		log.Fatal(err)
	}

	// four := server.NewServer(server.ServerConfig{
	// 	Version:     "VK POKER V0.0.1",
	// 	ListenAddr:  ":6000",
	// 	GameVariant: server.TexasHoldem,
	// })
	// go four.Start()
	// time.Sleep(500 * time.Millisecond)
	// if err := four.Connect(":5000"); err != nil {
	// 	log.Fatal(err)
	// }

	time.Sleep(100 * time.Millisecond)

	logrus.WithFields(logrus.Fields{
		"A": s.PeerConnectionList(),
		"B": remoteServer.PeerConnectionList(),
		"C": otherServer.PeerConnectionList(),
	}).Info("Connections are ")

	select {}
}
