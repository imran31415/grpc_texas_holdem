package server

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sort"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"google.golang.org/grpc"
	"grpc_texas_holdem/poker/deck"
	"grpc_texas_holdem/poker/models"
	pb "grpc_texas_holdem/poker/protobufs"
	"grpc_texas_holdem/poker/server/game_ring"
)

const (
	Port   = ":50051"
	dbName = "poker"
)

var (
	ErrPlayerNameExists        = fmt.Errorf("player with that name already exists")
	ErrEmptyPlayerName         = fmt.Errorf("can not create player with empty name")
	ErrInvalidPlayerCount      = fmt.Errorf("can not create game with supplied count of players")
	ErrGameNameExists          = fmt.Errorf("game with that name already exists")
	ErrEmptyGameName           = fmt.Errorf("can not create game with empty name")
	ErrInvalidSlotNumber       = fmt.Errorf("slot value invalid must be between 1-8")
	ErrInvalidSlotMinMax       = fmt.Errorf("slot value is greater than 8 or less than 1")
	ErrGameDoesntExist         = fmt.Errorf("no game found")
	ErrInvalidButtonAllocation = fmt.Errorf("buttons are not allocated correctly")
	ErrNoBetSet                = fmt.Errorf("no bet set for game")
	ErrPlayerDoesntExist       = fmt.Errorf("player doesn't exist")
	ErrGameInRound             = fmt.Errorf("can not perform operation when game is in round")
	ErrDeckNotFull             = fmt.Errorf("deck is not full")
	ErrExistingCards           = fmt.Errorf("player already has cards")
	ErrInsufficientChips       = fmt.Errorf("player doesn't have enough chips")
	ErrPlayerNotOnAction       = fmt.Errorf("player not on action, can not bet")
	ErrNoBetsAllowed           = fmt.Errorf("no bets allowed for round status")
	ErrInsufficientBet         = fmt.Errorf("insufficient bet for game minimum")
	ErrGameIsNotInRound        = fmt.Errorf("game is not in round")
	ErrWrongBetType            = fmt.Errorf("wrong bet type")
	ErrNoBetTypeSet            = fmt.Errorf("no bet type set")
	ErrUnImplementedLogic      = fmt.Errorf("this logic is unimplemented")
	ErrIncorrectBetForBetType  = fmt.Errorf("incorrect amount of chips for the given bet")
	ErrPlayerNotInHand         = fmt.Errorf("the given player is not in hand and can not perform that action")
	ErrIncompleteBets          = fmt.Errorf("wrong number of bets for round")
	ErrWrongBetStatus          = fmt.Errorf("wrong bet status")
	ErrNoExistingCards         = fmt.Errorf("expecting existing cards, but no cards for player in hand")
	ErrNoWinningPlayer         = fmt.Errorf("no winning player determined")
)

// TODOS:
// Add test for exiting early when everyone folds.

type Server struct {
	gormDb *gorm.DB
}

func NewServer(name string) (*Server, error) {
	s := &Server{}
	err := s.setupDatabase(name)
	return s, err
}

func Run() {
	lis, err := net.Listen("tcp", Port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	serv, err := NewServer(dbName)
	if err != nil {
		log.Fatalf("failed to Start poker server: %v", err)
	}
	pb.RegisterPokerServer(s, serv)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (s *Server) setupDatabase(name string) error {

	db, err := gorm.Open("sqlite3", fmt.Sprintf("./%s.db", name))
	if err != nil {
		return err
	}
	// Setup Players table
	if err := db.AutoMigrate(&models.Player{}).Error; err != nil {
		return err
	}

	if err := db.AutoMigrate(&models.GamePlayers{}).Error; err != nil {
		return err
	}
	if err := db.AutoMigrate(&models.Game{}).Error; err != nil {
		return err
	}
	if err := db.AutoMigrate(&models.Round{}).Error; err != nil {
		return err
	}
	if err := db.AutoMigrate(&models.RoundPlayers{}).Error; err != nil {
		return err
	}

	if err := db.AutoMigrate(&models.Bet{}).Error; err != nil {
		return err
	}

	s.gormDb = db
	return nil
}

func (s *Server) CreatePlayer(ctx context.Context, p *pb.Player) (*pb.Player, error) {
	if p.GetName() == "" {
		return nil, ErrEmptyPlayerName
	}

	exists, err := s.GetPlayersByName(ctx, &pb.Players{Players: []*pb.Player{p}})

	if err != nil {
		return nil, err
	}
	if len(exists.GetPlayers()) > 0 {
		if exists.GetPlayers()[0].GetId() != 0 {
			return nil, ErrPlayerNameExists
		}
	}

	toCreate := &models.Player{}
	toCreate.ProtoUnMarshal(p)

	if err := s.gormDb.Create(toCreate).Error; err != nil {
		return nil, err
	}

	player, err := s.GetPlayer(ctx, toCreate.ProtoMarshal())
	if err != nil {
		return nil, err
	}
	return player, nil

}

func (s *Server) CreatePlayers(ctx context.Context, players *pb.Players) (*pb.Players, error) {
	out := &pb.Players{}
	for _, p := range players.Players {
		player, err := s.CreatePlayer(ctx, p)
		if err != nil {
			return nil, err
		}
		out.Players = append(out.Players, player)
	}
	return out, nil

}

func (s *Server) GetPlayer(ctx context.Context, in *pb.Player) (*pb.Player, error) {
	p := &models.Player{}
	if err := s.gormDb.Where("id = ?", uint(in.GetId())).First(&p).Error; err != nil {
		return nil, err
	}
	return p.ProtoMarshal(), nil
}

func (s *Server) GetPlayers(ctx context.Context, players *pb.Players) (*pb.Players, error) {
	outs := []*models.Player{}
	ids := []int64{}

	for _, n := range players.GetPlayers() {
		ids = append(ids, n.GetId())
	}
	s.gormDb.Where("id IN (?)", ids).Find(&outs)

	return models.MarshalPlayers(outs), nil
}

func (s *Server) GetPlayersByName(ctx context.Context, players *pb.Players) (*pb.Players, error) {

	outs := []*models.Player{}
	names := []string{}

	for _, n := range players.GetPlayers() {
		names = append(names, n.GetName())
	}
	s.gormDb.Where("name IN (?)", names).Find(&outs)

	return models.MarshalPlayers(outs), nil

}

func (s *Server) GetGame(ctx context.Context, in *pb.Game) (*pb.Game, error) {

	g := &models.Game{}

	if err := s.gormDb.Where("id = (?)", in.GetId()).Find(g).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	} else if err != nil && err == gorm.ErrRecordNotFound {
		return nil, ErrGameDoesntExist
	}

	// Hydrate players
	playersId, err := s.GetGamePlayersByGameId(ctx, &pb.Game{Id: int64(g.ID)})

	if err != nil {
		return nil, err
	}

	players, err := s.GetPlayers(ctx, playersId)
	if err != nil {
		return nil, err
	}

	game := g.ProtoMarshal()
	game.Players = players

	return game, nil
}

func (s *Server) GetRound(ctx context.Context, in *pb.Round) (*pb.Round, error) {

	r := &models.Round{}

	if err := s.gormDb.Where("id = (?)", in.GetId()).Find(r).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	} else if err != nil && err == gorm.ErrRecordNotFound {
		return nil, ErrGameDoesntExist
	}

	// Hydrate players
	playersId, err := s.GetRoundPlayersByRoundId(ctx, &pb.Round{Id: int64(r.ID)})

	if err != nil {
		return nil, err
	}

	players, err := s.GetPlayers(ctx, playersId)
	if err != nil {
		return nil, err
	}

	round := r.ProtoMarshal()
	round.Players = players

	return round, nil
}

