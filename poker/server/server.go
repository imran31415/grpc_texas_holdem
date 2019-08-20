package server

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jinzhu/gorm"
	"log"
	"net"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	pb "imran/poker/protobufs"
)

const (
	Port   = ":50051"
	dbName = "poker"
)

var (
	ErrPlayerNameExists   = fmt.Errorf("player with that name already exists")
	ErrEmptyPlayerName    = fmt.Errorf("can not create player with empty name")
	ErrInvalidPlayerCount = fmt.Errorf("can not create game with supplied count of players")
	ErrGameNameExists     = fmt.Errorf("game with that name already exists")
	ErrEmptyGameName      = fmt.Errorf("can not create game with empty name")
	ErrTooManyPlayers     = fmt.Errorf("too many players to create game")
)

type Server struct {
	db     *sql.DB
	gormDb *gorm.DB
}

type Player struct {
	gorm.Model
	Name  string
	Chips int64
	H1    string
	H2    string
}

type GamePlayers struct {
	gorm.Model
	Player int64
	Game   int64
}

type Game struct {
	gorm.Model
	Name   string
	Dealer int64
	Big    int64
	Small  int64
	MinBet int64
	f1     string
	f2     string
	f3     string
	f4     string
	f5     string
}

func NewServer(name string) (*Server, error) {
	s := &Server{}
	err := s.setupDatabase(name)
	return s, err
}

func (s *Server) setupDatabase(name string) error {

	database, err := sql.Open("sqlite3", fmt.Sprintf("./%s.db", name))
	db, err := gorm.Open("sqlite3", fmt.Sprintf("./%s.db", name))

	// Setup Players table
	db.CreateTable(&Player{})
	if err != nil {
		return err
	}

	// Setup DbGame Players table
	db.CreateTable(&GamePlayers{})
	if err != nil {
		return err
	}

	// Setup DbGame  table
	db.CreateTable(&Game{})
	if err != nil {
		return err
	}

	s.db = database
	s.gormDb = db
	return nil
}

func (s *Server) teardownTable(name string) error {
	st := fmt.Sprintf("DROP TABLE IF EXISTS %s", name)

	statement, err := s.db.Prepare(st)
	if err != nil {
		return err
	}

	_, err = statement.Exec(name)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) CreatePlayer(ctx context.Context, p *pb.Player) (*pb.Player, error) {
	if p.GetName() == "" {
		return nil, ErrEmptyPlayerName
	}

	exists, err := s.GetPlayerByName(ctx, p)

	if err != nil {
		return nil, err
	}

	if exists.GetId() != 0 {
		return nil, ErrPlayerNameExists
	}

	toCreate := &Player{Name: p.GetName(), Chips: p.GetChips()}
	if err := s.gormDb.Create(toCreate).Error; err != nil {
		return nil, err
	}

	player, err := s.GetPlayer(ctx, &pb.Player{
		Id: int64(toCreate.Model.ID),
	})
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

	p := &Player{
		Model: gorm.Model{
			ID: uint(in.GetId()),
		},
	}

	if err := s.gormDb.Where("id = ?", uint(in.GetId())).First(&p).Error; err != nil {
		return nil, err

	}
	return &pb.Player{
		Id:    int64(p.ID),
		Name:  p.Name,
		Chips: int64(p.Chips),
	}, nil
}

func (s *Server) GetPlayers(ctx context.Context, players *pb.Players) (*pb.Players, error) {
	outs := []Player{}

	ids := []int64{}

	for _, n := range players.GetPlayers() {
		ids = append(ids, n.GetId())
		outs = append(outs, Player{
			Model: gorm.Model{
				ID: uint(n.GetId()),
			},
		})
	}

	s.gormDb.Where("id IN (?)", ids).Find(&outs)
	out := &pb.Players{}

	// TODO switch this to be 1 query
	for _, inp := range outs {
		out.Players = append(out.Players, &pb.Player{
			Id:    int64(inp.ID),
			Name:  inp.Name,
			Chips: inp.Chips,
		})

	}
	return out, nil
}

func (s *Server) GetPlayerByName(ctx context.Context, in *pb.Player) (*pb.Player, error) {
	if in.Name == "" {
		return nil, ErrEmptyPlayerName
	}
	p := &Player{
		Name: in.GetName(),
	}

	s.gormDb.Where("name", []string{"jinzhu", "jinzhu 2"}).Find(&p)
	if err := s.gormDb.Where("name = ?", in.GetName()).Find(&p).Error; err != nil && err == gorm.ErrRecordNotFound {
		return &pb.Player{}, nil
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err

	} else {

		return &pb.Player{
			Id:    int64(p.ID),
			Name:  p.Name,
			Chips: int64(p.Chips),
		}, nil
	}
}

