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
	ErrEmptyPlayerName = fmt.Errorf("can not create player with empty name")
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

	statement, err := database.Prepare("CREATE TABLE IF NOT EXISTS Players (id INTEGER PRIMARY KEY, name TEXT, chips INTEGER)")
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
		Id:insertedId,
	})
	if err != nil {
		return nil, err
	}
	return player, nil

}


func (s *Server) CreatePlayers(ctx context.Context, players *pb.Players) (*pb.Players, error) {
	out :=  &pb.Players{}
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
	if err != nil {return nil, err}
	row  := statement.QueryRow(in.GetId())
	var id, chips int
	var name string
	switch err := row.Scan(&id, &name, &chips); err {
	case sql.ErrNoRows:
		return nil, nil
	case nil:
		return &pb.Player{
			Id: int64(id),
			Name: name,
			Chips:int64(chips),
		}, nil
	default:
		return nil, err
	}
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
