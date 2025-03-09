package main

import (
	"DPokerGame/p2p"
	"log"
	"net/http"
	"time"
)

func main() {

	first := makeServerAndStart(":3000", ":3001")

	second := makeServerAndStart(":4000", ":40001")

	three := makeServerAndStart(":5000", ":5001")

	four := makeServerAndStart(":6000", ":6001")

	time.Sleep(500 * time.Millisecond)

	if err := second.Connect(first.ListenAddr); err != nil {
		log.Fatal(err)
	}

	if err := three.Connect(second.ListenAddr); err != nil {
		log.Fatal(err)
	}

	if err := four.Connect(three.ListenAddr); err != nil {
		log.Fatal(err)
	}

	go func() {
		time.Sleep(2 * time.Second)
		http.Get("http://localhost" + first.ApiListenAddr + "/ready/1")
		http.Get("http://localhost" + second.ApiListenAddr + "/ready/2")
		http.Get("http://localhost" + three.ApiListenAddr + "/ready/3")
		http.Get("http://localhost" + four.ApiListenAddr + "/ready/4")
	}()

	select {}
}

func makeServerAndStart(addr, apiAddr string) *p2p.Server {
	s := p2p.NewServer(p2p.ServerConfig{
		Version:       "VK POKER V0.0.1",
		ListenAddr:    addr,
		GameVariant:   p2p.TexasHoldem,
		ApiListenAddr: apiAddr,
	})
	go s.Start()
	return s
}
