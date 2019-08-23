package deck_test

import (
	"github.com/stretchr/testify/assert"
	"imran/poker/deck"
	"testing"
)

func TestServer_CreatePlayer(t *testing.T) {

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
			hand:     []string{"Ts", "9d", "8c", "7c", "6hh"},
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
