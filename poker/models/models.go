package models

import (
	"github.com/jinzhu/gorm"
	pb "imran/poker/protobufs"
)

type Player struct {
	gorm.Model
	Name  string
	Chips int64
	Slot  int64
}

type GamePlayers struct {
	gorm.Model
	Player int64
	Game   int64
}

type RoundPlayers struct {
	gorm.Model
	Player int64
	Game   int64
}

type Game struct {
	gorm.Model
	Name   string
	Dealer int64
	Min    int64
}

// ProtoUnMarshal gets db representation of the protobuf
func (p *Player) ProtoUnMarshal(player *pb.Player) {
	p.Model.ID = uint(player.GetId())
	p.Name = player.GetName()
	p.Chips = player.GetChips()
	p.Slot = player.GetChips()
}

// ProtoUnMarshal gets db representation of the protobuf
func (g *Game) ProtoUnMarshal(game *pb.Game) {
	g.Model.ID = uint(game.GetId())
	g.Name = game.GetName()
	g.Dealer = game.GetDealer()
	g.Min = game.GetMin()
}

// ProtoMarshal gets the protobuf representation of the DB
func (p *Player) ProtoMarshal() *pb.Player {
	return &pb.Player{
		Id:    int64(p.Model.ID),
		Name:  p.Name,
		Chips: p.Chips,
		Slot:  p.Slot,
	}
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

func MarshalPlayers(outs []*Player) *pb.Players {

	out := &pb.Players{}
	for _, inp := range outs {
		out.Players = append(out.Players, inp.ProtoMarshal())
	}
	return out
}