func (s *Server) GetPlayersByName(ctx context.Context, players *pb.Players) (*pb.Players, error) {

	outs := []Player{}

	names := []string{}

	for _, n := range players.GetPlayers() {
		names = append(names, n.GetName())
		outs = append(outs, Player{
			Name: n.GetName(),
		})
	}
	s.gormDb.Where("name IN (?)", names).Find(&outs)
	out := &pb.Players{}

	// TODO switch this to be 1 query
	for _, inp := range outs {
		out.Players = append(out.Players, &pb.Player{
			Id:    int64(inp.ID),
			Name:  inp.Name,
			Chips: inp.Chips,
		})

	}
	return out, nil

}

func (s *Server) GetGame(ctx context.Context, in *pb.Game) (*pb.Game, error) {
	g := &Game{
		Model: gorm.Model{
			ID: uint(in.GetId()),
		},
	}

	if err := s.gormDb.Where("id = (?)", in.GetId()).Find(g).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	} else if err != nil && err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &pb.Game{
		Id:   int64(g.ID),
		Name: g.Name,
	}, nil
}

func (s *Server) GetGameByName(ctx context.Context, in *pb.Game) (*pb.Game, error) {
	g := Game{
		Name: in.GetName(),
	}

	if err := s.gormDb.Where("name = ?", in.GetName()).Find(&g).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	} else if err != nil && err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &pb.Game{
		Id:   int64(g.ID),
		Name: g.Name,
	}, nil
}

func (s *Server) GetGamePlayersByGameId(ctx context.Context, in *pb.Game) (*pb.Players, error) {
	gp := []GamePlayers{
		{Game: in.GetId()},
	}

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

	return out, nil
}

// SetGamePlayers Sets the game players
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

	// 3.a create a map of requesting playerIds to whether they should join
	playersToJoinMap := map[int64]*pb.Player{}
	for _, p := range playersToJoinRecords.GetPlayers() {
		// Player is not already on the game list
		if _, ok := existingPlayersMap[p.GetId()]; !ok {
			playersToJoinMap[p.GetId()] = p
		}
	}

	for _, shouldAdd := range playersToJoinMap {

		statement, err := s.db.Prepare("INSERT INTO game_players (player, game) VALUES(?, ?)")
		if err != nil {
			return nil, err
		}
		result, err := statement.Exec(shouldAdd.GetId(), g.GetId())
		if err != nil {
			return nil, err
		}
		_, err = result.LastInsertId()
		if err != nil {
			return nil, err
		}

	}
	players, err := s.GetGamePlayersByGameId(ctx, g)
	if err != nil {
		return nil, err
	}
	return players, err

}

// SitGamePlayers allocates players to the game slots
func (s *Server) SitGamePlayers(ctx context.Context, g *pb.Game) (*pb.Game, error) {

	// 1. Get existing players IDs in the game
	existingIds, err := s.GetGamePlayersByGameId(ctx, g)

	if err != nil {
		return nil, err
	}

	// 2. Get Existing DbPlayer Records from the IDs
	existingPlayerRecords, err := s.GetPlayers(ctx, existingIds)
	if err != nil {
		return nil, err
	}

	if len(existingPlayerRecords.GetPlayers()) > 8 || len(existingPlayerRecords.GetPlayers()) < 2 {
		return nil, ErrInvalidPlayerCount
	}

	for i, p := range existingPlayerRecords.GetPlayers() {
		p.Slot = int64(i + 1)
	}

	return g, err

}

func (s *Server) SetPlayerSlot(ctx context.Context, p *pb.Player) (*pb.Player, error) {
	statement, err := s.db.Prepare("UPDATE Players SET slot=VALUE(?) WHERE id=VALUE(?)")
	if err != nil {
		return nil, err
	}
	result, err := statement.Exec(p.GetSlot(), p.GetId())
	if err != nil {
		return nil, err
	}
	_, err = result.LastInsertId()
	if err != nil {
		return nil, err
	}
	p, err = s.GetPlayer(ctx, p)
	if err != nil {
		return nil, err
	}
	return p, nil

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

	statement, err := s.db.Prepare("INSERT INTO Games (name) VALUES(?)")
	if err != nil {

		return nil, err
	}
	result, err := statement.Exec(g.GetName())
	if err != nil {
		return nil, err
	}
	insertedId, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	game, err := s.GetGame(ctx, &pb.Game{
		Id: insertedId,
	})
	if err != nil {
		return nil, err
	}
	return game, nil

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
