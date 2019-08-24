package models

import (
	"github.com/jinzhu/gorm"
	"imran/poker/deck"
	pb "imran/poker/protobufs"
)

type Game struct {
	gorm.Model
	Name    string
	Dealer  int64
	Min     int64
	InRound bool
}

// ProtoUnMarshal gets db representation of the protobuf
func (g *Game) ProtoUnMarshal(game *pb.Game) {
	g.Model.ID = uint(game.GetId())
	g.Name = game.GetName()
	g.Dealer = game.GetDealer()
	g.Min = game.GetMin()
	g.InRound = game.GetInRound()
}

// ProtoMarshal gets the protobuf representation of the DB
func (g *Game) ProtoMarshal() *pb.Game {
	return &pb.Game{
		Id:      int64(g.Model.ID),
		Name:    g.Name,
		Dealer:  g.Dealer,
		Min:     g.Min,
		InRound: g.InRound,
	}
}

func (g *Game) MarshalRound() *pb.Round {
	d := deck.New()

	return &pb.Round{
		// Id is nil as it will be reate
		Deck: d.String(),
		Game: int64(g.ID),
	}

}