func (s *Server) DeleteGames(ctx context.Context, toDelete *pb.Games) (*empty.Empty, error) {

	var ids = []int64{}
	for _, game := range toDelete.GetGames() {
		ids = append(ids, game.GetId())
	}

	if err := s.gormDb.Where("id in (?)", ids).Find(&models.Game{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return &empty.Empty{}, err
	} else if err != nil && err == gorm.ErrRecordNotFound {
		return &empty.Empty{}, ErrGameDoesntExist
	}

	if err := s.gormDb.Where("id in (?)", ids).Delete(&models.Game{}).Error; err != nil {
		return &empty.Empty{}, err
	}
	return &empty.Empty{}, nil
}

func (s *Server) DeletePlayers(ctx context.Context, toDelete *pb.Players) (*empty.Empty, error) {

	var ids = []int64{}
	for _, player := range toDelete.GetPlayers() {
		ids = append(ids, player.GetId())
	}

	if err := s.gormDb.Where("id in (?)", ids).Find(&models.Player{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return &empty.Empty{}, err
	} else if err != nil && err == gorm.ErrRecordNotFound {
		return &empty.Empty{}, ErrPlayerDoesntExist
	}

	if err := s.gormDb.Where("id in (?)", ids).Delete(&models.Player{}).Error; err != nil {
		return &empty.Empty{}, err
	}

	return &empty.Empty{}, nil
}

func (s *Server) GetGameByName(ctx context.Context, in *pb.Game) (*pb.Game, error) {
	g := &models.Game{}
	if err := s.gormDb.Where("name = ?", in.GetName()).Find(&g).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	} else if err != nil && err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return g.ProtoMarshal(), nil
}

func (s *Server) GetGamePlayersByGameId(ctx context.Context, in *pb.Game) (*pb.Players, error) {
	gp := []*models.GamePlayers{}

	if err := s.gormDb.Where("game = ?", in.GetId()).Find(&gp).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	} else if err != nil && err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	out := &pb.Players{}
	for _, pId := range gp {
		out.Players = append(out.Players, &pb.Player{
			Id: int64(pId.Player),
		})
	}
	// Get hydrated player instead of just their IDs
	players, err := s.GetPlayers(ctx, out)
	if err != nil {
		return nil, err
	}

	return players, nil
}

// SetGamePlayers Sets the game players.
// This method is flexible so if there are existing players in the game
// it will only add the difference (If the total number of players is less than 9 and greater than 1)
// This is not an indepodent operation so existing players are considered and only the difference is added
func (s *Server) SetGamePlayers(ctx context.Context, g *pb.Game) (*pb.Players, error) {

	// 1. Get existing players IDs in the game
	existingIds, err := s.GetGamePlayersByGameId(ctx, g)

	if err != nil {
		return nil, err
	}

	//2. Get Existing Player Records from the IDs
	existingPlayerRecords, err := s.GetPlayers(ctx, existingIds)
	if err != nil {
		return nil, err
	}
	// 2.a create a map of existing playerIds to the player record
	existingPlayersMap := map[int64]*pb.Player{}
	for _, p := range existingPlayerRecords.GetPlayers() {
		existingPlayersMap[p.GetId()] = p
	}

	//3. Get the players requesting to be added to the game
	playersToJoinRecords, err := s.GetPlayersByName(ctx, g.GetPlayers())
	if err != nil {
		return nil, err
	}

	// 3.a create a map of requesting playerIds to boolean of if they should join
	playersToJoinMap := map[int64]*pb.Player{}
	for _, p := range playersToJoinRecords.GetPlayers() {
		// Player is not already on the game list
		if _, ok := existingPlayersMap[p.GetId()]; !ok {
			playersToJoinMap[p.GetId()] = p
		}
	}

	for _, shouldAdd := range playersToJoinMap {
		toCreate := &models.GamePlayers{Player: shouldAdd.GetId(), Game: g.GetId()}
		if err := s.gormDb.Create(toCreate).Error; err != nil {
			return nil, err
		}

	}
	players, err := s.GetGamePlayersByGameId(ctx, g)
	if err != nil {
		return nil, err
	}
	return players, err
}

