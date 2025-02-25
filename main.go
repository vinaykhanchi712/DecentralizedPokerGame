package main

import (
	"DPokerGame/deck"
	"DPokerGame/server"
	"fmt"
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
	remoteServer.Connect(":8081")
	select {}
}
