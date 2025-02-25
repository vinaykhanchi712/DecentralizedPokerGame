package main

import (
	"DPokerGame/deck"
	"DPokerGame/server"
	"fmt"
	"log"
	"time"
)

func main() {

	deck := deck.NewDeck()
	deck.Shuffle()
	fmt.Println(deck)
	s := server.NewServer(server.ServerConfig{
		Version:     "VK POKER V0.0.1",
		ListenAddr:  ":8081",
		GameVariant: server.TexasHoldem,
	})
	go s.Start()
	remoteServer := server.NewServer(server.ServerConfig{
		Version:     "VK POKER V0.0.1",
		ListenAddr:  ":8082",
		GameVariant: server.TexasHoldem,
	})

	go remoteServer.Start()
	time.Sleep(1 * time.Second)
	if err := remoteServer.Connect(":8081"); err != nil {
		log.Fatal(err)
	}

	otherServer := server.NewServer(server.ServerConfig{
		Version:     "VK POKER V0.0.1",
		ListenAddr:  ":8083",
		GameVariant: server.TexasHoldem,
	})
	go otherServer.Start()
	time.Sleep(1 * time.Second)
	if err := otherServer.Connect(":8082"); err != nil {
		log.Fatal(err)
	}

	select {}
}