func (s *Server) SetPlayerSlot(ctx context.Context, p *pb.Player) (*pb.Player, error) {

	if p.GetSlot() > 8 || p.GetSlot() < 1 {
		return nil, ErrInvalidSlotMinMax
	}
	out := &models.Player{}

	if err := s.gormDb.Where("id = ?", p.GetId()).Find(out).Update(
		"slot", p.GetSlot()).Error; err != nil {
		return nil, err
	}

	player, err := s.GetPlayer(ctx, p)
	if err != nil {
		return nil, err
	}
	return player, nil

}

func (s *Server) AllocateGameSlots(ctx context.Context, g *pb.Game) (*pb.Game, error) {

	players := g.GetPlayers().GetPlayers()
	if len(players) < 2 || len(players) > 8 {
		return nil, ErrInvalidPlayerCount
	}
	for i, p := range players {

		// start at 1 because 0 is the nil value of a slot so 0 signifies unassigned
		slot := i + 1
		p.Slot = int64(slot)
	}

	for _, p := range players {
		_, err := s.SetPlayerSlot(ctx, p)
		if err != nil {
			return nil, err
		}
	}

	out, err := s.GetGame(ctx, g)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Server) CreateGame(ctx context.Context, g *pb.Game) (*pb.Game, error) {
	if g.GetName() == "" {
		return nil, ErrEmptyGameName
	}

	exists, err := s.GetGameByName(ctx, g)
	if err != nil {
		return nil, err
	}
	if exists != nil {
		return nil, ErrGameNameExists
	}

	toCreate := &models.Game{}
	toCreate.ProtoUnMarshal(g)
	if err := s.gormDb.Create(toCreate).Error; err != nil {
		return nil, err
	}

	game, err := s.GetGameByName(ctx, g)
	if err != nil {
		return nil, err
	}
	return game, nil

}

func (s *Server) SetButtonPositions(ctx context.Context, g *pb.Game) (*pb.Game, error) {
	if g.GetName() == "" {
		return nil, ErrEmptyGameName
	}

	game, err := s.GetGame(ctx, g)
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, ErrGameDoesntExist
	}

	toUpdate := models.Game{
		// Randomly allocate a dealer
		Dealer: int64(rand.Intn(len(game.GetPlayers().GetPlayers()))) + 1,
	}

	if err := s.gormDb.Where("id = ?", game.GetId()).Find(game).Updates(toUpdate).Error; err != nil {
		return nil, err
	}

	out, err := s.GetGame(ctx, g)

	if err != nil {
		return nil, err
	}
	return out, nil

}

func (s *Server) SetMin(ctx context.Context, g *pb.Game) (*pb.Game, error) {
	if g.GetName() == "" {
		return nil, ErrEmptyGameName
	}

	game, err := s.GetGame(ctx, g)
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, ErrGameDoesntExist
	}

	toUpdate := models.Game{
		Min: g.GetMin(),
	}
	if err := s.gormDb.Where("id = ?", game.GetId()).Find(game).Updates(toUpdate).Error; err != nil {
		return nil, err
	}

	out, err := s.GetGame(ctx, g)

	if err != nil {
		return nil, err
	}
	return out, nil

}

func (s *Server) SetNextOnBet(ctx context.Context, in *pb.Round) (*pb.Round, error) {
	r, err := s.GetRound(ctx, in)
	if err != nil {
		return nil, err
	}

	g, err := s.GetGame(ctx, &pb.Game{Id: r.GetGame()})
	if err != nil {
		return nil, err
	}

	gr, err := game_ring.NewRing(g)
	if err != nil {
		return nil, err
	}

	// Go to next person on bet
	p, err := gr.GetPlayerFromSlot(&pb.Player{Slot: in.GetAction()})
	if err != nil {
		return nil, err
	}

	nextAction, err := gr.NextInHand(p)

	if err != nil {
		return nil, err
	}

	r.Action = nextAction.GetSlot()

	r, err = s.SetAction(ctx, r)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Server) NextDealer(ctx context.Context, g *pb.Game) (*pb.Game, error) {
	if g.GetName() == "" {
		return nil, ErrEmptyGameName
	}

	game, err := s.GetGame(ctx, g)
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, ErrGameDoesntExist
	}

	r, err := game_ring.NewRing(g)
	if err != nil {
		return nil, err
	}

	err = r.CurrentSmallBlind()
	if err != nil {
		return nil, err
	}
	// Set new dealer to the current small blind
	newDealer, err := r.MarshalValue()

	toUpdate := models.Game{
		Min: g.GetMin(),
		// Randomly allocate a dealer
		Dealer: newDealer.GetSlot(),
	}

	if err := s.gormDb.Where("id = ?", game.GetId()).Find(game).Updates(toUpdate).Error; err != nil {
		return nil, err
	}

	out, err := s.GetGame(ctx, g)

	if err != nil {
		return nil, err
	}
	return out, nil

}

// ValidatePreGame returns an error if the game is invalid
// Invalid reasons are
//  1. Not enough, or too many players
//  2. Slots are allocated to players incorrectly
//  3. Button positions and bet is not set.
func (s *Server) ValidatePreGame(ctx context.Context, g *pb.Game) (*pb.Game, error) {
	if g.InRound {
		return nil, ErrGameInRound
	}
	// mapping of user id to slot
	slotMap := map[int64]int64{}

	// mapping of slot to userId
	userMap := map[int64]int64{}

	// get a slice of all the player slots
	slotList := []int64{}

	for _, player := range g.GetPlayers().GetPlayers() {

		// Only seats 1-8 are valid
		if player.GetSlot() < 1 || player.GetSlot() > 8 {

			return g, ErrInvalidSlotNumber
		}
		// 2 players are allocated to the same slot
		if _, ok := slotMap[player.GetId()]; ok {

			return g, ErrInvalidSlotNumber
		}
		slotMap[player.GetId()] = player.GetSlot()
		userMap[player.GetSlot()] = player.GetId()

		slotList = append(slotList, player.GetSlot())
	}

	sort.Slice(slotList, func(i, j int) bool {
		return slotList[i] < slotList[j]
	})

	for i, v := range slotList {
		if !(i == 0) {
			prev := slotList[i-1]
			if !(prev < v) {
				//The slots are not sequential, or there is a gap

				return g, ErrInvalidSlotNumber
			}
		}
		if v == 0 {
			return g, ErrInvalidSlotNumber
		}
	}

	if g.GetDealer() == 0 {
		return g, ErrInvalidButtonAllocation
	}

	if g.GetMin() < 1 {
		return g, ErrNoBetSet
	}

	return g, nil
}

