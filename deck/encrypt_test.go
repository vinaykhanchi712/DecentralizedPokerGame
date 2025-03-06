package deck

import (
	"fmt"
	"reflect"
	"testing"
)

func TestEncryptCard(t *testing.T) {
	key := []byte("foobar")
	card := Card{
		Suit:  Spades,
		Value: 1,
	}

	response, err := EncryptCard(key, card)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(response)
	decCard, err := DecryptCard(key, response)
	fmt.Println(decCard)

	if !reflect.DeepEqual(card, decCard) {
		t.Errorf("want this [%+v] but got [%+v]", card, decCard)
	}
}
