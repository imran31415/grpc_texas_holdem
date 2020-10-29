package models

import (
	"github.com/jinzhu/gorm"
	pb "grpc_texas_holdem/poker/protobufs"
)

type Player struct {
	gorm.Model
	Name   string
	Chips  int64
	Slot   int64
	Cards  string
	InHand bool
}

type GamePlayers struct {
	gorm.Model
	Player int64
	Game   int64
}

func (p *Player) ProtoUnMarshal(player *pb.Player) {
	p.Model.ID = uint(player.GetId())
	p.Name = player.GetName()
	p.Chips = player.GetChips()
	p.Slot = player.GetSlot()
	p.InHand = player.GetInHand()
	p.Cards = player.GetCards()

}

// ProtoMarshal gets the protobuf representation of the DB
func (p *Player) ProtoMarshal() *pb.Player {
	return &pb.Player{
		Id:     int64(p.Model.ID),
		Name:   p.Name,
		Chips:  p.Chips,
		Slot:   p.Slot,
		InHand: p.InHand,
		Cards:  p.Cards,
	}
}

func MarshalPlayers(outs []*Player) *pb.Players {

	out := &pb.Players{}
	for _, inp := range outs {
		out.Players = append(out.Players, inp.ProtoMarshal())
	}
	return out
}
