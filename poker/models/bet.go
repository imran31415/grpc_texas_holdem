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
		Status: pb.Bet_Status(pb.Bet_Status_value[b.Status]),
		Round:  b.Round,
		Game:   b.Game,
		Player: b.Player,
		Chips:  b.Chips,
		Type:   pb.Bet_BetType(pb.Bet_BetType_value[b.Type]),
	}
}

//message Bet{
//enum Status {
//NOT_STARTED = 0; // No betting
//PRE_FLOP = 1;    // first round of betting is happening
//FLOP = 2;        // second round of betting
//RIVER = 3;       // third round of betting
//TURN = 4;        // final round of betting
//SHOW = 5;        // All bets are closed and we show any hands remaining
//OVER = 6;        // Winner has been determined and chips have been disbursed
//}
//Status status = 1;
//int64 round = 2;
//int64 game = 3;
//int64 player = 4;
//int64 chips = 5;
//enum BetType {
//NONE = 0;
//FOLD = 1;
//CALL = 2;
//RAISE = 3;
//}
//BetType type = 6;
//}
