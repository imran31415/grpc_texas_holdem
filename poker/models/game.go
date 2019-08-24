package models

import (
	"github.com/jinzhu/gorm"
	pb "imran/poker/protobufs"
)

type Game struct {
	gorm.Model
	Name   string
	Dealer int64
	Min    int64
}

// ProtoUnMarshal gets db representation of the protobuf
func (g *Game) ProtoUnMarshal(game *pb.Game) {
	g.Model.ID = uint(game.GetId())
	g.Name = game.GetName()
	g.Dealer = game.GetDealer()
	g.Min = game.GetMin()
}

// ProtoMarshal gets the protobuf representation of the DB
func (g *Game) ProtoMarshal() *pb.Game {
	return &pb.Game{
		Id:     int64(g.Model.ID),
		Name:   g.Name,
		Dealer: g.Dealer,
		Min:    g.Min,
	}
}