func (s *Server) RemovePlayerFromGame(ctx context.Context, player *pb.Player) (*empty.Empty, error) {

	gp := &models.GamePlayers{}
	game := &models.Game{}

	if err := s.gormDb.Where("player in (?)", player.GetId()).Find(gp).Error; err != nil && err != gorm.ErrRecordNotFound {
		return &empty.Empty{}, err
	} else if err != nil && err == gorm.ErrRecordNotFound {
		return &empty.Empty{}, ErrPlayerDoesntExist
	}

	if err := s.gormDb.Where("id in (?)", gp.Game).Find(game).Error; err != nil && err != gorm.ErrRecordNotFound {
		return &empty.Empty{}, err
	} else if err != nil && err == gorm.ErrRecordNotFound {
		return &empty.Empty{}, ErrGameDoesntExist
	}

	if game.InRound {
		return &empty.Empty{}, ErrGameInRound
	}

	if err := s.gormDb.Where("player in (?)", player.GetId()).Delete(&models.GamePlayers{}).Error; err != nil {
		return &empty.Empty{}, err
	}
	return &empty.Empty{}, nil
}

func (s *Server) GetRoundPlayersByRoundId(ctx context.Context, in *pb.Round) (*pb.Players, error) {
	gp := []*models.RoundPlayers{}

	if err := s.gormDb.Where("round = ?", in.GetId()).Find(&gp).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	} else if err != nil && err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	out := &pb.Players{}
	for _, pId := range gp {
		out.Players = append(out.Players, &pb.Player{
			Id: int64(pId.Player),
		})
	}
	// Get hydrated player instead of just their IDs
	players, err := s.GetPlayers(ctx, out)
	if err != nil {
		return nil, err
	}
	return players, nil
}

func (s *Server) UpdateGameInRound(ctx context.Context, g *pb.Game) (*pb.Game, error) {
	game, err := s.GetGame(ctx, g)
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, ErrGameDoesntExist
	}

	game.InRound = true

	toUpdate := &models.Game{}
	toUpdate.ProtoUnMarshal(game)

	if err := s.gormDb.Where("id = ?", game.GetId()).Find(&models.Game{}).Updates(toUpdate).Error; err != nil {
		return nil, err
	}

	out, err := s.GetGame(ctx, g)

	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Server) UpdateRoundStatus(ctx context.Context, r *pb.Round) (*pb.Round, error) {

	toUpdate := &models.Round{}
	toUpdate.ProtoUnMarshal(r)

	if err := s.gormDb.Where("id = ?", r.GetId()).Find(&models.Round{}).Updates(toUpdate).Error; err != nil {
		return nil, err
	}

	out, err := s.GetRound(ctx, r)

	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Server) UpdateGameStatus(ctx context.Context, g *pb.Game) (*pb.Game, error) {

	toUpdate := &models.Game{}
	toUpdate.ProtoUnMarshal(g)

	if err := s.gormDb.Where("id = ?", g.GetId()).Find(toUpdate).Update(
		"InRound", false).Error; err != nil {
		return nil, err
	}
	out, err := s.GetGame(ctx, g)

	if err != nil {
		return nil, err
	}
	return out, nil
}

// CreateRoundFromGame does the following ops:
// 1. validates the game is in a state where a new round could be created
// 2. creates an initial model of the round from the game
// 3. creates the round which generates a round ID
// 4. adds the game players to the round to the join table RoundPlayers
func (s *Server) CreateRoundFromGame(ctx context.Context, g *pb.Game) (*pb.Round, error) {
	game, err := s.GetGame(ctx, g)
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, ErrGameDoesntExist
	}

	game, err = s.ValidatePreGame(ctx, game)

	if err != nil {
		return nil, err
	}

	gModel := &models.Game{}
	gModel.ProtoUnMarshal(game)
	// HydratePlayers
	r := gModel.MarshalRound()

	if err := s.gormDb.Create(r).Error; err != nil {
		return nil, err
	}

	round, err := s.GetRound(ctx, r.ProtoMarshal())
	round.Players = g.GetPlayers()

	round, err = s.CreateRoundPlayers(ctx, round)
	if err != nil {
		return nil, err
	}
	return round, nil
}

func (s *Server) CreateRoundPlayers(ctx context.Context, r *pb.Round) (*pb.Round, error) {

	// Clear any existing players in the round
	// Ignore record not found errors
	// Ensures running CreateRoundPlayers is idempotent operations
	if err := s.gormDb.Where("id = (?)", r.GetId()).Delete(&models.RoundPlayers{}).Error; err != gorm.ErrRecordNotFound && err != nil {
		return nil, err
	}

	for _, shouldAdd := range r.GetPlayers().GetPlayers() {
		toCreate := &models.RoundPlayers{Player: shouldAdd.GetId(), Game: r.GetGame(), Round: r.GetId()}
		if err := s.gormDb.Create(toCreate).Error; err != nil {
			return nil, err
		}
	}

	round, err := s.GetRound(ctx, r)
	if err != nil {
		return nil, err
	}
	return round, nil
}

func (s *Server) ValidatePreRound(ctx context.Context, r *pb.Round) (*pb.Round, error) {

	round, err := s.GetRound(ctx, r)
	if err != nil {
		return nil, err
	}

	game, err := s.GetGame(ctx, &pb.Game{Id: r.Game})
	if err != nil {
		return nil, err
	}

	game, err = s.ValidatePreGame(ctx, game)
	if err != nil {
		return nil, err
	}

	if len(r.GetPlayers().GetPlayers()) != len(game.GetPlayers().GetPlayers()) {
		return nil, ErrInvalidPlayerCount
	}

	for _, p := range r.GetPlayers().GetPlayers() {
		if p.Chips < game.GetMin()*2 {
			return nil, ErrInsufficientChips
		}
	}

	d := deck.Deck{}
	d = d.Marshal(r.GetDeck())
	return round, nil
}

