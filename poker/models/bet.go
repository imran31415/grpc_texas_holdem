package models

import (
	"github.com/jinzhu/gorm"
	pb "imran/poker/protobufs"
)

type Bet struct {
	gorm.Model
	Status string
	Round  int64
	Game   int64
	Player int64
	Chips  int64
	Type   string
}

// ProtoUnMarshal gets db representation of the protobuf
func (b *Bet) ProtoUnMarshal(bet *pb.Bet) {
	b.Model.ID = uint(bet.GetId())
	b.Status = bet.GetStatus().String()
	b.Round = bet.GetRound()
	b.Game = bet.GetGame()
	b.Player = bet.GetGame()
	b.Chips = bet.GetChips()
	b.Type = bet.GetType().String()

}

// ProtoMarshal gets the protobuf representation of the DB
func (b *Bet) ProtoMarshal() *pb.Bet {
	return &pb.Bet{
		Id:     int64(b.Model.ID),
		Status: pb.RoundStatus(pb.RoundStatus_value[b.Status]),
		Round:  b.Round,
		Game:   b.Game,
		Player: b.Player,
		Chips:  b.Chips,
		Type:   pb.Bet_BetType(pb.Bet_BetType_value[b.Type]),
	}
}
