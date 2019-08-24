package server

import (
	"context"
	"fmt"
	"imran/poker/deck"
	"imran/poker/models"
	"log"
	"math/rand"
	"net"
	"sort"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"google.golang.org/grpc"
	pb "imran/poker/protobufs"
	"imran/poker/server/game_ring"
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
	ErrTooManyPlayers          = fmt.Errorf("too many players to create game")
	ErrInvalidSlotNumber       = fmt.Errorf("slot value invalid must be between 1-8")
	ErrInvalidSlotMinMax       = fmt.Errorf("slot value is greater than 8 or less than 1")
	ErrGameDoesntExist         = fmt.Errorf("no game found")
	ErrInvalidButtonAllocation = fmt.Errorf("buttons are not allocated correctly")
	ErrNoBetSet                = fmt.Errorf("no bet set for game")
	ErrPlayerDoesntExist       = fmt.Errorf("player doesn't exist")
	ErrGameInRound             = fmt.Errorf("can not perform operation when game is in round")
	ErrRoundInRound            = fmt.Errorf("can not perform operation when round is in round")
	ErrDeckNotFull             = fmt.Errorf("deck is not full")
	ErrExistingCards           = fmt.Errorf("player already has cards")
)

type Server struct {
	gormDb *gorm.DB
}

func NewServer(name string) (*Server, error) {
	s := &Server{}
	err := s.setupDatabase(name)
	return s, err
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

	if err := s.gormDb.Where("id = ?", game.GetId()).Find(game).Updates(toUpdate).Error; err != nil {
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

	fmt.Println("Marshaled round", r)

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
		fmt.Println(err)
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

// TODO write test
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

	if r.InRound == true {
		return nil, ErrRoundInRound
	}

	if len(r.GetPlayers().GetPlayers()) != len(game.GetPlayers().GetPlayers()) {
		return nil, ErrInvalidPlayerCount
	}

	d := deck.Deck{}
	d.Marshal(r.GetDeck())

	if !d.IsFull() {
		return nil, ErrDeckNotFull
	}
	return round, nil
}

func (s *Server) StartRound(ctx context.Context, r *pb.Round) (*pb.Round, error) {

	round, err := s.ValidatePreRound(ctx, r)
	if err != nil {
		return nil, err
	}
	// TODO define how a round starts
	// Deal cards, set on_bet

	return round, nil
}

func (s *Server) DealCards(ctx context.Context, r *pb.Round) (*pb.Round, error) {
	round, err := s.ValidatePreRound(ctx, r)
	if err != nil {
		return nil, err
	}

	for _, p := range round.GetPlayers().GetPlayers() {
		if p.GetCards() != "" {
			return nil, ErrExistingCards
		}
	}
	d := &deck.Deck{}
	d.Marshal(round.GetDeck())
	if !d.IsFull() {
		return nil, ErrDeckNotFull
	}

	//burn one
	_ = d.DealCard()
	for _, p := range r.GetPlayers().GetPlayers(){
		c1, c2 := d.DealCard(), d.DealCard()
		p.Cards = c1.String()+c2.String()



	}
	//TODO UPDATE PLAYER RECORd

	// TODO DEAL
	//burn := d.Deal

	r.Deck = d.String()
	return nil, nil

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
