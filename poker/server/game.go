package server

import (
	"container/ring"
	pb "imran/poker/protobufs"
)

type GameRing struct {
	*ring.Ring
	*pb.Game
}

func NewGameRing(g *pb.Game) *GameRing {
	// construct game ring:
	r := ring.New(len(g.GetPlayers().GetPlayers()))
	for _, p := range g.GetPlayers().GetPlayers() {
		r.Value = p
		r = r.Next()
	}

	gr := &GameRing{
		Ring: r,
		Game: g,
	}
	return gr
}

func (g *GameRing) CurrentDealer() (*pb.Player, error) {
	g.Move(int(g.GetDealer()))
	player, ok := g.Value.(*pb.Player)

	if !ok {
		return nil, ErrIncorrectRingValueType
	}
	return player, nil

}

func (g *GameRing) CurrentBigBlind() (*pb.Player, error) {
	g.Move(int(g.GetDealer()))
	g.Next()
	player, ok := g.Value.(*pb.Player)

	if !ok {
		return nil, ErrIncorrectRingValueType
	}
	return player, nil
}

func (g *GameRing) CurrentSmallBlind() (*pb.Player, error) {
	g.Move(int(g.GetDealer()))
	g.Next()
	g.Next()
	player, ok := g.Value.(*pb.Player)

	if !ok {
		return nil, ErrIncorrectRingValueType
	}
	return player, nil
}
