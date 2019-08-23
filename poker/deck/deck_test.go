package deck_test

import (
	"github.com/stretchr/testify/assert"
	"imran/poker/deck"
	"testing"
)

// Verify we can create a Deck,
// serialize it to a string, and finally
// re-injest it back to a Deck object

func TestDeck_Serialize(t *testing.T) {

	tests := []struct {
		Name     string
		hand     []string
		ExpScore uint32
	}{
		{
			Name: "Creates a deck",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			d := deck.New()
			// Upon creation the deck is not shuffled
			assert.Equal(t, 52, len(d))
			assert.Equal(t, "h", d[0].Suit)
			assert.Equal(t, "2", d[0].Type)
			assert.Equal(t, "d", d[1].Suit)
			assert.Equal(t, "2", d[1].Type)
			assert.Equal(t, "c", d[2].Suit)
			assert.Equal(t, "2", d[2].Type)
			assert.Equal(t, "s", d[3].Suit)
			assert.Equal(t, "2", d[3].Type)
			assert.Equal(t, "h", d[4].Suit)
			assert.Equal(t, "3", d[4].Type)

			// serialize the deck into a string

			deckstring := d.String()
			assert.Equal(t, 104, len(deckstring))
			assert.Equal(t, "2h2d2c2s3h3d3c3s4h4d4c4s5h5d5c5s6h6d6c6s7h7d7c7s8h8d8c8s9h9d9c9sThTdTcTsJhJdJcJsQhQdQcQsKhKdKcKsAhAdAcAs", deckstring)

			d = deck.Deck{}
			assert.Equal(t, 0, len(d))
			d = d.Marshal(deckstring)

			assert.Equal(t, 52, len(d))
			assert.Equal(t, 52, len(d))
			assert.Equal(t, "h", d[0].Suit)
			assert.Equal(t, "2", d[0].Type)
			assert.Equal(t, "d", d[1].Suit)
			assert.Equal(t, "2", d[1].Type)
			assert.Equal(t, "c", d[2].Suit)
			assert.Equal(t, "2", d[2].Type)
			assert.Equal(t, "s", d[3].Suit)
			assert.Equal(t, "2", d[3].Type)
			assert.Equal(t, "h", d[4].Suit)
			assert.Equal(t, "3", d[4].Type)

			// verify the last card in a sorted deck
			assert.Equal(t, "s", d[51].Suit)
			assert.Equal(t, "A", d[51].Type)

			// Deals from the bottom of the deck
			c, d := deck.DealCard(d)
			// 1 less card since we dealt a card
			assert.Equal(t, 51, len(d))
			// Dealt card should be last card
			assert.Equal(t, deck.Card{Type: "A", Suit: "s"}, c)

		})

	}

}
