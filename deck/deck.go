package deck

import (
	"fmt"
	"math/rand"

	"github.com/sirupsen/logrus"
)

type Suit int

const (
	Spades Suit = iota
	Hearts
	Diamonds
	Clubs
)

func (s Suit) String() string {
	switch s {
	case Spades:
		return "SPADES"
	case Hearts:
		return "Hearts"
	case Diamonds:
		return "DIAMONDS"
	case Clubs:
		return "CLUBS"
	default:
		return ("Invalid card type")
	}
}

func (s Suit) Unicode() string {
	switch s {
	case Spades:
		return "♠"
	case Hearts:
		return "♥"
	case Diamonds:
		return "♦"
	case Clubs:
		return "♣"
	default:
		return ("Invalid card type")
	}
}

type Card struct {
	Suit  Suit
	Value int
}

func (c Card) String() string {
	return fmt.Sprintf("%d of %s %s", c.Value, c.Suit.String(), c.Suit.Unicode())
}

func NewCard(s Suit, v int) Card {
	if v > 13 {
		logrus.Fatalln("Invalid card value", v)
	}
	return Card{
		Suit:  s,
		Value: v,
	}
}

type Deck [52]Card

func NewDeck() Deck {
	var deck Deck
	for i := 0; i < 13; i++ {
		deck[i] = NewCard(Spades, i+1)
		deck[i+13] = NewCard(Hearts, i+1)
		deck[i+26] = NewCard(Diamonds, i+1)
		deck[i+39] = NewCard(Clubs, i+1)
	}
	return deck
}

func (d *Deck) Shuffle() {
	for idx := range d {
		newPosition := idx
		for newPosition == idx {
			newPosition = rand.Intn(52)
		}
		d[idx], d[newPosition] = d[newPosition], d[idx]
	}
}

func (d Deck) String() string {
	var s string
	for _, c := range d {
		s += c.String() + "\n"
	}
	return s
}
