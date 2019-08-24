package models

import (
	"github.com/jinzhu/gorm"
)

type RoundPlayers struct {
	gorm.Model
	Player int64
	Game   int64
}

// This is the database representation of all cards.
// Each card is associated to a round, deck, and possible a player
type Cards struct {
	gorm.Model
	Cards string
	Game  int64
	Round int64
}
