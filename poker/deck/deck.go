package deck

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// shamelessly stolen from https://gist.github.com/montanaflynn/4cc2779d2e353d7524a7bdce57869a75
// 		-With some slight modifications to make it work with the evaluation logic

// Seed our randomness with the current time
func init() {
	rand.Seed(time.Now().UnixNano())
}

// Card holds the card suits and types in the deck
type Card struct {
	Type string
	Suit string
}

func (c *Card) String() string {
	return fmt.Sprintf("%s%s", c.Type, c.Suit)
}

func (c *Card) Marshal(string) {
	c = &Card{}
}

type Deck []Card

func (d Deck) String() string {
	b := strings.Builder{}
	for _, i := range d {
		b.WriteString(i.String())
	}
	return b.String()
}

func (d Deck) Marshal(deck string) Deck {
	out := []Card{}
	for i, _ := range deck {
		index := i + 1
		if index%2 == 0 {
			out = append(out, Card{
				Type: string(deck[i-1]),
				Suit: string(deck[i]),
			})

		}
	}
	return out

}

func New() (deck Deck) {
	types := []string{"2", "3", "4", "5", "6", "7",
		"8", "9", "T", "J", "Q", "K", "A"}

	// Valid suits include Heart, Diamond, Club & Spade
	suits := []string{"h", "d", "c", "s"}

	// Loop over each type and suit appending to the deck
	for i := 0; i < len(types); i++ {
		for n := 0; n < len(suits); n++ {
			card := Card{
				Type: types[i],
				Suit: suits[n],
			}
			deck = append(deck, card)
		}
	}
	return
}

func (d Deck) IsFull() bool {
	// 104 is a full deck (2*52)
	if len(d) == 52 {
		return true
	}
	return false

}

// Shuffle the deck
func Shuffle(d Deck) Deck {
	for i := 1; i < len(d); i++ {
		// Create a random int up to the number of cards
		r := rand.Intn(i + 1)

		// If the the current card doesn't match the random
		// int we generated then we'll switch them out
		if i != r {
			d[r], d[i] = d[i], d[r]
		}
	}
	return d
}

// Deal a specified amount of cards
func DealCard(d Deck) (Card, Deck) {
	c, d := d[len(d)-1], d[:len(d)-1]
	return c, d
}

func EvaluateHand(cards []Card) uint32 {
	toEval := []string{}
	for _, c := range cards {
		toEval = append(toEval, c.String())
	}
	return Logic(toEval)

}
