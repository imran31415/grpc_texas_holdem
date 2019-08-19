package server

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	pb "imran/poker/protobufs"
	"log"
	"net"
)

const (
	Port   = ":50051"
	dbName = "poker"
)

var (
	ErrPlayerNameExists   = fmt.Errorf("Player with that name already exists")
	ErrEmptyPlayerName    = fmt.Errorf("can not create player with empty name")
	ErrInvalidPlayerCount = fmt.Errorf("can not create game with supplied count of players")
	ErrGameNameExists     = fmt.Errorf("game with that name already exists")
	ErrEmptyGameName      = fmt.Errorf("can not create game with empty name")
	ErrTooManyPlayers     = fmt.Errorf("too many players to create game")
)

type Server struct {
	db *sql.DB
}

func NewServer(name string) (*Server, error) {
	s := &Server{}
	err := s.setupDatabase(name)
	return s, err
}

func (s *Server) setupDatabase(name string) error {

	database, err := sql.Open("sqlite3", fmt.Sprintf("./%s.db", name))
	if err != nil {
		return err
	}

	// Setup Players table
	statement, err := database.Prepare("CREATE TABLE IF NOT EXISTS Players (id INTEGER PRIMARY KEY, name TEXT, chips INTEGER, h1 TEXT, h2 TEXT)")
	if err != nil {
		return err
	}

	_, err = statement.Exec()
	if err != nil {
		return err
	}

	// Setup Game Players table
	statement, err = database.Prepare("CREATE TABLE IF NOT EXISTS GamePlayers (id INTEGER PRIMARY KEY, player INTEGER, game INTEGER)")
	if err != nil {
		return err
	}

	_, err = statement.Exec()
	if err != nil {
		return err
	}

	statement, err = database.Prepare("CREATE TABLE IF NOT EXISTS Game (id INTEGER PRIMARY KEY,  name TEXT, dealer_slot INTEGER, big_slot INTEGER, small_slot INTEGER, small_amount INTEGER, f1 TEXT, f2 TEXT, f3 TEXT, f4 TEXT, f5 TEXT)")
	if err != nil {
		return err
	}

	_, err = statement.Exec()
	if err != nil {
		return err
	}

	s.db = database
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
	if exists != nil {
		return nil, ErrPlayerNameExists
	}

	statement, err := s.db.Prepare("INSERT INTO Players (name, chips) VALUES (?, ?)")
	if err != nil {
		return nil, err
	}
	result, err := statement.Exec(p.GetName(), p.GetChips())
	if err != nil {
		return nil, err
	}
	insertedId, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	player, err := s.GetPlayer(ctx, &pb.Player{
		Id: insertedId,
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
	statement, err := s.db.Prepare("SELECT id, name, chips FROM Players WHERE id=(?)")
	if err != nil {
		return nil, err
	}
	row := statement.QueryRow(in.GetId())
	var id, chips int
	var name string
	switch err := row.Scan(&id, &name, &chips); err {
	case sql.ErrNoRows:
		return nil, nil
	case nil:
		return &pb.Player{
			Id:    int64(id),
			Name:  name,
			Chips: int64(chips),
		}, nil
	default:
		return nil, err
	}
}

func (s *Server) GetPlayers(ctx context.Context, players *pb.Players) (*pb.Players, error) {
	out := &pb.Players{}
	for _, in := range players.GetPlayers() {
		p, err := s.GetPlayer(ctx, in)
		if err != nil {
			return nil, err
		}
		out.Players = append(out.GetPlayers(), p)
	}
	return out, nil
}

func (s *Server) GetPlayerByName(ctx context.Context, in *pb.Player) (*pb.Player, error) {
	statement, err := s.db.Prepare("SELECT id, name, chips FROM Players WHERE name=(?)")
	if err != nil {
		return nil, err
	}
	row := statement.QueryRow(in.GetName())
	var id, chips int
	var name string
	switch err := row.Scan(&id, &name, &chips); err {
	case sql.ErrNoRows:
		return nil, nil
	case nil:
		return &pb.Player{
			Id:    int64(id),
			Name:  name,
			Chips: int64(chips),
		}, nil
	default:
		return nil, err
	}
}

func (s *Server) GetPlayersByName(ctx context.Context, players *pb.Players) (*pb.Players, error) {

	out := &pb.Players{}
	// TODO switch this to be 1 query
	for _, in := range players.GetPlayers() {
		p, err := s.GetPlayerByName(ctx, in)
		if err != nil {
			return nil, err
		}
		out.Players = append(out.GetPlayers(), p)

	}
	return out, nil

}

func (s *Server) GetGame(ctx context.Context, in *pb.Game) (*pb.Game, error) {
	statement, err := s.db.Prepare("SELECT id, name FROM Game WHERE id=(?)")
	if err != nil {
		return nil, err
	}
	row := statement.QueryRow(in.GetId())
	var id int
	var name string
	switch err := row.Scan(&id, &name); err {
	case sql.ErrNoRows:
		return nil, nil
	case nil:
		return &pb.Game{
			Id:   int64(id),
			Name: name,
		}, nil
	default:
		return nil, err
	}
}

func (s *Server) GetGameByName(ctx context.Context, in *pb.Game) (*pb.Game, error) {
	statement, err := s.db.Prepare("SELECT id, name FROM Game WHERE name=(?)")
	if err != nil {
		return nil, err
	}
	row := statement.QueryRow(in.GetName())
	var id int
	var name string
	switch err := row.Scan(&id, &name); err {
	case sql.ErrNoRows:
		return nil, nil
	case nil:
		return &pb.Game{
			Id:   int64(id),
			Name: name,
		}, nil
	default:
		return nil, err
	}
}

func (s *Server) GetGamePlayersByGameId(ctx context.Context, in *pb.Game) (*pb.Players, error) {

	statement, err := s.db.Prepare("SELECT id, player, game FROM GamePlayers WHERE game=(?)")
	if err != nil {
		return nil, err
	}
	rows, err := statement.Query(in.GetId())
	if err != nil {
		return nil, err
	}
	out := []*pb.Player{}

	if err != nil {
		return nil, err
	}
	var id, player, game int
	for rows.Next() {
		err = rows.Scan(&id, &player, &game)
		if err != nil {
			return nil, err
		}
		out = append(out, &pb.Player{
			Id: int64(player),
		})

	}
	players := &pb.Players{
		Players: out,
	}
	return players, nil
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
	fmt.Println("Existing Players", existingPlayerRecords.GetPlayers())
	// 2.a create a map of existing playerIds to the player record
	existingPlayersMap := map[int64]*pb.Player{}
	for _, p := range existingPlayerRecords.GetPlayers() {
		existingPlayersMap[p.GetId()] = p
	}
	fmt.Println("Existing Players Map", existingPlayersMap)

	//3. Get the players requesting to be added to the game
	playersToJoinRecords, err := s.GetPlayersByName(ctx, g.GetPlayers())
	if err != nil {
		return nil, err
	}
	fmt.Println("Players to Join", playersToJoinRecords.GetPlayers())
	// 3.a create a map of requesting playerIds to whether they should join
	playersToJoinMap := map[int64]*pb.Player{}
	for _, p := range playersToJoinRecords.GetPlayers() {
		// Player is not already on the game list
		if _, ok := existingPlayersMap[p.GetId()]; !ok {
			playersToJoinMap[p.GetId()] = p
		}
	}
	fmt.Println("Players to join Map", playersToJoinMap)

	for _, shouldAdd := range playersToJoinMap {

		statement, err := s.db.Prepare("INSERT INTO GamePlayers (player, game) VALUES(?, ?)")
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
	fmt.Println("Resulting PLayers", players.GetPlayers())
	return players, err

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

	statement, err := s.db.Prepare("INSERT INTO Game (name) VALUES(?)")
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
