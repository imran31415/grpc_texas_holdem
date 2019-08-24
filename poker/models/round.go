package models

import (
	"github.com/jinzhu/gorm"
	pb "imran/poker/protobufs"
)

// db representation of a round of 1 sequence of dealing out cards
type Round struct {
	gorm.Model
	Deck   string
	Status string
	Flop   string
	Turn   string
	River  string
	Game   int64
}

type RoundPlayers struct {
	gorm.Model
	Round int64
	Player int64
	Game   int64
}

func (r *Round) ProtoUnMarshal(round *pb.Round) {
	r.Model.ID = uint(round.GetId())
	r.Deck = round.GetDeck()
	r.Flop = round.GetFlop()
	r.Turn = round.GetTurn()
	r.River = round.GetRiver()
	r.Game = round.GetGame()
	r.Status = round.GetStatus().String()
}

// ProtoMarshal gets the protobuf representation of the DB
func (p *Round) ProtoMarshal() *pb.Round {
	return &pb.Round{
		Id:     int64(p.Model.ID),
		Deck:   p.Deck,
		Flop:   p.Flop,
		Turn:   p.Turn,
		River:  p.River,
		Game:   p.Game,
		Status: pb.Round_Status(pb.Round_Status_value[p.Status]),
	}
}
