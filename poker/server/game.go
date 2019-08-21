package server

import (
	"container/ring"
	pb "imran/poker/protobufs"
	"sort"
)

// use a game ring to manage turns
type GameRing struct {
	*ring.Ring
	*pb.Game
}

// NewGameRing generates a ring data type using the players in a game.
// This makes it easy to traverse through players and determine
// who is dealer/
func NewGameRing(g *pb.Game) *GameRing {
	// construct game ring:

	players := g.GetPlayers().GetPlayers()
	r := ring.New(len(players))
	gr := &GameRing{
		Ring: r,
		Game: g,
	}

	sort.Slice(players, func(i, j int) bool {
		return players[i].GetSlot() < players[j].GetSlot()
	})
	for _, p := range players {
		gr.Value = p
		gr.next()
	}

	return gr
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
	err := g.CurrentSmallBlind()
	if err != nil {
		return err
	}
	g.next()
	return nil
}

func (g *GameRing) CurrentSmallBlind() error {
	_, err := g.CurrentDealer()
	if err != nil {
		return err
	}
	g.next()

	return nil
}

func (g *GameRing) next() {
	g.Ring = g.Next()
}

func (g *GameRing) MarshalValue() (*pb.Player, error) {
	player, ok := g.Value.(*pb.Player)

	if !ok {
		return nil, ErrIncorrectRingValueType
	}
	return player, nil
}