// StartRound is executed to start and setup the round
// Creates and deals a deck
// deducts small/big blind and sets on bet to small blind
func (s *Server) StartRound(ctx context.Context, r *pb.Round) (*pb.Round, error) {
	r, err := s.CreateDeck(ctx, r)
	if err != nil {
		return nil, err
	}

	r, err = s.DealCards(ctx, r)
	if err != nil {
		return nil, err
	}

	r.Status = pb.RoundStatus_PRE_FLOP

	r, err = s.UpdateRoundStatus(ctx, r)
	if err != nil {
		return nil, err
	}

	game, err := s.GetGame(ctx, &pb.Game{Id: r.GetGame()})

	if err != nil {
		return nil, err
	}
	game, err = s.UpdateGameInRound(ctx, game)
	if err != nil {
		return nil, err
	}

	game, err = s.AllocateGameSlots(ctx, game)
	if err != nil {
		return nil, err
	}
	ring, err := game_ring.NewRing(game)
	if err != nil {
		return nil, err
	}

	big, small, err := ring.GetBigAndSmallBlind()

	if err != nil {
		return nil, err
	}

	r.Status = pb.RoundStatus_PRE_FLOP
	// go to next position after big blind
	r.Action = small.GetSlot()

	r, err = s.SetAction(ctx, r)
	if err != nil {
		return nil, err
	}

	smallBet := &pb.Bet{
		Status: r.GetStatus(),
		Round:  r.GetId(),
		Game:   r.GetGame(),
		Player: small.GetId(),
		Chips:  game.GetMin(),
		Type:   pb.Bet_SMALL,
	}

	bigBet := &pb.Bet{
		Status: r.GetStatus(),
		Round:  r.GetId(),
		Game:   r.GetGame(),
		Player: big.GetId(),
		Chips:  game.GetMin() * 2,
		Type:   pb.Bet_BIG,
	}

	if _, err := s.MakeBet(ctx, smallBet); err != nil {

		return nil, err
	}
	if _, err := s.MakeBet(ctx, bigBet); err != nil {
		return nil, err
	}
	r, err = s.GetRound(ctx, r)
	if err != nil {
		return nil, err
	}
	r, err = s.UpdateRoundStatus(ctx, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (s *Server) DealFlop(ctx context.Context, r *pb.Round) (*pb.Round, error) {

	for _, p := range r.GetPlayers().GetPlayers() {
		if p.GetInHand() && p.GetCards() == "" {
			return nil, ErrNoExistingCards
		}
	}
	d := deck.Deck{}.Marshal(r.GetDeck())

	//burn one
	_, d = deck.DealCard(d)
	var c1, c2, c3 deck.Card
	c1, d = deck.DealCard(d)
	c2, d = deck.DealCard(d)
	c3, d = deck.DealCard(d)

	r.Flop = c1.String() + c2.String() + c3.String()
	r, err := s.UpdateRoundFlop(ctx, r)
	if err != nil {
		return nil, err
	}

	r.Deck = d.String()

	r, err = s.UpdateDeck(ctx, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (s *Server) DealRiver(ctx context.Context, r *pb.Round) (*pb.Round, error) {

	for _, p := range r.GetPlayers().GetPlayers() {
		if p.GetInHand() && p.GetCards() == "" {
			return nil, ErrNoExistingCards
		}
	}
	r, err := s.GetRound(ctx, r)
	if err != nil {
		return nil, err
	}
	d := deck.Deck{}.Marshal(r.GetDeck())

	//burn one
	_, d = deck.DealCard(d)
	var c1 deck.Card
	c1, d = deck.DealCard(d)
	r.River = c1.String()
	r, err = s.UpdateRoundRiver(ctx, r)
	if err != nil {
		return nil, err
	}

	r.Deck = d.String()

	r, err = s.UpdateDeck(ctx, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (s *Server) DealTurn(ctx context.Context, r *pb.Round) (*pb.Round, error) {

	for _, p := range r.GetPlayers().GetPlayers() {
		if p.GetInHand() && p.GetCards() == "" {
			return nil, ErrNoExistingCards
		}
	}
	r, err := s.GetRound(ctx, r)
	if err != nil {
		return nil, err
	}
	d := deck.Deck{}.Marshal(r.GetDeck())

	//burn one
	_, d = deck.DealCard(d)
	var c1 deck.Card
	c1, d = deck.DealCard(d)
	r.Turn = c1.String()
	r, err = s.UpdateRoundTurn(ctx, r)
	if err != nil {
		return nil, err
	}

	r.Deck = d.String()

	r, err = s.UpdateDeck(ctx, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (s *Server) DealCards(ctx context.Context, r *pb.Round) (*pb.Round, error) {

	for _, p := range r.GetPlayers().GetPlayers() {
		if p.GetCards() != "" {
			return nil, ErrExistingCards
		}
	}
	d := deck.Deck{}
	d = d.Marshal(r.GetDeck())
	if !d.IsFull() {
		return nil, ErrDeckNotFull
	}

	//burn one
	_, d = deck.DealCard(d)
	var c1, c2 deck.Card
	for _, p := range r.GetPlayers().GetPlayers() {
		c1, d = deck.DealCard(d)
		c2, d = deck.DealCard(d)
		p.Cards = c1.String() + c2.String()
	}

	_, err := s.UpdatePlayersCards(ctx, r.GetPlayers())
	if err != nil {
		return nil, err
	}

	r.Deck = d.String()

	r, err = s.UpdateDeck(ctx, r)
	return r, nil
}

func (s *Server) UpdateDeck(ctx context.Context, r *pb.Round) (*pb.Round, error) {
	round, err := s.GetRound(ctx, r)
	if err != nil {
		return nil, err
	}

	round.Deck = r.GetDeck()
	out := &models.Round{}
	if err := s.gormDb.Where("id = ?", round.GetId()).Find(out).Update(
		"deck", round.GetDeck()).Error; err != nil {
		return nil, err
	}
	round, err = s.GetRound(ctx, r)
	if err != nil {
		return nil, err
	}

	return round, nil
}

func (s *Server) SetAction(ctx context.Context, r *pb.Round) (*pb.Round, error) {
	round, err := s.GetRound(ctx, r)
	if err != nil {
		return nil, err
	}
	out := &models.Round{}
	if err := s.gormDb.Where("id = ?", round.GetId()).Find(out).Update(
		"action", r.GetAction()).Error; err != nil {
		return nil, err
	}
	round, err = s.GetRound(ctx, r)
	if err != nil {
		return nil, err
	}

	return round, nil
}
func (s *Server) CreateDeck(ctx context.Context, r *pb.Round) (*pb.Round, error) {
	round, err := s.GetRound(ctx, r)
	if err != nil {
		return nil, err
	}
	d := deck.New()
	d = deck.Shuffle(d)

	round.Deck = d.String()
	out := &models.Round{}
	if err := s.gormDb.Where("id = ?", round.GetId()).Find(out).Update(
		"deck", round.GetDeck()).Error; err != nil {
		return nil, err
	}
	round, err = s.GetRound(ctx, r)
	if err != nil {
		return nil, err
	}

	return round, nil
}

func (s *Server) UpdatePlayersCards(ctx context.Context, in *pb.Players) (*pb.Players, error) {

	for _, p := range in.GetPlayers() {
		out := &models.Player{}
		toUpdate := &models.Player{
			Cards:  p.GetCards(),
			InHand: true,
		}
		if err := s.gormDb.Where("id = ?", p.GetId()).Find(out).Updates(&toUpdate).Error; err != nil {
			return nil, err
		}
	}

	players, err := s.GetPlayers(ctx, in)
	if err != nil {
		return nil, err
	}

	return players, nil
}

func (s *Server) UpdateRoundFlop(ctx context.Context, in *pb.Round) (*pb.Round, error) {

	out := &models.Round{}
	if err := s.gormDb.Where("id = ?", in.GetId()).Find(out).Update(
		"Flop", in.GetFlop()).Error; err != nil {
		return nil, err
	}

	round, err := s.GetRound(ctx, in)
	if err != nil {
		return nil, err
	}
	return round, nil
}
func (s *Server) UpdateRoundRiver(ctx context.Context, in *pb.Round) (*pb.Round, error) {

	out := &models.Round{}
	if err := s.gormDb.Where("id = ?", in.GetId()).Find(out).Update(
		"River", in.GetRiver()).Error; err != nil {
		return nil, err
	}

	round, err := s.GetRound(ctx, in)
	if err != nil {
		return nil, err
	}
	return round, nil
}

func (s *Server) UpdateRoundTurn(ctx context.Context, in *pb.Round) (*pb.Round, error) {

	out := &models.Round{}
	if err := s.gormDb.Where("id = ?", in.GetId()).Find(out).Update(
		"Turn", in.GetTurn()).Error; err != nil {
		return nil, err
	}

	round, err := s.GetRound(ctx, in)
	if err != nil {
		return nil, err
	}
	return round, nil
}
func (s *Server) UpdatePlayerNotinHand(ctx context.Context, in *pb.Player) (*pb.Player, error) {

	out := &models.Player{}

	if err := s.gormDb.Where("id = ?", in.GetId()).Find(out).Update(
		"InHand", false).Error; err != nil {
		return nil, err
	}

	player, err := s.GetPlayer(ctx, in)
	if err != nil {
		return nil, err
	}

	return player, nil
}

func (s *Server) UpdatePlayersChips(ctx context.Context, in *pb.Players) (*pb.Players, error) {

	for _, p := range in.GetPlayers() {
		out := &models.Player{}
		if err := s.gormDb.Where("id = ?", p.GetId()).Find(out).Update(
			"chips", p.GetChips()).Error; err != nil {
			return nil, err
		}
	}
	players, err := s.GetPlayers(ctx, in)
	if err != nil {
		return nil, err
	}
	return players, nil
}

func (s *Server) MakeBet(ctx context.Context, in *pb.Bet) (*pb.Round, error) {

	// validate game exists
	game, err := s.GetGame(ctx, &pb.Game{Id: in.GetGame()})
	if err != nil {
		return nil, ErrGameDoesntExist
	}
	// Validate game state is live
	if game.GetInRound() != true {
		return nil, ErrGameIsNotInRound
	}
	// Get the round info needed to validate bet
	r, err := s.GetRound(ctx, &pb.Round{Id: in.GetRound()})
	if err != nil {
		return nil, err
	}

	// validate the round is in a state to accept bets
	if !statusIsValidForBet(r.GetStatus()) {
		return nil, ErrNoBetsAllowed
	} else if r.GetStatus() != in.GetStatus() {
		return nil, ErrWrongBetStatus
	}

	// validate player exists and the player's slot is the one that should be betting
	player, err := s.GetPlayer(ctx, &pb.Player{Id: in.GetPlayer()})
	if err != nil || player == nil {
		return nil, ErrPlayerDoesntExist
	}
	if r.GetAction() != player.GetSlot() {
		return nil, ErrPlayerNotOnAction
	}

	if !player.GetInHand() {
		return nil, ErrPlayerNotInHand
	}

	// Get the bets for the current round
	amtToCall, err := s.GetAmountToCallForPlayer(ctx, &pb.AmountToCall{
		Player: player,
		Round:  r,
	})
	tableMinBetRequired := amtToCall.GetChips()
	if err != nil {
		return nil, err
	}

	// validate bet type
	switch in.GetType() {
	case pb.Bet_FOLD:
		// Process fold and return if they are in action
		player, err = s.UpdatePlayerNotinHand(ctx, &pb.Player{Id: player.GetId()})
		if err != nil {
			return nil, err
		}

	case pb.Bet_CALL:
		if err := validateChips(
			player.GetChips(),
			in.GetChips(),
			tableMinBetRequired); err != nil {
			return nil, err
		}
		if in.GetChips() < tableMinBetRequired {
			return nil, ErrInsufficientBet
		}

		if in.GetChips() != tableMinBetRequired {
			return nil, ErrIncorrectBetForBetType
		}

	case pb.Bet_RAISE:
		if err := validateChips(
			player.GetChips(),
			in.GetChips(),
			tableMinBetRequired); err != nil {
			return nil, err
		}
		if in.GetChips() <= tableMinBetRequired {
			return nil, ErrWrongBetType
		}
	case pb.Bet_NONE:
		return nil, ErrNoBetTypeSet
	}

	// At this point we have validated
	//     - the game exists and is in round,
	//     - the round exists and is in round, and is in the correct status for a bet
	//     - The player exists and is the correct player on bet for the round
	//     - Player has sufficient chips, and has not bet this round
	//     - the bet type and amount of chips bet are valid (greater or equal to highest bet for that betting round/status)
	// Create bet since its validated

	toCreate := &models.Bet{}
	toCreate.ProtoUnMarshal(in)

	if err := s.gormDb.Create(toCreate).Error; err != nil {
		return nil, err
	}

	r, err = s.GetRound(ctx, r)
	if err != nil {
		return nil, err
	}

	if _, err := s.SetNextOnBet(ctx, r); err != nil {
		return nil, err
	}

	// Update the players chips
	player.Chips = player.Chips - in.GetChips()

	if _, err = s.UpdatePlayersChips(ctx,
		&pb.Players{
			Players: []*pb.Player{
				player,
			},
		},
	); err != nil {
		return nil, err
	}

	over, err := s.IsBettingOver(ctx, &pb.AmountToCall{
		Player: &pb.Player{Id: in.GetPlayer()},
		Round:  &pb.Round{Id: in.GetRound(), Game: in.GetGame()},
	})
	if err != nil {
		return nil, err
	}

	if over.GetBettingOver() {
		return s.SetNextRound(ctx, r)
	}

	return r, nil
}

func (s *Server) SetNextRound(ctx context.Context, in *pb.Round) (*pb.Round, error) {
	r, err := s.GetRound(ctx, &pb.Round{Id: in.GetId()})
	if err != nil {
		return nil, err
	}

	over, err := s.IsBettingOver(ctx, &pb.AmountToCall{
		Round: r,
	})
	if err != nil {
		return nil, err
	}

	if !over.GetBettingOver() {
		return nil, ErrIncompleteBets
	}

	rMap := map[pb.RoundStatus]pb.RoundStatus{
		//pb.RoundStatus_NOT_STARTED: pb.RoundStatus_PRE_FLOP,
		pb.RoundStatus_PRE_FLOP: pb.RoundStatus_FLOP,
		pb.RoundStatus_FLOP:     pb.RoundStatus_RIVER,
		pb.RoundStatus_RIVER:    pb.RoundStatus_TURN,
		pb.RoundStatus_TURN:     pb.RoundStatus_SHOW,
		pb.RoundStatus_SHOW:     pb.RoundStatus_OVER,
	}
	nextRound := rMap[r.GetStatus()]
	r.Status = nextRound
	r, err = s.UpdateRoundStatus(ctx, r)
	if err != nil {
		return nil, err
	}
	g, err := s.GetGame(ctx, &pb.Game{Id: r.GetGame()})
	if err != nil {
		return nil, err
	}
	ring, err := game_ring.NewRing(g)
	if err != nil {
		return nil, err
	}
	nextUp, err := ring.FirstOnBet()

	r.Action = nextUp.GetSlot()
	r, err = s.SetAction(ctx, r)
	if err != nil {
		return nil, err
	}

	// We ignore status RoundStatus_SHOW, since we don't need to deal any cards
	switch nextRound {
	case pb.RoundStatus_FLOP:
		r, err = s.DealFlop(ctx, r)
		if err != nil {
			return nil, err
		}
	case pb.RoundStatus_RIVER:
		r, err = s.DealRiver(ctx, r)
		if err != nil {
			return nil, err
		}
	case pb.RoundStatus_TURN:
		r, err = s.DealTurn(ctx, r)
		if err != nil {
			return nil, err
		}
	case pb.RoundStatus_OVER:
		// TODO: create function to call here to end game and evaluate hand
		r, err = s.EvaluateHands(ctx, r)

		if err != nil {
			return nil, err
		}
		return s.UpdateRoundWinner(ctx, r)

	}

	c := 0

	for _, p := range r.GetPlayers().GetPlayers() {
		if p.GetInHand() {
			c += 1
		}

	}

	if c == 1 {
		r, err = s.EvaluateHands(ctx, r)

		if err != nil {
			return nil, err
		}
		return s.UpdateRoundWinner(ctx, r)
	}

	r, err = s.GetRound(ctx, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func validateChips(bank, bet, min int64) error {
	if bet < min {
		return ErrInsufficientBet
	}

	if bank < bet {
		return ErrInsufficientChips
	}
	return nil
}

func (s *Server) GetRoundBets(ctx context.Context, in *pb.Round) (*pb.Bets, error) {
	var bets []*models.Bet

	if err := s.gormDb.Where("game = ? AND round = ?", in.GetGame(), in.GetId()).Find(&bets).Error; err != nil {
		return nil, err
	}
	outs := []*pb.Bet{}

	for _, b := range bets {
		outs = append(outs, b.ProtoMarshal())
	}
	return &pb.Bets{
		Bets: outs,
	}, nil
}

func (s *Server) GetRoundBetsForStatus(ctx context.Context, in *pb.Round) (*pb.Bets, error) {
	bets, err := s.GetRoundBets(ctx, in)
	if err != nil {
		return nil, err
	}

	outs := []*pb.Bet{}
	for _, b := range bets.GetBets() {

		if b.GetStatus() == in.GetStatus() {
			outs = append(outs, b)
		}
	}
	return &pb.Bets{
		Bets: outs,
	}, nil

}

// TODO: make this a query instead
func (s *Server) GetAmountToCallForPlayer(ctx context.Context, in *pb.AmountToCall) (*pb.AmountToCall, error) {
	bets, err := s.GetRoundBetsForStatus(ctx, in.GetRound())
	if err != nil {
		return nil, err
	}
	// make a map of each player and the bets they have made:
	m := map[int64]int64{}

	for _, i := range bets.GetBets() {
		m[i.GetPlayer()] = i.GetChips() + m[i.GetPlayer()]
	}

	// get player with biggest bet:
	bigBet := int64(0)
	for _, v := range m {
		if v > bigBet {
			bigBet = v
		}
	}

	// get player who is bettings, current bet
	playerBet := int64(0)
	if v, ok := m[in.GetPlayer().GetId()]; ok {
		playerBet = v
	}

	if bigBet > playerBet {
		in.Chips = bigBet - playerBet
		return in, nil
	}
	in.Chips = 0
	return in, nil
}

// TODO: use query from GetAMountTOCallFromPlayer (once query is implemented there)
// maybe TODO: refactor so the input is a pre-inflated value and we dont have to re-inflate.
// Perhaps a RoundState wrapper proto that can just be passed around?
func (s *Server) IsBettingOver(ctx context.Context, in *pb.AmountToCall) (*pb.AmountToCall, error) {
	r, err := s.GetRound(ctx, in.GetRound())
	if err != nil {
		return nil, err
	}

	bets, err := s.GetRoundBetsForStatus(ctx, r)
	if err != nil {
		return nil, err
	}
	g, err := s.GetGame(ctx, &pb.Game{Id: r.GetGame()})

	players, err := s.GetGamePlayersByGameId(ctx, g)
	if err != nil {
		return nil, err
	}

	activePlayers := 0

	for _, p := range players.GetPlayers() {
		if p.GetInHand() {
			activePlayers += 1
		}
	}

	liveBetMap := map[int64]int64{}

	for _, i := range bets.GetBets() {
		switch i.Type {
		case pb.Bet_CALL, pb.Bet_RAISE, pb.Bet_BIG, pb.Bet_SMALL:
			liveBetMap[i.GetPlayer()] += i.GetChips()
		}
	}

	// get player with biggest bet:
	bigBet := int64(0)
	uniqueBets := 0
	for _, v := range liveBetMap {
		if v > bigBet {
			bigBet = v
		}
		uniqueBets += 1
	}

	// If we don't have the min # of bets of active players than give up
	if activePlayers > uniqueBets {
		in.BettingOver = false
		return in, nil
	}

	for _, v := range liveBetMap {
		if v != bigBet {
			in.BettingOver = false
			return in, nil
		}
	}

	if activePlayers == uniqueBets {
		in.BettingOver = true
		return in, nil
	}

	return in, nil
}

func (s *Server) GetPlayerOnBet(ctx context.Context, in *pb.Round) (*pb.Player, error) {
	g, err := s.GetGame(ctx, &pb.Game{Id: in.GetGame()})
	if err != nil {
		return nil, ErrGameDoesntExist
	}

	gr, err := game_ring.NewRing(g)
	if err != nil {
		return nil, err
	}
	p, err := gr.GetPlayerFromSlot(&pb.Player{Slot: in.GetAction()})
	if err != nil {
		return nil, err
	}
	return p, nil

}

func (s *Server) UpdateRoundWinner(ctx context.Context, r *pb.Round) (*pb.Round, error) {

	g, err := s.GetGame(ctx, &pb.Game{Id: r.GetGame()})
	if err != nil {
		return nil, err
	}

	g.InRound = false

	g, err = s.UpdateGameStatus(ctx, g)
	if err != nil {
		return nil, err
	}

	return s.UpdateRoundStatus(ctx, r)

}

func (s *Server) EvaluateHands(ctx context.Context, round *pb.Round) (*pb.Round, error) {
	// expects an inflated round
	players := round.GetPlayers()
	if len(players.GetPlayers()) < 1 {
		return nil, ErrPlayerDoesntExist
	}
	pMap := map[int64]*pb.Player{}

	handsToRank := make(deck.PlayerHands, len(players.GetPlayers()))

	for _, player := range players.GetPlayers() {

		hand := deck.NewHand(player.GetCards() + round.GetFlop() + round.GetRiver() + round.GetTurn())
		score := hand.EvaluateHand()
		// set the score on response
		player.Score = score
		playerHand := deck.PlayerHand{
			Hand:     hand,
			Value:    score,
			PlayerId: player.GetId(),
		}
		handsToRank = append(handsToRank, playerHand)

		pMap[player.GetId()] = player
	}

	sort.Sort(handsToRank)

	out := pb.Players{}
	for _, hand := range handsToRank {
		if v, ok := pMap[hand.PlayerId]; ok {
			out.Players = append(out.Players, v)
		}
	}
	if len(out.GetPlayers()) < 1 {
		return nil, ErrPlayerDoesntExist
	}

	round.Players = &out

	winner := out.GetPlayers()[0]
	// set winner and set action to 0
	round.WinningPlayer = winner.GetId()
	round.WinningScore = winner.GetScore()
	round.WinningHand = winner.GetCards() + round.GetFlop() + round.GetRiver() + round.GetTurn()
	round.Action = 0

	return round, nil

}

func statusIsValidForBet(status pb.RoundStatus) bool {
	valid := map[pb.RoundStatus]bool{
		pb.RoundStatus_PRE_FLOP: true,
		pb.RoundStatus_FLOP:     true,
		pb.RoundStatus_RIVER:    true,
		pb.RoundStatus_TURN:     true,
		pb.RoundStatus_SHOW:     true,
	}
	if s := valid[status]; s {
		return true
	}
	return false
}
