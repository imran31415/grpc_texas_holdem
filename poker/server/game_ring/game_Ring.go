package game_ring

import (
	"container/ring"
	"fmt"
	"sort"

	pb "imran/poker/protobufs"
)

var ErrNilRingItems = fmt.Errorf("ring should not have nil items")
var ErrIncorrectRingValueType = fmt.Errorf("unable to marshal value from ring")
var ErrDealerNotSet = fmt.Errorf("dealer not set")

// use a game ring to manage turns
type GameRing struct {
	*ring.Ring
	*pb.Game
}

// NewRing generates a ring data type using the players in a game.
// This makes it easy to traverse through players and determine
// who is dealer, big, small, or who is next in turn
// The database representation of the right consists of the players and their allocated
// game slots, as well as the position of the dealer.
// The player slots are stored in the players table and the dealer position is stored in the game table.
// The Game proto is a serialized version of the game + players, which is the necessary info we need
// to generate a Ring.

// doing a server.GetGame() on a valid game should give the necessary info to generate a ring and start a game.
// the server.ValidateGame() call can help determine if a game has the required info to generate a ring.
func NewRing(g *pb.Game) (*GameRing, error) {
	// construct game ring:

	players := g.GetPlayers().GetPlayers()
	r := ring.New(len(players))
	gr := &GameRing{
		Ring: r,
		Game: g,
	}
	// ensure we allocate players to the rin in correct order
	sort.Slice(players, func(i, j int) bool {
		return players[i].GetSlot() < players[j].GetSlot()
	})
	for _, p := range players {
		gr.Value = p
		gr.next()
	}

	hasNil := false
	// validate all slots are taken
	r.Do(func(p interface{}) {
		if p == nil {
			hasNil = true
		}
	})
	if hasNil {
		return nil, ErrNilRingItems
	}

	return gr, nil
}

func (g *GameRing) CurrentDealer() (*pb.Player, error) {

	for i := 0; i < g.Len(); i++ {

		player, ok := g.Value.(*pb.Player)
		if !ok {
			return nil, ErrIncorrectRingValueType
		}
		if player.GetSlot() == g.GetDealer() {
			return player, nil
		}
		g.next()
	}
	return nil, ErrDealerNotSet

}

func (g *GameRing) CurrentBigBlind() error {
	// heads up means the would be big blind and small blind are
	// switched in a 2 ring arrangement
	if g.headsUp() {
		err := g.CurrentSmallBlind()
		if err != nil {
			return err
		}
		return nil
	}

	err := g.CurrentSmallBlind()
	if err != nil {
		return err
	}
	g.next()
	return nil
}

func (g *GameRing) CurrentSmallBlind() error {
	// heads up means the would be big blind and small blind are
	// switched in a 2 ring arrangement
	if g.headsUp() {
		_, err := g.CurrentDealer()
		if err != nil {
			return err
		}
		return nil
	}
	_, err := g.CurrentDealer()
	if err != nil {
		return err
	}
	g.next()
	return nil
}

// next() is a local convenience method to avoid having to
// manually re-assign ring every time we call an operation on it
func (g *GameRing) next() {
	g.Ring = g.Next()
}

func (g *GameRing) headsUp() bool {
	if g.Len() == 2 {
		return true
	}
	return false
}

func (g *GameRing) MarshalValue() (*pb.Player, error) {
	// TODO: investigate if this perhaps has terrible performance (doesn't seem to)
	player, ok := g.Value.(*pb.Player)

	if !ok {
		return nil, ErrIncorrectRingValueType
	}
	return player, nil
}
