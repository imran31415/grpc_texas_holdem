package deck_test

import (
	"github.com/stretchr/testify/assert"
	"imran/poker/deck"
	"testing"
)

func TestDeck_Logic(t *testing.T) {

	tests := []struct {
		Name     string
		hand     []string
		ExpScore uint32
	}{
		{
			Name:     "Parses a hand",
			hand:     []string{"Ah", "Ad", "4s", "Ac", "As"},
			ExpScore: 20,
		},
		{
			Name:     "Best hand possible",
			hand:     []string{"As", "Ks", "Qs", "Js", "Ts"},
			ExpScore: 1,
		},
		{
			Name:     "Worst hand possible",
			hand:     []string{"7h", "5d", "4c", "3s", "2h"},
			ExpScore: 7462,
		},
		{
			Name:     "Worst hand possible",
			hand:     []string{"Ts", "9d", "8c", "7c", "6h"},
			ExpScore: 1604,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			score := deck.Logic(tt.hand)
			assert.Equal(t, int(tt.ExpScore), int(score))

		})

	}

}

func TestDeck_Evaluate(t *testing.T) {

	tests := []struct {
		Name     string
		hand     []string
		ExpScore uint32
	}{
		{
			Name:     "Parses a hand",
			hand:     []string{"Ah", "Ad", "4s", "Ac", "As"},
			ExpScore: 20,
		},
		{
			Name:     "Best hand possible",
			hand:     []string{"As", "Ks", "Qs", "Js", "Ts"},
			ExpScore: 1,
		},
		{
			Name:     "Worst hand possible",
			hand:     []string{"7h", "5d", "4c", "3s", "2h"},
			ExpScore: 7462,
		},
		{
			Name:     "another hand",
			hand:     []string{"Ts", "9d", "8c", "7c", "6h"},
			ExpScore: 1604,
		},
		{
			Name:     "another hand with 7 items",
			hand:     []string{"As", "Ks", "Qs", "Js", "Ts", "2d", "3c"},
			ExpScore: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			hand := deck.Hand{}
			for _, c := range tt.hand {
				card := deck.NewCard(c)
				hand = append(hand, *card)
			}
			assert.Equal(t, int(tt.ExpScore), int(hand.EvaluateHand()))

		})

	}

}
